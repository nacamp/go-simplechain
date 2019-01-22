package core

import (
	"crypto/ecdsa"
	"errors"

	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/crypto"
	"github.com/nacamp/go-simplechain/rlp"
	"golang.org/x/crypto/sha3"
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
	SnapshotHash      common.Hash
	SnapshotVoterTime uint64
	//not need signature at pow
	//need signature, to prevent malicious behavior like to skip deliberately block in the previous turn
	Signature common.Signature
}

// Simple Block
//TODO: refactor BaseBlock, Block
type BaseBlock struct {
	Header       *Header
	Transactions []*Transaction
}

type Block struct {
	BaseBlock

	AccountState     *AccountState
	TransactionState *TransactionState
	MinerState       MinerState
	VoterState       *AccountState
	Snapshot         interface{}
}

func (b *BaseBlock) NewBlock() *Block {
	return &Block{
		BaseBlock: *b,
	}
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
		b.Header.SnapshotHash,
	})
	hasher.Sum(hash[:0])
	return hash
}

func (b *Block) Sign(prv *ecdsa.PrivateKey) error {
	bytes, err := crypto.Sign(common.HashToBytes(b.Hash()), prv)
	if err != nil {
		return err
	}
	copy(b.Header.Signature[:], bytes)
	return nil
}

func (b *Block) SignWithSignature(sign []byte) {
	copy(b.Header.Signature[:], sign)
}

func (b *Block) VerifySign() (bool, error) {
	pub, err := crypto.Ecrecover(b.Header.Hash[:], b.Header.Signature[:])
	if crypto.CreateAddressFromPublicKeyByte(pub) == b.Header.Coinbase {
		return true, nil
	}
	return false, err
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
