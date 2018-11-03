package core

import (
	"math/big"
	"time"

	"github.com/najimmy/go-simplechain/common"
)

type Transaction struct {
	Hash   common.Hash
	From   common.Address
	To     common.Address
	Amount *big.Int
	// Nonce  uint64, next
	Time int64
	Sign common.Hash
}

func NewTransaction(from, to common.Address, amount *big.Int) *Transaction {
	tx := &Transaction{
		From:   from,
		To:     to,
		Amount: amount,
		Time:   time.Now().Unix(),
	}
	return tx
}

/*TODO
where privatekey , keystore
unique : nonce  or timestamp
if encodeing, how ?
*/
