package account

import (
	"crypto/ecdsa"

	// "github.com/ethereum/go-ethereum/crypto"
	// "github.com/ethereum/go-ethereum/crypto"

	"github.com/najimmy/go-simplechain/common"
	"github.com/najimmy/go-simplechain/crypto"
)

type Key struct {
	Address    common.Address
	PrivateKey *ecdsa.PrivateKey
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
