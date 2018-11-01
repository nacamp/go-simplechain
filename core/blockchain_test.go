package core_test

import (
	"encoding/hex"
	"testing"

	// "github.com/najimmy/go-simplechain/core"

	"github.com/najimmy/go-simplechain/core"
	"github.com/najimmy/go-simplechain/crypto"
	"github.com/stretchr/testify/assert"
)

func TestStorage(t *testing.T) {
	h := core.Header{}
	//h.Height = new(big.Int).SetUint64(11)
	h.Height = 11
	h.ParentHash.SetBytes(crypto.Sha3b256([]byte("dummy test")))
	block := core.Block{&h}
	block.MakeHash()
	chain, err := core.NewBlockChain()
	if err != nil {
		return
	}
	chain.PutBlock(block)

	b1, err := chain.GetBlockByHeight(h.Height)
	assert.Equal(t, "6151d993d53d37941297e3f3e31a26a7cdc1d5fb3efc4a5a25132cdd38e05b15", hex.EncodeToString(b1.Header.ParentHash[:]), "")
	assert.Equal(t, uint64(11), b1.Header.Height, "")
	assert.Equal(t, block.Hash(), b1.Hash(), "")

	b2, err := chain.GetBlockByHash(block.Hash())
	assert.Equal(t, "6151d993d53d37941297e3f3e31a26a7cdc1d5fb3efc4a5a25132cdd38e05b15", hex.EncodeToString(b2.Header.ParentHash[:]), "")
	assert.Equal(t, uint64(11), b2.Header.Height, "")
	assert.Equal(t, block.Hash(), b2.Hash(), "")

}
