package common

//first naive publickey
const (
	HashLength    = 32
	AddressLength = 66
)

// Address
type Address [AddressLength]byte

// Hash
type Hash [HashLength]byte
