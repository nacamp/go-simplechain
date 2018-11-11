package core_test

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/btcsuite/btcd/btcec"
	"github.com/najimmy/go-simplechain/consensus"

	"github.com/najimmy/go-simplechain/common"
	"github.com/najimmy/go-simplechain/core"
	"github.com/stretchr/testify/assert"
)

var GenesisCoinbaseAddress = string("0x036407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0")
var keystore = map[string]string{
	GenesisCoinbaseAddress: "0xe68fb0a479c495910c8351c3593667028b45d679f55ce22b0514c4a8a6bcbdd1",
	"0x03fdefdefbb2478f3d1ed3221d38b8bad6d939e50f17ffda40f0510b4d28506bd3": "0xf390e256b6ed8a1b283d3ea80b103b868c14c31e5b7114fc32fff21c4cb263eb",
	"0x03e864b08b08f632c61c6727cde0e23d125f7784b5a5a188446fc5c91ffa51faa1": "0xb385aca81e134722cca902bf85443528c3d3a783cf54008cfc34a2ca563fc5b6",
}

func TestGenesisBlock(t *testing.T) {
	dpos := consensus.NewDpos()
	bc, _ := core.NewBlockChain(dpos)

	assert.Equal(t, common.HexToAddress(GenesisCoinbaseAddress), bc.GenesisBlock.Header.Coinbase, "")
	assert.Equal(t, bc.GenesisBlock.Header.SnapshotVoterTime, uint64(0), "")

	//Test GetMinerGroup
	minerGroup, _ := bc.GenesisBlock.MinerState.GetMinerGroup(bc, bc.GenesisBlock)
	assert.Equal(t, common.HexToAddress(GenesisCoinbaseAddress), minerGroup[0], "")
	assert.Equal(t, common.HexToAddress("0x03fdefdefbb2478f3d1ed3221d38b8bad6d939e50f17ffda40f0510b4d28506bd3"), minerGroup[2], "")
	assert.Equal(t, common.HexToAddress("0x03e864b08b08f632c61c6727cde0e23d125f7784b5a5a188446fc5c91ffa51faa1"), minerGroup[1], "")
}

func TestStorage(t *testing.T) {
	dpos := consensus.NewDpos()
	bc, _ := core.NewBlockChain(dpos)
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

func makeBlock(parentBlock *core.Block, from, to string, amount *big.Int) *core.Block {
	// var coinbaseAddress = "0x036407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0"
	// func makeBlock(bc *core.BlockChain, height uint64, parentHash common.Hash) *core.Block {
	h := &core.Header{}
	h.ParentHash = parentBlock.Hash()
	h.Height = parentBlock.Header.Height + 1
	h.Time = 0 + h.Height //new(big.Int).SetUint64(1541112770 + h.Height)
	h.Coinbase = common.BytesToAddress(common.FromHex(GenesisCoinbaseAddress))
	block := &core.Block{Header: h}

	// coinbaseAccount := core.Account{}
	// copy(coinbaseAccount.Address[:], common.FromHex(GenesisCoinbaseAddress))
	//account.AddBalance(new(big.Int).SetUint64(100 + 100*h.Height))
	block.AccountState, _ = parentBlock.AccountState.Clone()
	coinbaseAccount := block.AccountState.GetAccount(common.HexToAddress(GenesisCoinbaseAddress))
	coinbaseAccount.AddBalance(new(big.Int).SetUint64(100))
	block.AccountState.PutAccount(coinbaseAccount)
	block.TransactionState, _ = parentBlock.TransactionState.Clone()
	// block.TransactionState.PutTransaction(&core.Transaction{})
	tx := makeTransaction(from, to, new(big.Int).Div(amount, new(big.Int).SetUint64(2)))
	//tx := makeTransaction(from, to, amount)
	//new(big.Int).Div(amount, new(big.Int).SetUint64(2))
	block.TransactionState.PutTransaction(tx)
	block.Transactions = make([]*core.Transaction, 2)
	block.Transactions[0] = tx
	block.Transactions[1] = tx

	accs := block.AccountState
	txs := block.TransactionState
	fromAccount := accs.GetAccount(tx.From)
	toAccount := accs.GetAccount(tx.To)
	fromAccount.SubBalance(tx.Amount)
	toAccount.AddBalance(tx.Amount)
	fromAccount.SubBalance(tx.Amount)
	toAccount.AddBalance(tx.Amount)

	accs.PutAccount(fromAccount)
	accs.PutAccount(toAccount)
	txs.PutTransaction(tx)

	h.AccountHash = block.AccountState.RootHash()
	h.TransactionHash = block.TransactionState.RootHash()

	// fmt.Printf("%v\n", h.AccountHash)
	// copy(h.TransactionHash[:], bc.TransactionState.Trie.RootHash())

	block.MakeHash()
	return block
}

func makeTransaction(from, to string, amount *big.Int) *core.Transaction {
	tx := core.NewTransaction(common.HexToAddress(from), common.HexToAddress(to), amount)
	tx.MakeHash()
	priv, _ := btcec.PrivKeyFromBytes(btcec.S256(), common.FromHex(keystore[from]))
	tx.Sign((*ecdsa.PrivateKey)(priv))
	return tx
}

func TestPutBlockIfParentExist(t *testing.T) {
	dpos := consensus.NewDpos()
	//balance genesis 100
	remoteBc, _ := core.NewBlockChain(dpos)
	//balance genesis 100, 1:100
	block1 := makeBlock(remoteBc.GenesisBlock, GenesisCoinbaseAddress, "0x03fdefdefbb2478f3d1ed3221d38b8bad6d939e50f17ffda40f0510b4d28506bd3", new(big.Int).SetUint64(100))
	// fmt.Printf("%v\n", remoteBc.GenesisBlock.Hash())
	//balance genesis 200, 1:90,   2:10
	block2 := makeBlock(block1, "0x03fdefdefbb2478f3d1ed3221d38b8bad6d939e50f17ffda40f0510b4d28506bd3", "0x03e864b08b08f632c61c6727cde0e23d125f7784b5a5a188446fc5c91ffa51faa1", new(big.Int).SetUint64(10))
	//balance genesis 300, 1:80,   2:20
	block3 := makeBlock(block2, "0x03fdefdefbb2478f3d1ed3221d38b8bad6d939e50f17ffda40f0510b4d28506bd3", "0x03e864b08b08f632c61c6727cde0e23d125f7784b5a5a188446fc5c91ffa51faa1", new(big.Int).SetUint64(10))
	//balance genesis 400, 1:70,   2:30
	block4 := makeBlock(block3, "0x03fdefdefbb2478f3d1ed3221d38b8bad6d939e50f17ffda40f0510b4d28506bd3", "0x03e864b08b08f632c61c6727cde0e23d125f7784b5a5a188446fc5c91ffa51faa1", new(big.Int).SetUint64(10))

	bc, _ := core.NewBlockChain(dpos)
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
