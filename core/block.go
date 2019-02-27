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
	ParentHash      common.Hash
	Coinbase        common.Address
	Height          uint64
	Time            uint64
	Hash            common.Hash
	AccountHash     common.Hash
	TransactionHash common.Hash
	ConsensusHash   common.Hash
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
	ConsensusState   ConsensusState
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
		b.Header.ConsensusHash,
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

func (b *Block) VerifySign() error {
	pub, err := crypto.Ecrecover(b.Header.Hash[:], b.Header.Signature[:])
	if err != nil {
		return err
	}
	if crypto.CreateAddressFromPublicKeyByte(pub) == b.Header.Coinbase {
		return nil
	}
	return errors.New("Public key cannot generate correct address") ////Signature is invalid
}

func (b *Block) VerifyTransacion() error {
	for _, tx := range b.Transactions {
		if tx.Hash != tx.CalcHash() {
			return errors.New("tx.Hash != tx.CalcHash()")
		}
		err := tx.VerifySign()
		if err != nil {
			return err
		}
	}
	return nil
}
