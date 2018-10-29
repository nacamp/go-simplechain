package crypto

import (
	"golang.org/x/crypto/sha3"
)

//Sha3b256
func Sha3b256(args ...[]byte) []byte {
	hasher := sha3.New256()
	for _, bytes := range args {
		hasher.Write(bytes)
	}
	return hasher.Sum(nil)
}
