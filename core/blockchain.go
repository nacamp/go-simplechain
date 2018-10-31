package core

import (
	// "github.com/ethereum/go-ethereum/core/types"

	"github.com/najimmy/go-simplechain/common"
	"github.com/najimmy/go-simplechain/core"
	"github.com/najimmy/go-simplechain/storage"
)

/*
First time
search the height or the hash
add block
ignore block validity
*/
type BlockChain struct {
	storage storage.Storage
}

func NewBlockChain() (*BlockChain, error) {
	return nil, nil
}

func (bc *BlockChain) GetBlockByHash(hash common.Hash) *core.Block {
	return nil
}

func (bc *BlockChain) GetBlockByHeight(height uint64) *core.Block {
	return nil
}

func (bc *BlockChain) PutBlock(block *core.Block) {
	return nil
}
