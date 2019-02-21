package consensus

import (
	"bytes"
	"errors"
	"math/big"
	"math/rand"
	"sort"
	"time"

	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/core"
	"github.com/nacamp/go-simplechain/rlp"
	"github.com/nacamp/go-simplechain/trie"
)

type Miner struct {
	Candidate                map[common.Address]*big.Int
	MinerGroup               []common.Address
	SnapshotVoterHash        common.Hash
	SnapshotElectedTime      uint64
	SnapshotElectedBlockHash common.Hash
}

func (m *Miner) Stake(candidate common.Address, amount *big.Int) {
	v, ok := m.Candidate[candidate]
	if ok {
		m.Candidate[candidate] = amount
	} else {
		m.Candidate[candidate].Add(v, amount)
	}
}

func (m *Miner) Unstake(candidate common.Address, amount *big.Int) error {
	v, ok := m.Candidate[candidate]
	if ok {
		if v.Cmp(amount) < 0 {
			return errors.New("Staking is insufficient for candidate")
		}
		m.Candidate[candidate].Sub(v, amount)
	} else {
		return errors.New("Staking is insufficient for candidate")
	}
	return nil
}

func (m *Miner) ElectNewMiner(time uint64, maxMiner int) (bool, error) {
	if time < m.SnapshotElectedTime+3*3*3 { // 3round * 3miner * 3 duration for making block
		if len(m.Candidate) < maxMiner {
			return false, errors.New("The number of candidated miner is smaller than the minimum miner number.")
		}
		type kv struct {
			Key   common.Address
			Value *big.Int
		}
		var ss []kv
		for k, v := range m.Candidate {
			ss = append(ss, kv{k, v})
		}

		sort.Slice(ss, func(i, j int) bool {
			//.Cmp(accounts[j].Balance) > 0
			return ss[i].Value.Cmp(ss[j].Value) > 0
		})
		m.SnapshotElectedTime = time
		for i, v := range ss {
			if maxMiner == i {
				break
			}
			m.MinerGroup = append(m.MinerGroup, v.Key)
		}
		//shffle은 다른곳에서
		//shuffle2(m.MinerGroup, int64(m.SnapshotElectedTime))
		return true, nil
	}
	return false, nil
}

func shuffle2(slice []common.Address, seed int64) {
	r := rand.New(rand.NewSource(seed))
	for len(slice) > 0 {
		n := len(slice)
		randIndex := r.Intn(n)
		slice[n-1], slice[randIndex] = slice[randIndex], slice[n-1]
		slice = slice[:n-1]
	}
}

type MinerState struct {
	Trie *trie.Trie
}

func (ms *MinerState) Clone() (core.MinerState, error) {
	tr, err := ms.Trie.Clone()
	return &MinerState{
		Trie: tr,
	}, err
}

func (ms *MinerState) RootHash() (hash common.Hash) {
	copy(hash[:], ms.Trie.RootHash())
	return hash
}

func (ms *MinerState) Put2(miner *Miner) (hash common.Hash) {
	// miner := Miner{MinerGroup: minerGroup, SnapshotVoterHash: snapshotVoterHash}
	encodedBytes, _ := rlp.EncodeToBytes(miner)
	ms.Trie.Put(miner.SnapshotElectedBlockHash[:], encodedBytes)
	copy(hash[:], ms.Trie.RootHash())
	return hash
}

func (ms *MinerState) Get2(hash common.Hash) *Miner {
	// encodedBytes, _ := rlp.EncodeToBytes(snapshotVoterTime)
	miner := Miner{}
	decodedBytes, _ := ms.Trie.Get(hash[:])
	rlp.NewStream(bytes.NewReader(decodedBytes), 0).Decode(&miner)
	return &miner
}

//TODO: 문제점은 state는 개별정보를 저장후  RootHash를 block에 포함하는데 현재구조는 그렇지 않다. candidate를 별도로 저장해야 될 것 같다.
//TODO: prefix가 필요하다, account로 저장시

func (ms *MinerState) Put(minerGroup []common.Address, snapshotVoterHash common.Hash) (hash common.Hash) {
	miner := Miner{MinerGroup: minerGroup, SnapshotVoterHash: snapshotVoterHash}
	encodedBytes, _ := rlp.EncodeToBytes(miner)
	ms.Trie.Put(miner.SnapshotVoterHash[:], encodedBytes)
	copy(hash[:], ms.Trie.RootHash())
	return hash
}

func (ms *MinerState) Get(hash common.Hash) *Miner {
	// encodedBytes, _ := rlp.EncodeToBytes(snapshotVoterTime)
	miner := Miner{}
	decodedBytes, _ := ms.Trie.Get(hash[:])
	rlp.NewStream(bytes.NewReader(decodedBytes), 0).Decode(&miner)
	return &miner
}

func (ms *MinerState) GetMinerGroup(bc *core.BlockChain, block *core.Block) ([]common.Address, *core.Block, error) {
	if block.Header.Height == 0 {
		minerGroup, err := ms.MakeMiner(block.VoterState, 3)
		block.Header.SnapshotVoterTime = block.Header.Time
		if err != nil {
			return nil, nil, nil
		}
		return minerGroup, block, nil
	}
	//reuse miner group in SnapshotVoterTime
	if block.Header.Time < block.Header.SnapshotVoterTime+3*3*3 { // 3round * 3miner * 3 duration for making block
		for block.Header.Time != block.Header.SnapshotVoterTime {
			block = bc.GetBlockByHash(block.Header.ParentHash)
		}
		miner := ms.Get(block.Header.VoterHash)
		return miner.MinerGroup, block, nil

	}
	//make new miner group
	makeMiner, err := ms.MakeMiner(block.VoterState, 3)
	block.Header.SnapshotVoterTime = block.Header.Time
	if err != nil {
		return nil, nil, nil
	}
	return makeMiner, block, nil
}

func (ms *MinerState) MakeMiner(voterState *core.AccountState, maxMaker int) ([]common.Address, error) {

	accounts := make([]core.Account, 0)
	miners := make([]common.Address, 0)

	iter, err := voterState.Trie.Iterator(nil)
	if err != nil {
		return nil, err
	}
	exist, _ := iter.Next()
	for exist {
		account := core.Account{}
		decodedBytes := iter.Value()
		rlp.NewStream(bytes.NewReader(decodedBytes), 0).Decode(&account)
		accounts = append(accounts, account)
		exist, err = iter.Next()
	}

	if len(accounts) < maxMaker {
		return nil, errors.New("The number of candidated miner is smaller than the minimum miner number.")
	}

	sort.Slice(accounts, func(i, j int) bool {
		return accounts[i].Balance.Cmp(accounts[j].Balance) > 0
	})

	for i, v := range accounts {
		if maxMaker == i {
			break
		}
		miners = append(miners, v.Address)
	}
	shuffle(miners)
	return miners, nil
}

func shuffle(slice []common.Address) {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	for len(slice) > 0 {
		n := len(slice)
		randIndex := r.Intn(n)
		slice[n-1], slice[randIndex] = slice[randIndex], slice[n-1]
		slice = slice[:n-1]
	}
}
