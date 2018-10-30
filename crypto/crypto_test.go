package crypto_test

import (
	"encoding/hex"
	"testing"

	"github.com/najimmy/go-simplechain/crypto"
	"github.com/stretchr/testify/assert"
)

func TestHash(t *testing.T) {
	assert.Equal(t, hex.EncodeToString(crypto.Sha3b256([]byte("dummy test"))), "6151d993d53d37941297e3f3e31a26a7cdc1d5fb3efc4a5a25132cdd38e05b15", "test sha3-256")
}
