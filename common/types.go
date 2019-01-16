package common

import (
	"encoding/hex"
	"reflect"

	"github.com/nacamp/go-simplechain/common/hexutil"
)

//first naive publickey
const (
	HashLength    = 32
	AddressLength = 33
	SigLength     = 65
)

var (
	hashT    = reflect.TypeOf(Hash{})
	addressT = reflect.TypeOf(Address{})
)

// Address
type Address [AddressLength]byte

// Hash
type Hash [HashLength]byte

//Signature
type Sig [SigLength]byte

// BytesToHash sets b to hash.
// If b is larger than len(h), b will be cropped from the left.
func BytesToHash(b []byte) Hash {
	var h Hash
	h.SetBytes(b)
	return h
}

// SetBytes sets the hash to the value of b.
// If b is larger than len(h), b will be cropped from the left.
func (h *Hash) SetBytes(b []byte) {
	if len(b) > len(h) {
		b = b[len(b)-HashLength:]
	}

	copy(h[HashLength-len(b):], b)
}

func HexToHash(s string) Hash {
	return BytesToHash(FromHex(s))
}

func BytesToAddress(b []byte) Address {
	var a Address
	copy(a[0:], b)
	return a
}
func HexToAddress(s string) Address { return BytesToAddress(FromHex(s)) }

func HashToBytes(hash Hash) []byte {
	if hash == (Hash{}) {
		return nil
	} else {
		return hash[:]
	}

}

func Hash2Hex(hash Hash) string {
	return hex.EncodeToString(hash[:])
}

func Address2Hex(address Address) string {
	return hex.EncodeToString(address[:])
}

// MarshalText returns the hex representation of a.
func (a Address) MarshalText() ([]byte, error) {
	return hexutil.Bytes(a[:]).MarshalText()
}

// UnmarshalText parses a hash in hex syntax.
func (a *Address) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedText("Address", input, a[:])
}

// UnmarshalJSON parses a hash in hex syntax.
func (a *Address) UnmarshalJSON(input []byte) error {
	return hexutil.UnmarshalFixedJSON(addressT, input, a[:])
}
