package consensus

import (
	"crypto/ecdsa"

	"github.com/najimmy/go-simplechain/common"
	"github.com/najimmy/go-simplechain/core"
	"github.com/najimmy/go-simplechain/log"
	"github.com/najimmy/go-simplechain/net"
	"github.com/najimmy/go-simplechain/storage"
	"github.com/najimmy/go-simplechain/trie"
	"github.com/sirupsen/logrus"
)

type Poa struct {
	bc           *core.BlockChain
	node         *net.Node
	coinbase     common.Address
	priv         *ecdsa.PrivateKey
	enableMining bool
}

//---------- Consensus
func (d *Poa) NewMinerState(rootHash common.Hash, storage storage.Storage) (core.MinerState, error) {
	tr, err := trie.NewTrie(common.HashToBytes(rootHash), storage, false)
	return &MinerState{
		Trie: tr,
	}, err
}

func (d *Poa) UpdateLIB(bc *core.BlockChain) {
	block := bc.Tail
	//FIXME: consider timestamp, changed minerGroup
	miners := make(map[common.Address]bool)
	turn := 1
	for bc.Lib.Hash() != block.Hash() {
		miners[block.Header.Coinbase] = true
		//minerGroup, _, _ := block.MinerState.GetMinerGroup(bc, block)
		if turn == 3 {
			if len(miners) == 3 {
				bc.SetLib(block)
				log.CLog().WithFields(logrus.Fields{
					"Height": block.Header.Height,
					//"address": common.Hash2Hex(block.Hash()),
				}).Info("Updated Lib")
				return
			}
			miners = make(map[common.Address]bool)
			miners[block.Header.Coinbase] = true
			turn = 0
		}
		block = bc.GetBlockByHash(block.Header.ParentHash)
		turn++
	}
	return
}
func (d *Poa) ConsensusType() string {
	return "POA"
}
