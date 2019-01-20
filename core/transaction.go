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

//FIXME: temporary Keystore
var Addr0 = string("0xc6d40a9bf9fe9d90019511a2147dc0958657da97463ca59d2594d5536dcdfd30ed93707d")
var Addr1 = string("0xd182458d4f299f73f496b7025912b0688653dbef74bc98638cd73e7e9ca01f8e9d416e44")
var Addr2 = string("0xfdf75c884f7f1d1537177a3a35e783236739a426ee649fa3e2d8aed598b4f29e838170e2")

var Keystore = map[string]string{ //0, 2, 1
	Addr0: "0x8a21cd44e684dd2d8d9205b0bfb69339435c7bd016ebc21fddaddffd0d47ed63",
	Addr1: "0xd7573bb27684e1911b5e8bfb3a553f860ce873562e64016fec0974a6163a5cff",
	Addr2: "0x47661aa6cccada84454842404ec0cca83760254191232f1d4cc11653d397ac2e",
}

func MakeTransaction(from, to string, amount *big.Int, nonce uint64) *Transaction {
	tx := NewTransaction(common.HexToAddress(from), common.HexToAddress(to), amount, nonce)
	tx.MakeHash()
	crypto.ByteToPrivateKey(common.FromHex(Keystore[from]))
	tx.Sign(crypto.ByteToPrivateKey(common.FromHex(Keystore[from])))
	return tx
}

func MakeTransactionPayload(from, to string, amount *big.Int, nonce uint64, payload []byte) *Transaction {
	tx := NewTransactionPayload(common.HexToAddress(from), common.HexToAddress(to), amount, nonce, payload)
	tx.MakeHash()
	crypto.ByteToPrivateKey(common.FromHex(Keystore[from]))
	tx.Sign(crypto.ByteToPrivateKey(common.FromHex(Keystore[from])))
	return tx
}