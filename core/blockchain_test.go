package core_test

import (
	"encoding/hex"
	"math/big"
	"testing"

	// "github.com/najimmy/go-simplechain/core"
	"github.com/najimmy/go-simplechain/common"
	"github.com/najimmy/go-simplechain/core"
	"github.com/najimmy/go-simplechain/crypto"
	"github.com/stretchr/testify/assert"
)

func TestGenesisBlock(t *testing.T) {
	var coinbaseAddress = "036407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0"
	block, err := core.GetGenesisBlock()
	if err != nil {
	}
	assert.Equal(t, coinbaseAddress, common.Bytes2Hex(block.Header.Coinbase[:]), "")
}

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
	chain.PutBlock(&block)

	b1, err := chain.GetBlockByHeight(h.Height)
	assert.Equal(t, "6151d993d53d37941297e3f3e31a26a7cdc1d5fb3efc4a5a25132cdd38e05b15", hex.EncodeToString(b1.Header.ParentHash[:]), "")
	assert.Equal(t, uint64(11), b1.Header.Height, "")
	assert.Equal(t, block.Hash(), b1.Hash(), "")

	b2, err := chain.GetBlockByHash(block.Hash())
	assert.Equal(t, "6151d993d53d37941297e3f3e31a26a7cdc1d5fb3efc4a5a25132cdd38e05b15", hex.EncodeToString(b2.Header.ParentHash[:]), "")
	assert.Equal(t, uint64(11), b2.Header.Height, "")
	assert.Equal(t, block.Hash(), b2.Hash(), "")

	b3, err := chain.GetBlockByHash(common.Hash{0x01})
	assert.Nil(t, b3, "")

	h = core.Header{}
	h.ParentHash = b1.Hash()
	block = core.Block{&h}
	assert.Equal(t, true, chain.HasParentInBlockChain(&block), "")
	h.ParentHash.SetBytes([]byte{0x01})
	assert.Equal(t, false, chain.HasParentInBlockChain(&block), "")

}

func makeBlock(height uint64, parentHash common.Hash) *core.Block {
	//1
	h := &core.Header{}
	h.ParentHash = parentHash
	h.Time = new(big.Int).SetUint64(1541112770 + height)
	h.Height = height
	block := &core.Block{h}
	block.MakeHash()
	return block
}

func TestPutBlockIfParentExist(t *testing.T) {
	bc, _ := core.NewBlockChain()
	block, _ := core.GetGenesisBlock()
	parentHash := block.Hash()

	block1 := makeBlock(1, parentHash)
	block2 := makeBlock(2, block1.Hash())
	block3 := makeBlock(3, block2.Hash())
	block4 := makeBlock(4, block3.Hash())

	bc.PutBlockIfParentExist(block1)
	b, _ := bc.GetBlockByHash(block1.Hash())
	assert.Equal(t, block1.Hash(), b.Hash(), "")

	bc.PutBlockIfParentExist(block4)
	b, _ = bc.GetBlockByHash(block4.Hash())
	assert.Nil(t, b, "")

	bc.PutBlockIfParentExist(block3)
	b, _ = bc.GetBlockByHash(block3.Hash())
	assert.Nil(t, b, "")

	bc.PutBlockIfParentExist(block2)
	b, _ = bc.GetBlockByHash(block2.Hash())
	assert.NotNil(t, b, "")

	b, _ = bc.GetBlockByHash(block3.Hash())
	assert.NotNil(t, b, "")

	b, _ = bc.GetBlockByHash(block4.Hash())
	assert.NotNil(t, b, "")
}
