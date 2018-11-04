package core

import (
	"github.com/najimmy/go-simplechain/common"
)

type TransactionPool struct {
	all map[common.Hash]*Transaction
}

func (pool *TransactionPool) Put(tx *Transaction) {
	pool.all[tx.Hash] = tx
}

func (pool *TransactionPool) Get(hash common.Hash) (tx *Transaction) {
	return pool.all[hash]
}

func (pool *TransactionPool) Del(hash common.Hash) {
	delete(pool.all, hash)
}
