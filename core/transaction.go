package core

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/rlp"
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

func (tx *Transaction) MakeHash() {
	hasher := sha3.New256()
	rlp.Encode(hasher, []interface{}{
		tx.From,
		tx.To,
		tx.Amount,
		tx.Time,
	})
	hasher.Sum(tx.Hash[:0])
}

/*TODO
where privatekey , keystore
unique : nonce  or timestamp
if encodeing, how ?
*/
