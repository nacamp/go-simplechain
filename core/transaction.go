package core

import (
	"crypto/ecdsa"
	"math/big"
	"time"

	// "github.com/ethereum/go-ethereum/crypto"
	// "github.com/ethereum/go-ethereum/crypto/sha3"

	// "github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/najimmy/go-simplechain/common"
	"github.com/najimmy/go-simplechain/crypto"
	"golang.org/x/crypto/sha3"
)

type Transaction struct {
	Hash   common.Hash
	From   common.Address
	To     common.Address
	Amount *big.Int
	// Nonce  uint64, next
	Time int64
	Sig  common.Sig // TODO: change name
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

func (tx *Transaction) Sign(prv *ecdsa.PrivateKey) {
	sign, _ := crypto.Sign(tx.Hash[:], prv)
	copy(tx.Sig[:], sign)
}

func (tx *Transaction) VerifySign() (bool, error) {
	pub, err := crypto.Ecrecover(tx.Hash[:], tx.Sig[:])
	if common.BytesToAddress(pub) == tx.From {
		return true, nil
	}
	return false, err
}

/*TODO
where privatekey , keystore
unique : nonce  or timestamp
if encodeing, how ?
*/
