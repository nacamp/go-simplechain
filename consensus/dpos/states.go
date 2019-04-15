package dpos

import (
	"bytes"
	"math/big"
	"math/rand"
	"sort"
	"sync"

	"github.com/pkg/errors"

	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/core"
	"github.com/nacamp/go-simplechain/crypto"
	"github.com/nacamp/go-simplechain/log"
	"github.com/nacamp/go-simplechain/rlp"
	"github.com/nacamp/go-simplechain/storage"
	"github.com/nacamp/go-simplechain/trie"
	"github.com/sirupsen/logrus"
)

type Candidate struct {
	Address common.Address
	Balance *big.Int
}

var _stateShuffle func() //debugging

type DposState struct {
	mu          sync.RWMutex
	Candidate   *trie.Trie
	Miner       *trie.Trie
	Voter       *trie.Trie
	MinersHash  common.Hash
	ElectedTime uint64
}

func (cs *DposState) Stake(voter, candidate common.Address, amount *big.Int) (err error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if amount.Cmp(new(big.Int)) <= 0 {
		return errors.New("Stake amout must be greater than 0")
	}
	encodedBytes, err := cs.Candidate.Get(candidate[:])
	if err != nil {
		if err == trie.ErrNotFound {
			encodedBytes, err := rlp.EncodeToBytes(amount)
			if err != nil {
				return err
			}
			cs.Voter.Put(voter[:], []byte{})
			cs.Candidate.Put(candidate[:], encodedBytes)
			return nil
		}
		return err
	}
	balance := new(big.Int)
	err = rlp.Decode(bytes.NewReader(encodedBytes), balance)
	if err != nil {
		return err
	}
	balance.Add(balance, amount)
	encodedBytes, err = rlp.EncodeToBytes(balance)
	if err != nil {
		return err
	}
	cs.Voter.Put(voter[:], []byte{})
	cs.Candidate.Put(candidate[:], encodedBytes)
	return nil
}

/*
There is no record of who voted for the candidate, so un-voting users can unstack it
Before Unstake we must check staking  at Account in advance
*/
func (cs *DposState) Unstake(voter, candidate common.Address, amount *big.Int) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	encodedBytes, err := cs.Candidate.Get(candidate[:])
	if err != nil {
		if err == trie.ErrNotFound {
			return errors.New("Staking is insufficient for candidate")
		}
		return err
	}
	balance := new(big.Int)
	err = rlp.Decode(bytes.NewReader(encodedBytes), balance)
	if err != nil {
		return err
	}
	if balance.Cmp(amount) < 0 {
		return errors.New("Staking is insufficient for candidate")
	}
	balance.Sub(balance, amount)
	encodedBytes, err = rlp.EncodeToBytes(balance)
	if err != nil {
		return err
	}
	cs.Voter.Put(voter[:], []byte{})
	cs.Candidate.Put(candidate[:], encodedBytes)
	return nil
}

func GetNewElectedTime(parentElectedTime, now, cycle, round, totalMiners uint64) uint64 {
	if now >= parentElectedTime+(cycle*round*totalMiners) {
		return now
	}
	return parentElectedTime
}

func (ds *DposState) GetMiners(minerHash common.Hash) ([]common.Address, error) {
	miner := []common.Address{}
	decodedBytes, _ := ds.Miner.Get(minerHash[:])
	rlp.NewStream(bytes.NewReader(decodedBytes), 0).Decode(&miner)
	return miner, nil
}

func (ds *DposState) GetNewRoundMiners(electedTime uint64, totalMiners uint64) ([]common.Address, error) {
	iter, err := ds.Candidate.Iterator(nil)
	if err != nil {
		return nil, err
	}
	exist, _ := iter.Next()
	candidates := []core.BasicAccount{}
	for exist {
		account := core.BasicAccount{Address: common.Address{}}

		// encodedBytes1 := iter.Key()
		// key := new([]byte)
		// rlp.NewStream(bytes.NewReader(encodedBytes1), 0).Decode(key)
		account.Address = common.BytesToAddress(iter.Key())

		encodedBytes2 := iter.Value()
		value := new(big.Int)
		rlp.NewStream(bytes.NewReader(encodedBytes2), 0).Decode(value)
		account.Balance = value

		candidates = append(candidates, account)
		exist, err = iter.Next()
	}

	if len(candidates) < int(totalMiners) {
		return nil, errors.New("The number of candidated miner is smaller than the minimum miner number.")
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Balance.Cmp(candidates[j].Balance) > 0
	})

	candidates = candidates[:totalMiners]
	candidateAddrs := []common.Address{}
	for _, v := range candidates {
		candidateAddrs = append(candidateAddrs, v.Address)
	}
	if _stateShuffle == nil {
		randomShuffle(candidateAddrs, int64(electedTime))
	} else {
		_stateShuffle()
	}
	return candidateAddrs, nil
}

func (ds *DposState) PutMiners(miners []common.Address) (hash common.Hash, err error) {
	encodedBytes, err := rlp.EncodeToBytes(miners)
	if err != nil {
		return common.Hash{}, err
	}
	hashBytes := crypto.Sha3b256(encodedBytes)
	ds.Miner.Put(hashBytes, encodedBytes)
	copy(hash[:], hashBytes)
	return hash, nil
}

