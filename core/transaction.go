package core

import (
	"crypto/ecdsa"
	"errors"
	"math/big"
	"time"

	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/crypto"
	"github.com/nacamp/go-simplechain/rlp"
	"golang.org/x/crypto/sha3"
)

type Transaction struct {
	Hash      common.Hash
	From      common.Address
	To        common.Address
	Amount    *big.Int
	Nonce     uint64
	Time      uint64 // int64 rlp encoding error
	Signature common.Signature
	Payload   *Payload
}

const (
	TxCVoteUnStake = uint64(0x00)
	TxCVoteStake   = uint64(0x01)
)

type Payload struct {
	Code uint64
	Data []byte
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

func NewTransactionPayload(from, to common.Address, amount *big.Int, nonce uint64, payload *Payload) *Transaction {
	tx := NewTransaction(from, to, amount, nonce)
	tx.Payload = payload
	return tx
}

// func NewTransactionPayload(from, to common.Address, amount *big.Int, nonce uint64, payload []byte) *Transaction {
// 	tx := NewTransaction(from, to, amount, nonce)
// 	tx.Payload = payload
// 	return tx
// }

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
	copy(tx.Signature[:], sign)
}

func (tx *Transaction) SignWithSignature(sign []byte) {
	copy(tx.Signature[:], sign)
}

func (tx *Transaction) VerifySign() error {
	pub, err := crypto.Ecrecover(tx.Hash[:], tx.Signature[:])
	if err != nil {
		return err
	}
	if crypto.CreateAddressFromPublicKeyByte(pub) == tx.From {
		return nil
	}
	return errors.New("Public key cannot generate correct address") //Signature is invalid
}
