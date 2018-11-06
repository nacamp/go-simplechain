package core

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/najimmy/go-simplechain/common"
	"github.com/najimmy/go-simplechain/rlp"
)

// Simple Header
type Header struct {
	ParentHash      common.Hash
	Coinbase        common.Address
	Height          uint64
	Time            *big.Int
	Hash            common.Hash
	AccountHash     common.Hash
	TransactionHash common.Hash
}

// Simple Block
type Block struct {
	Header       *Header
	Transactions []*Transaction

	AccountState     *AccountState
	TransactionState *TransactionState
}

func (b *Block) Hash() common.Hash {
	return b.Header.Hash
}

func (b *Block) MakeHash() {
	b.Header.Hash = b.CalcHash()
}

func (b *Block) CalcHash() (hash common.Hash) {
	hasher := sha3.New256()
	rlp.Encode(hasher, []interface{}{
		b.Header.ParentHash,
		b.Header.Coinbase,
		b.Header.Height,
		b.Header.Time,
		b.Header.AccountHash,
		b.Header.TransactionHash,
	})
	hasher.Sum(hash[:0])
	return hash
}

func (b *Block) VerifyTransacion() error {
	for _, tx := range b.Transactions {
		if tx.Hash != tx.CalcHash() {
			fmt.Println("tx.Hash != tx.CalcHash()")
			return errors.New("tx.Hash != tx.CalcHash()")
		}
		status, err := tx.VerifySign()
		if status != true || err != nil {
			fmt.Println("tx.VerifySign")
			return errors.New("tx.VerifySign")
		}
	}
	return nil
}
