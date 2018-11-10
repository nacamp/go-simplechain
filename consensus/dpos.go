package consensus

import (
	"github.com/najimmy/go-simplechain/common"
	"github.com/najimmy/go-simplechain/core"
	"github.com/najimmy/go-simplechain/storage"
	"github.com/najimmy/go-simplechain/trie"
)

type Dpos struct {
}

func NewDpos() *Dpos {
	return &Dpos{}
}

func (dpos *Dpos) MakeBlock() {

}

func (dpos *Dpos) Seal() {

}

//---------- Consensus
func (d *Dpos) NewMinerState(rootHash common.Hash, storage storage.Storage) (core.MinerState, error) {
	tr, err := trie.NewTrie(common.HashToBytes(rootHash), storage, false)
	return &MinerState{
		Trie: tr,
	}, err
}
