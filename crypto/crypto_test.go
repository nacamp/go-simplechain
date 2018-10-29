package crypto_test

import (
	"encoding/hex"
	"testing"

	"github.com/najimmy/go-simplechain/crypto"
	"github.com/stretchr/testify/assert"
)

func TestHash(t *testing.T) {
	assert.Equal(t, hex.EncodeToString(crypto.Sha3b256([]byte("testetesttesttest"))), "d30e2f276e0cfa51d5bef64753e82138c60e0e1deb1b27f3e39dc9aab4c4a2f3", "test sha3-256")
}
