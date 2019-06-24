package core

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nacamp/go-simplechain/common"
)

func TestRefresh(t *testing.T) {
	from := common.HexToAddress("0xd182458d4f299f73f496b7025912b0688653dbef74bc98638cd73e7e9ca01f8e9d416e44")
	to := common.HexToAddress("0xfdf75c884f7f1d1537177a3a35e783236739a426ee649fa3e2d8aed598b4f29e838170e2")

	pool := NewTransactionPool()

	for i := 0; i < 10; i++ {
		tx := NewTransaction(from, to, new(big.Int).SetInt64(100), uint64(i))
		tx.MakeHash()
		// fmt.Println(common.HashToHex(tx.Hash))
		pool.Put(tx)
	}

	var tx *Transaction
	tx = NewTransaction(from, to, new(big.Int).SetInt64(100), uint64(0))
	tx.MakeHash()
	pool.Del(tx.Hash)

	tx = NewTransaction(from, to, new(big.Int).SetInt64(100), uint64(1))
	tx.MakeHash()
	pool.Del(tx.Hash)

	tx = NewTransaction(from, to, new(big.Int).SetInt64(100), uint64(2))
	tx.MakeHash()
	pool.Del(tx.Hash)

	tx = NewTransaction(from, to, new(big.Int).SetInt64(100), uint64(6))
	tx.MakeHash()
	pool.Del(tx.Hash)

	tx = NewTransaction(from, to, new(big.Int).SetInt64(100), uint64(8))
	tx.MakeHash()
	pool.Del(tx.Hash)

	tx = NewTransaction(from, to, new(big.Int).SetInt64(100), uint64(9))
	tx.MakeHash()
	pool.Del(tx.Hash)

	pool.Refresh()

	tx = NewTransaction(from, to, new(big.Int).SetInt64(100), uint64(3))
	tx.MakeHash()
	assert.Equal(t, tx.Hash, pool.queue[0])
	tx = NewTransaction(from, to, new(big.Int).SetInt64(100), uint64(4))
	tx.MakeHash()
	assert.Equal(t, tx.Hash, pool.queue[1])
	tx = NewTransaction(from, to, new(big.Int).SetInt64(100), uint64(5))
	tx.MakeHash()
	assert.Equal(t, tx.Hash, pool.queue[2])
	tx = NewTransaction(from, to, new(big.Int).SetInt64(100), uint64(7))
	tx.MakeHash()
	assert.Equal(t, tx.Hash, pool.queue[3])
}
