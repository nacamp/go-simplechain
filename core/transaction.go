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

func (tx *Transaction) VerifySign() (bool, error) {
	pub, err := crypto.Ecrecover(tx.Hash[:], tx.Sig[:])
	if crypto.CreateAddressFromPublickeyByte(pub) == tx.From {
		return true, nil
	}
	return false, err
}

//FIXME: temporary Keystore
var Addr0 = string("0x036407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0")
var Addr1 = string("0x03e864b08b08f632c61c6727cde0e23d125f7784b5a5a188446fc5c91ffa51faa1")
var Addr2 = string("0x03fdefdefbb2478f3d1ed3221d38b8bad6d939e50f17ffda40f0510b4d28506bd3")
var Keystore = map[string]string{ //0, 2, 1
	Addr0: "0xe68fb0a479c495910c8351c3593667028b45d679f55ce22b0514c4a8a6bcbdd1",
	Addr2: "0xf390e256b6ed8a1b283d3ea80b103b868c14c31e5b7114fc32fff21c4cb263eb",
	Addr1: "0xb385aca81e134722cca902bf85443528c3d3a783cf54008cfc34a2ca563fc5b6",
}

func MakeTransaction(from, to string, amount *big.Int, nonce uint64) *Transaction {
	tx := NewTransaction(common.HexToAddress(from), common.HexToAddress(to), amount, nonce)
	tx.MakeHash()
	crypto.ByteToPrivatekey(common.FromHex(Keystore[from]))
	tx.Sign(crypto.ByteToPrivatekey(common.FromHex(Keystore[from])))
	return tx
}

func MakeTransactionPayload(from, to string, amount *big.Int, nonce uint64, payload []byte) *Transaction {
	tx := NewTransactionPayload(common.HexToAddress(from), common.HexToAddress(to), amount, nonce, payload)
	tx.MakeHash()
	crypto.ByteToPrivatekey(common.FromHex(Keystore[from]))
	tx.Sign(crypto.ByteToPrivatekey(common.FromHex(Keystore[from])))
	return tx
}

/*TODO
where privatekey , keystore
unique : nonce  or timestamp
if encodeing, how ?
*/
