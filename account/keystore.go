package account

import (
	"crypto/ecdsa"

	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/crypto"
)

type Key struct {
	Address    common.Address
	PrivateKey *ecdsa.PrivateKey
}

type keyByte struct {
	Address    common.Address
	PrivateKey []byte
}

func NewKey() *Key {
	priv, address := crypto.CreateAddress()
	key := &Key{
		Address:    address,
		PrivateKey: priv,
	}
	return key
}

type keyStore interface {
	GetKey(addr common.Address, auth string) (*Key, error)
	StoreKey(k *Key, auth string) error
}
