package core

import (
	"github.com/najimmy/go-simplechain/common"
	"github.com/najimmy/go-simplechain/storage"
)

type State interface {
	// NewState() (interface{}, error)
	Clone() (interface{}, error)
	RootHash() (hash common.Hash)
	Put(interface{}) (hash common.Hash)
	Get(interface{}) interface{}
}

/*
type Miner struct {
	// Timestamp  uint64
	Address    common.Address
	nonce      uint64
	MinerGroup []common.Address
	VoterHash  common.Hash
}
*/
type MinerState interface {
	State
	// GetTrie() *trie.Trie
	GetMinerGroup() []common.Address
	// GetNonce() uint64
}

type Consensus interface {
	NewMinerState(rootHash common.Hash, storage storage.Storage) (MinerState, error)
}
