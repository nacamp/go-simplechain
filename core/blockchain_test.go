package core_test

import (
	"math/big"
	"testing"

	"github.com/najimmy/go-simplechain/common"
	"github.com/najimmy/go-simplechain/core"
	"github.com/najimmy/go-simplechain/storage"
	"github.com/stretchr/testify/assert"
)

func TestGenesisBlock(t *testing.T) {
	var coinbaseAddress = "036407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0"
	storage, _ := storage.NewMemoryStorage()
	block, err := core.GetGenesisBlock(storage)
	if err != nil {
	}
	assert.Equal(t, coinbaseAddress, common.Bytes2Hex(block.Header.Coinbase[:]), "")
}

func TestStorage(t *testing.T) {
	bc, _ := core.NewBlockChain()
	bc.PutBlock(bc.GenesisBlock)

	b1, _ := bc.GetBlockByHeight(0)
	assert.Equal(t, uint64(0), b1.Header.Height, "")
	assert.Equal(t, bc.GenesisBlock.Hash(), b1.Hash(), "")

	b2, _ := bc.GetBlockByHash(bc.GenesisBlock.Hash())
	assert.Equal(t, uint64(0), b2.Header.Height, "")
	assert.Equal(t, bc.GenesisBlock.Hash(), b2.Hash(), "")

	b3, _ := bc.GetBlockByHash(common.Hash{0x01})
	assert.Nil(t, b3, "")

	h := core.Header{}
	h.ParentHash = b1.Hash()
	block := core.Block{Header: &h}
	assert.Equal(t, true, bc.HasParentInBlockChain(&block), "")
	h.ParentHash.SetBytes([]byte{0x01})
	assert.Equal(t, false, bc.HasParentInBlockChain(&block), "")

}

func makeBlock(parentBlock *core.Block) *core.Block {
	// func makeBlock(bc *core.BlockChain, height uint64, parentHash common.Hash) *core.Block {
	h := &core.Header{}
	h.ParentHash = parentBlock.Hash()
	h.Height = parentBlock.Header.Height + 1
	h.Time = new(big.Int).SetUint64(1541112770 + h.Height)
	block := &core.Block{Header: h}

	var coinbaseAddress = "036407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0"
	account := core.Account{}
	copy(account.Address[:], common.Hex2Bytes(coinbaseAddress))
	account.AddBalance(new(big.Int).SetUint64(100 + 100*h.Height))
	block.AccountState, _ = parentBlock.AccountState.Clone()
	block.AccountState.PutAccount(&account)
	block.TransactionState, _ = parentBlock.TransactionState.Clone()
	block.TransactionState.PutTransaction(&core.Transaction{})

	h.AccountHash = block.AccountState.RootHash()
	h.TransactionHash = block.TransactionState.RootHash()
	// fmt.Printf("%v\n", h.AccountHash)
	// copy(h.TransactionHash[:], bc.TransactionState.Trie.RootHash())

	block.MakeHash()
	return block
}

func TestPutBlockIfParentExist(t *testing.T) {
	remoteBc, _ := core.NewBlockChain()
	block1 := makeBlock(remoteBc.GenesisBlock)
	// fmt.Printf("%v\n", remoteBc.GenesisBlock.Hash())
	block2 := makeBlock(block1)
	block3 := makeBlock(block2)
	block4 := makeBlock(block3)

	bc, _ := core.NewBlockChain()
	bc.PutBlock(bc.GenesisBlock)
	// fmt.Printf("%v\n", bc.GenesisBlock.Hash())

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
