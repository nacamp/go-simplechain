package core

import (
	"errors"

	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/najimmy/go-simplechain/common"
	"github.com/najimmy/go-simplechain/rlp"
)

// Simple Header
type Header struct {
	ParentHash        common.Hash
	Coinbase          common.Address
	Height            uint64
	Time              uint64
	Hash              common.Hash
	AccountHash       common.Hash
	TransactionHash   common.Hash
	MinerHash         common.Hash
	VoterHash         common.Hash
	SnapshotVoterTime uint64
}

// Simple Block
type Block struct {
	Header       *Header
	Transactions []*Transaction

	AccountState     *AccountState
	TransactionState *TransactionState
	MinerState       MinerState
	VoterState       *AccountState
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
		b.Header.MinerHash,
	})
	hasher.Sum(hash[:0])
	return hash
}

func (b *Block) VerifyTransacion() error {
	for _, tx := range b.Transactions {
		if tx.Hash != tx.CalcHash() {
			return errors.New("tx.Hash != tx.CalcHash()")
		}
		status, err := tx.VerifySign()
		if status != true || err != nil {
			return errors.New("tx.VerifySign")
		}
	}
	return nil
}
