package core

import "github.com/najimmy/go-simplechain/common"

type TransactionPool struct {
	queue []common.Hash
	txMap map[common.Hash]*Transaction
}

func NewTransactionPool() *TransactionPool {
	return &TransactionPool{txMap: make(map[common.Hash]*Transaction)}
}

func (pool *TransactionPool) Put(tx *Transaction) {
	pool.txMap[tx.Hash] = tx
	pool.queue = append(pool.queue, tx.Hash)
}

func (pool *TransactionPool) Pop() (tx *Transaction) {
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
