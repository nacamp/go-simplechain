package crypto

import (
	"golang.org/x/crypto/scrypt"
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

func HashPassword(password string) ([]byte, error) {
	salt := []byte{0x12, 0x18, 0xff, 0x38, 0xe7, 0x9a, 0xda, 0x7c, 0x8c}
	return scrypt.Key([]byte(password), salt, 1<<15, 8, 1, 32)
}