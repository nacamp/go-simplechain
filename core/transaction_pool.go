package core

type TransactionPool struct {
	queue []*Transaction
}

func NewTransactionPool() *TransactionPool {
	return &TransactionPool{}
}

func (pool *TransactionPool) Put(tx *Transaction) {
	pool.queue = append(pool.queue, tx)
}

func (pool *TransactionPool) Pop() (tx *Transaction) {
	if len(pool.queue) > 0 {
		tx = pool.queue[0]
		pool.queue = pool.queue[1:]
		return tx
	}
	return nil
}

func (pool *TransactionPool) Peek() (tx *Transaction) {
	if len(pool.queue) > 0 {
		tx = pool.queue[0]
		pool.queue = pool.queue[1:]
		return tx
	}
	return nil
}
