package core

import (
	"sync"

	"github.com/nacamp/go-simplechain/common"
)

type TransactionPool struct {
	mu    sync.RWMutex
	queue []common.Hash
	txMap map[common.Hash]*Transaction
}

func NewTransactionPool() *TransactionPool {
	return &TransactionPool{txMap: make(map[common.Hash]*Transaction)}
}

func (pool *TransactionPool) Put(tx *Transaction) {
	pool.mu.Lock()
	pool.txMap[tx.Hash] = tx
	pool.queue = append(pool.queue, tx.Hash)
	pool.mu.Unlock()
}

func (pool *TransactionPool) Pop() (tx *Transaction) {
	pool.mu.RLock()
	defer pool.mu.RUnlock()
	for {
		if len(pool.queue) > 0 {
			hash := pool.queue[0]
			pool.queue = pool.queue[1:]
			tx = pool.txMap[hash]
			if tx != nil {
				return tx
			}
		} else {
			break
		}
	}

	return nil
}

func (pool *TransactionPool) Peek() (tx *Transaction) {
	pool.mu.RLock()
	defer pool.mu.RUnlock()
	if len(pool.queue) > 0 {
		hash := pool.queue[0]
		return pool.txMap[hash]
	}
	return nil
}
func (pool *TransactionPool) Get(hash common.Hash) (tx *Transaction) {
	return pool.txMap[hash]
}

func (pool *TransactionPool) Del(hash common.Hash) {
	delete(pool.txMap, hash)
}

func (pool *TransactionPool) Len() int {
	return len(pool.txMap)
}

func (pool *TransactionPool) FromTransactions(from common.Address) (txs []*Transaction) {
	txs = make([]*Transaction, 0)
	for _, v := range pool.txMap {
		if v.From == from {
			txs = append(txs, v)
		}
	}
	return txs
}

func (pool *TransactionPool) Refresh() {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	empty := make([]int, 0)
	for i, hash := range pool.queue {
		if _, ok := pool.txMap[hash]; ok == false {
			empty = append(empty, i)
		}
	}
	for i, j := range empty {
		pool.queue = append(pool.queue[:j-i], pool.queue[j+1-i:]...)
	}

}
