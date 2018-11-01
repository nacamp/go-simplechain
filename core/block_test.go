package core_test

import (
	"encoding/hex"
	"testing"

	"github.com/najimmy/go-simplechain/core"
	"github.com/najimmy/go-simplechain/crypto"
	"github.com/stretchr/testify/assert"
)

func TestHash(t *testing.T) {
	h := core.Header{}
	h.ParentHash.SetBytes(crypto.Sha3b256([]byte("dummy test")))
	assert.Equal(t, "6151d993d53d37941297e3f3e31a26a7cdc1d5fb3efc4a5a25132cdd38e05b15", hex.EncodeToString(h.ParentHash[:]), "")
}
