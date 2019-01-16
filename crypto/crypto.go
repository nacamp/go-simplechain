// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package crypto

import (
	"crypto/ecdsa"
	"fmt"
	"reflect"

	"github.com/btcsuite/btcd/btcec"
	"github.com/nacamp/go-simplechain/common"
)

//ethereum address : ECDSA(secp256k1)=>(priv, pub), last 20byte from Keccak256(pub)
//Keccak256 ealry sha-3
//our address : ECDSA(secp256k1)=>(priv, pub), sha3-256(publickey) + checksum sha3-256(sha3-256(publickey))[0:4]
func CreateAddress() (*ecdsa.PrivateKey, common.Address) {
	priv := CreatePrivatekey()
	return priv, CreateAddressFromPrivatekey(priv)
}

func CreatePrivatekey() *ecdsa.PrivateKey {
	priv, _ := btcec.NewPrivateKey(btcec.S256())
	return (*ecdsa.PrivateKey)(priv)
}

func CreateAddressFromPrivatekey(priv *ecdsa.PrivateKey) common.Address {
	priv2 := (*btcec.PrivateKey)(priv)
	pub := priv2.PubKey().SerializeUncompressed()
	hash := Sha3b256(pub)
	hash = append(hash, Sha3b256(hash)[0:4]...)
	address := common.BytesToAddress(hash)
	//SerializeUncompressed
	return address
}

func CreateAddressFromPublickeyByte(pub []byte) common.Address {
	hash := Sha3b256(pub)
	hash = append(hash, Sha3b256(hash)[0:4]...)
	address := common.BytesToAddress(hash)
	return address
}

func ValidateAddress(address common.Address) bool {
	return reflect.DeepEqual(Sha3b256(address[0:32])[0:4], address[32:36])
}

// Ecrecover returns the uncompressed public key that created the given signature.
func Ecrecover(hash, sig []byte) ([]byte, error) {
	pub, err := SigToPub(hash, sig)
	if err != nil {
		return nil, err
	}
	bytes := (*btcec.PublicKey)(pub).SerializeUncompressed()
	return bytes, err
}

func SigToPub(hash, sig []byte) (*ecdsa.PublicKey, error) {
	pub, _, err := btcec.RecoverCompact(btcec.S256(), sig, hash)
	return (*ecdsa.PublicKey)(pub), err
}

func Sign(hash []byte, prv *ecdsa.PrivateKey) ([]byte, error) {
	if len(hash) != 32 {
		return nil, fmt.Errorf("hash is required to be exactly 32 bytes (%d)", len(hash))
	}
	if prv.Curve != btcec.S256() {
		return nil, fmt.Errorf("private key curve is not secp256k1")
	}
	sig, err := btcec.SignCompact(btcec.S256(), (*btcec.PrivateKey)(prv), hash, false)
	if err != nil {
		return nil, err
	}
	return sig, nil
}
