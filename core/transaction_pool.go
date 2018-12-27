package core

import (
	"sync"

	"github.com/najimmy/go-simplechain/common"
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

//TODO: remove hash in queue
func (pool *TransactionPool) Del(hash common.Hash) {
	delete(pool.txMap, hash)
}

//TODO: remove hash in queue
func (pool *TransactionPool) Len() int {
	return len(pool.txMap)
}