type StateHash struct {
	Candidate   []byte
	Voter       []byte
	Miner       []byte
	ElectedTime uint64
}

func (ds *DposState) Put(blockNumber, electedTime uint64, minersHash common.Hash) error {
	keyEncodedBytes, err := rlp.EncodeToBytes(blockNumber)
	if err != nil {
		return err
	}
	stateHash := new(StateHash)
	stateHash.ElectedTime = electedTime
	stateHash.Candidate = ds.Candidate.RootHash()
	stateHash.Voter = ds.Voter.RootHash()
	stateHash.Miner = minersHash[:]

	encodedStateHash, err := rlp.EncodeToBytes(stateHash)
	if err != nil {
		return err
	}
	_, err = ds.Miner.Put(crypto.Sha3b256(keyEncodedBytes), encodedStateHash)
	if err != nil {
		return err
	}

	return nil
}

/* return candidateHash, minersHash, electedTime*/
func (ds *DposState) Get(blockNumber uint64) (stateHash *StateHash, err error) {
	keyEncodedBytes, err := rlp.EncodeToBytes(blockNumber)
	if err != nil {
		return nil, err
	}
	encbytes, err := ds.Miner.Get(crypto.Sha3b256(keyEncodedBytes))
	if err != nil {
		return nil, err
	}

	stateHash = new(StateHash)
	err = rlp.Decode(bytes.NewReader(encbytes), stateHash)
	if err != nil {
		return nil, err
	}
	return stateHash, nil
}

func (ds *DposState) RootHash() (hash common.Hash) {
	copy(hash[:], ds.Miner.RootHash())
	return hash
}

func (ds *DposState) Clone() (core.ConsensusState, error) {
	tr1, err1 := ds.Candidate.Clone()
	if err1 != nil {
		return nil, err1
	}
	tr2, err2 := ds.Miner.Clone()
	if err2 != nil {
		return nil, err2
	}
	tr3, err3 := ds.Voter.Clone()
	if err3 != nil {
		return nil, err3
	}
	return &DposState{
		Candidate:   tr1,
		Miner:       tr2,
		Voter:       tr3,
		MinersHash:  ds.MinersHash,
		ElectedTime: ds.ElectedTime,
	}, nil
}

func (cs *DposState) ExecuteTransaction(block *core.Block, txIndex int, account *core.Account) (err error) {

	tx := block.Transactions[txIndex]
	amount := new(big.Int)
	err = rlp.Decode(bytes.NewReader(tx.Payload.Data), amount)
	if err != nil {
		return err
	}
	if tx.Payload.Code == core.TxCVoteStake {
		err = account.Stake(tx.To, amount)
		if err != nil {
			return err
		}
		return cs.Stake(account.Address, tx.To, amount)
	} else if tx.Payload.Code == core.TxCVoteUnStake {
		err = account.UnStake(tx.To, amount)
		if err != nil {
			return err
		}
		return cs.Unstake(account.Address, tx.To, amount)
	}
	return nil
}

/* Make new state by rootHash and initialized by blockNumber*/
func NewInitState(rootHash common.Hash, blockNumber uint64, storage storage.Storage) (*DposState, error) {
	var rootHashByte []byte
	if rootHash == (common.Hash{}) {
		rootHashByte = nil
	} else {
		rootHashByte = rootHash[:]
	}

	tr, err := trie.NewTrie(rootHashByte, storage, false)
	if err != nil {
		log.CLog().WithFields(logrus.Fields{"BlockNumber": blockNumber}).Panic(err)
		//return nil, err
	}

	state := new(DposState)
	state.Miner = tr
	stateHash, err := state.Get(blockNumber)
	if err != nil {
		if err == trie.ErrNotFound {
			tr2, err := trie.NewTrie(nil, storage, false)
			if err != nil {
				log.CLog().WithFields(logrus.Fields{}).Panic(err)
			}
			state.Candidate = tr2
			tr3, err := trie.NewTrie(nil, storage, false)
			if err != nil {
				log.CLog().WithFields(logrus.Fields{}).Panic(err)
			}
			state.Voter = tr3
			return state, nil
		}
		//return nil, err
		log.CLog().WithFields(logrus.Fields{}).Panic(err)
	}

	tr2, err := trie.NewTrie(stateHash.Candidate, storage, false)
	if err != nil {
		log.CLog().WithFields(logrus.Fields{}).Panic(err)
	}
	state.Candidate = tr2
	tr3, err := trie.NewTrie(stateHash.Voter, storage, false)
	if err != nil {
		log.CLog().WithFields(logrus.Fields{}).Panic(err)
	}
	state.Voter = tr3
	state.MinersHash = common.BytesToHash(stateHash.Miner)
	state.ElectedTime = stateHash.ElectedTime
	return state, nil
}

func randomShuffle(slice []common.Address, seed int64) {
	r := rand.New(rand.NewSource(seed))
	for len(slice) > 0 {
		n := len(slice)
		randIndex := r.Intn(n)
		slice[n-1], slice[randIndex] = slice[randIndex], slice[n-1]
		slice = slice[:n-1]
	}
}

func noneShuffle() {
}
