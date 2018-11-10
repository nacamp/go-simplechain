package consensus

import (
	"bytes"

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

func (ms *MinerState) GetMinerGroup(bc *core.BlockChain, block *core.Block) []common.Address {
	if block.Header.Height == 0 {
		//new MinerGroup
		return nil
	}
	// var SnapshotVoterHash
	snapshotVoterTime := block.Header.SnapshotVoterTime
	if block.Header.Time >= snapshotVoterTime+3*3*3 { // 3round * 3miner * 3 duration for making block
		//new MinerGroup
		return nil
	}

	// block, _ = bc.GetBlockByHash(block.Header.ParentHash)
	for block.Header.Time != block.Header.SnapshotVoterTime {
		block, _ = bc.GetBlockByHash(block.Header.ParentHash)
	}
	miner := ms.Get(block.Header.VoterHash)
	return miner.MinerGroup
}
