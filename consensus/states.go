package consensus

import (
	"bytes"
	"sort"

	"github.com/najimmy/go-simplechain/common"
	"github.com/najimmy/go-simplechain/core"
	"github.com/najimmy/go-simplechain/rlp"
	"github.com/najimmy/go-simplechain/trie"
)

type Miner struct {
	MinerGroup        []common.Address
	SnapshotVoterHash common.Hash
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
			block, _ = bc.GetBlockByHash(block.Header.ParentHash)
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

	// // var SnapshotVoterHash
	// // snapshotVoterTime := block.Header.SnapshotVoterTime
	// if block.Header.Time < block.Header.SnapshotVoterTime+3*3*3 { // 3round * 3miner * 3 duration for making block
	// 	//return ms.MakeMiner(block.VoterState, 3)
	// 	for block.Header.Time != block.Header.SnapshotVoterTime {
	// 		block, _ = bc.GetBlockByHash(block.Header.ParentHash)
	// 	}
	// 	miner := ms.Get(block.Header.VoterHash)
	// 	return miner.MinerGroup, block, nil

	// }

	// // block, _ = bc.GetBlockByHash(block.Header.ParentHash)
	// for block.Header.Time != block.Header.SnapshotVoterTime {
	// 	block, _ = bc.GetBlockByHash(block.Header.ParentHash)
	// }
	// miner := ms.Get(block.Header.VoterHash)
	// return miner.MinerGroup, block, nil

	// // // var SnapshotVoterHash
	// // snapshotVoterTime := block.Header.SnapshotVoterTime
	// // if block.Header.Time >= snapshotVoterTime+3*3*3 { // 3round * 3miner * 3 duration for making block
	// // 	return ms.MakeMiner(block.VoterState, 3)
	// // }

	// // // block, _ = bc.GetBlockByHash(block.Header.ParentHash)
	// // for block.Header.Time != block.Header.SnapshotVoterTime {
	// // 	block, _ = bc.GetBlockByHash(block.Header.ParentHash)
	// // }
	// // miner := ms.Get(block.Header.VoterHash)
	// // return miner.MinerGroup, block, nil
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

	sort.Slice(accounts, func(i, j int) bool {
		return accounts[i].Balance.Cmp(accounts[j].Balance) > 0
	})

	//TODO: if len(accouts) < maxMaker
	for i, v := range accounts {
		if maxMaker == i {
			break
		}
		miners = append(miners, v.Address)
	}
	//TODO: random sort for miners
	return miners, nil
}
