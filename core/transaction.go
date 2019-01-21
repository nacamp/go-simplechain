package core

import (
	"crypto/ecdsa"
	"math/big"
	"time"

	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/crypto"
	"github.com/nacamp/go-simplechain/rlp"
	"golang.org/x/crypto/sha3"
)

type Transaction struct {
	Hash    common.Hash
	From    common.Address
	To      common.Address
	Amount  *big.Int
	Nonce   uint64
	Time    uint64     // int64 rlp encoding error
	Sig     common.Sig // TODO: change name
	Payload []byte
}

func NewTransaction(from, to common.Address, amount *big.Int, nonce uint64) *Transaction {
	tx := &Transaction{
		From:   from,
		To:     to,
		Amount: amount,
		Time:   uint64(time.Now().Unix()),
		Nonce:  nonce,
	}
	return tx
}

func NewTransactionPayload(from, to common.Address, amount *big.Int, nonce uint64, payload []byte) *Transaction {
	tx := NewTransaction(from, to, amount, nonce)
	tx.Payload = payload
	return tx
}

func (tx *Transaction) MakeHash() {
	tx.Hash = tx.CalcHash()
}

func (tx *Transaction) CalcHash() (hash common.Hash) {
	hasher := sha3.New256()
	rlp.Encode(hasher, []interface{}{
		tx.From,
		tx.To,
		tx.Amount,
		tx.Nonce,
		tx.Payload,
	})
	hasher.Sum(hash[:0])
	return hash
}

func (tx *Transaction) Sign(prv *ecdsa.PrivateKey) {
	sign, _ := crypto.Sign(tx.Hash[:], prv)
	copy(tx.Sig[:], sign)
}

func (tx *Transaction) SignWithSignature(sign []byte) {
	copy(tx.Sig[:], sign)
}

func (tx *Transaction) VerifySign() (bool, error) {
	pub, err := crypto.Ecrecover(tx.Hash[:], tx.Sig[:])
	if crypto.CreateAddressFromPublicKeyByte(pub) == tx.From {
		return true, nil
	}
	return false, err
}
