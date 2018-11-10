package consensus

import (
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

func (ms *MinerState) GetMinerGroup(block *core.Block) []common.Address {
	return nil
}
