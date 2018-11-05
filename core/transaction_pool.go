package core

import (
	"github.com/najimmy/go-simplechain/common"
)

type TransactionPool struct {
	all map[common.Hash]*Transaction
}

func (pool *TransactionPool) Put(tx *Transaction) {
	//TODO: validate hash and sign before to put
	pool.all[tx.Hash] = tx
}

// use this when make block in consensus
func (pool *TransactionPool) Get(hash common.Hash) (tx *Transaction) {
	return pool.all[hash]
}

func (pool *TransactionPool) Del(hash common.Hash) {
	delete(pool.all, hash)
}
