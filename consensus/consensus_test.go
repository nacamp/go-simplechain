package consensus_test

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/btcsuite/btcd/btcec"
	"github.com/najimmy/go-simplechain/common"
	"github.com/najimmy/go-simplechain/consensus"
	"github.com/najimmy/go-simplechain/core"
)

// import (
// 	"crypto/ecdsa"
// 	"fmt"
// 	"math/big"
// 	"testing"

// 	"github.com/btcsuite/btcd/btcec"
// 	"github.com/najimmy/go-simplechain/consensus"
// 	"github.com/stretchr/testify/assert"

// 	"github.com/najimmy/go-simplechain/common"
// 	"github.com/najimmy/go-simplechain/core"
// )

//TODO: make common test blockchain
var GenesisCoinbaseAddress = string("0x036407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0")
var keystore = map[string]string{ //0, 2, 1
	GenesisCoinbaseAddress: "0xe68fb0a479c495910c8351c3593667028b45d679f55ce22b0514c4a8a6bcbdd1",
	"0x03fdefdefbb2478f3d1ed3221d38b8bad6d939e50f17ffda40f0510b4d28506bd3": "0xf390e256b6ed8a1b283d3ea80b103b868c14c31e5b7114fc32fff21c4cb263eb",
	"0x03e864b08b08f632c61c6727cde0e23d125f7784b5a5a188446fc5c91ffa51faa1": "0xb385aca81e134722cca902bf85443528c3d3a783cf54008cfc34a2ca563fc5b6",
}

type trick int

const (
	None trick = iota
	CHANGE_COINBASE
	LangObjC
)

func makeBlock(bc *core.BlockChain, parentBlock *core.Block, from, to string, amount *big.Int, trickId trick, trickValue interface{}) *core.Block {
	h := &core.Header{}
	h.ParentHash = parentBlock.Hash()
	h.Height = parentBlock.Header.Height + 1
	h.Time = parentBlock.Header.Time + 3
	block := &core.Block{Header: h}

	//voter
	block.VoterState, _ = parentBlock.VoterState.Clone()
	h.VoterHash = block.VoterState.RootHash()

	//miner
	block.MinerState, _ = parentBlock.MinerState.Clone()
	minerGroup, voterBlock, _ := block.MinerState.GetMinerGroup(bc, block)
	//TODO: we need to test  when voter transaction make
	if voterBlock.Header.Height == block.Header.Height {
		block.MinerState.Put(minerGroup, block.Header.VoterHash)
		fmt.Printf("VoterHash(put), height, time, >>>%v, %v, %v\n", block.Header.Height, block.Header.Time, block.Header.VoterHash)
	} else {
		fmt.Printf("VoterHash(   ), height, time, >>>%v, %v, %v\n", block.Header.Height, block.Header.Time, block.Header.VoterHash)
	}
	index := block.Header.Height % 3
	h.Coinbase = minerGroup[index]
	if trickId == CHANGE_COINBASE {
		h.Coinbase = common.HexToAddress((trickValue).(string))
	}
	fmt.Printf("height,index,address : %v-%v-%v\n", block.Header.Height, index, common.Bytes2Hex(h.Coinbase[:]))

	//account, transaction
	block.AccountState, _ = parentBlock.AccountState.Clone()
	block.TransactionState, _ = parentBlock.TransactionState.Clone()
	coinbaseAccount := block.AccountState.GetAccount(h.Coinbase)
	coinbaseAccount.AddBalance(new(big.Int).SetUint64(100))
	block.AccountState.PutAccount(coinbaseAccount)
	tx := makeTransaction(from, to, new(big.Int).Div(amount, new(big.Int).SetUint64(2)))
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
	h.AccountHash = block.AccountState.RootHash()

	txs.PutTransaction(tx)
	h.TransactionHash = block.TransactionState.RootHash()

	h.MinerHash = block.MinerState.RootHash()

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

func TestMakeBlockChain(t *testing.T) {
	dpos := consensus.NewDpos()
	remoteBc, _ := core.NewBlockChain(dpos)
	remoteBc.PutBlockByCoinbase(remoteBc.GenesisBlock)
	block1 := makeBlock(remoteBc, remoteBc.GenesisBlock, GenesisCoinbaseAddress, "0x03fdefdefbb2478f3d1ed3221d38b8bad6d939e50f17ffda40f0510b4d28506bd3", new(big.Int).SetUint64(100), None, nil)
	remoteBc.PutBlockByCoinbase(block1)
	block2 := makeBlock(remoteBc, block1, "0x03fdefdefbb2478f3d1ed3221d38b8bad6d939e50f17ffda40f0510b4d28506bd3", "0x03e864b08b08f632c61c6727cde0e23d125f7784b5a5a188446fc5c91ffa51faa1", new(big.Int).SetUint64(10), None, nil)
	remoteBc.PutBlockByCoinbase(block2)
	block3 := makeBlock(remoteBc, block2, "0x03fdefdefbb2478f3d1ed3221d38b8bad6d939e50f17ffda40f0510b4d28506bd3", "0x03e864b08b08f632c61c6727cde0e23d125f7784b5a5a188446fc5c91ffa51faa1", new(big.Int).SetUint64(10), None, nil)
	remoteBc.PutBlockByCoinbase(block3)
	block4 := makeBlock(remoteBc, block3, "0x03fdefdefbb2478f3d1ed3221d38b8bad6d939e50f17ffda40f0510b4d28506bd3", "0x03e864b08b08f632c61c6727cde0e23d125f7784b5a5a188446fc5c91ffa51faa1", new(big.Int).SetUint64(10), None, nil)
	remoteBc.PutBlockByCoinbase(block4)
	dpos.UpdateLIB(remoteBc)
	assert.Equal(t, uint64(2), remoteBc.Lib.Header.Height, "")

	//change coinbase address
	block5 := makeBlock(remoteBc, block4, "0x03fdefdefbb2478f3d1ed3221d38b8bad6d939e50f17ffda40f0510b4d28506bd3", "0x03e864b08b08f632c61c6727cde0e23d125f7784b5a5a188446fc5c91ffa51faa1", new(big.Int).SetUint64(10), CHANGE_COINBASE, GenesisCoinbaseAddress)
	remoteBc.PutBlockByCoinbase(block5)
	dpos.UpdateLIB(remoteBc)
	assert.Equal(t, uint64(2), remoteBc.Lib.Header.Height, "")

	block6 := makeBlock(remoteBc, block5, GenesisCoinbaseAddress, "0x03e864b08b08f632c61c6727cde0e23d125f7784b5a5a188446fc5c91ffa51faa1", new(big.Int).SetUint64(2), None, nil)
	remoteBc.PutBlockByCoinbase(block6)
	block7 := makeBlock(remoteBc, block6, GenesisCoinbaseAddress, "0x03e864b08b08f632c61c6727cde0e23d125f7784b5a5a188446fc5c91ffa51faa1", new(big.Int).SetUint64(2), None, nil)
	remoteBc.PutBlockByCoinbase(block7)
	dpos.UpdateLIB(remoteBc)
	assert.Equal(t, uint64(2), remoteBc.Lib.Header.Height, "")

	block8 := makeBlock(remoteBc, block7, GenesisCoinbaseAddress, "0x03e864b08b08f632c61c6727cde0e23d125f7784b5a5a188446fc5c91ffa51faa1", new(big.Int).SetUint64(2), None, nil)
	remoteBc.PutBlockByCoinbase(block8)
	dpos.UpdateLIB(remoteBc)
	assert.Equal(t, uint64(6), remoteBc.Lib.Header.Height, "")

	block9 := makeBlock(remoteBc, block8, GenesisCoinbaseAddress, "0x03e864b08b08f632c61c6727cde0e23d125f7784b5a5a188446fc5c91ffa51faa1", new(big.Int).SetUint64(2), None, nil)
	remoteBc.PutBlockByCoinbase(block9)
	dpos.UpdateLIB(remoteBc)
	assert.Equal(t, uint64(7), remoteBc.Lib.Header.Height, "")
}

func TestDpos_MakeBlock(t *testing.T) {
	dpos := consensus.NewDpos()
	remoteBc, _ := core.NewBlockChain(dpos)
	remoteBc.PutBlockByCoinbase(remoteBc.GenesisBlock)

	dpos.Setup(remoteBc, nil, common.HexToAddress(GenesisCoinbaseAddress))
	block := dpos.MakeBlock(uint64(3)) //0
	assert.NotNil(t, block, "")
	assert.NotEqual(t, block.Header.AccountHash, remoteBc.GenesisBlock.Header.AccountHash, "")
	assert.Equal(t, block.Header.VoterHash, remoteBc.GenesisBlock.Header.VoterHash, "")
	assert.Equal(t, block.Header.MinerHash, remoteBc.GenesisBlock.Header.MinerHash, "")
	assert.Equal(t, block.Header.TransactionHash, remoteBc.GenesisBlock.Header.TransactionHash, "")
}

func TestDpos_MakeBlock2(t *testing.T) {
	dpos := consensus.NewDpos()
	remoteBc, _ := core.NewBlockChain(dpos)
	remoteBc.PutBlockByCoinbase(remoteBc.GenesisBlock)

	dpos.Setup(remoteBc, nil, common.HexToAddress(GenesisCoinbaseAddress))
	block := dpos.MakeBlock(uint64(3 * 3 * 3)) //0
	assert.NotNil(t, block, "")
	assert.NotEqual(t, block.Header.AccountHash, remoteBc.GenesisBlock.Header.AccountHash, "")
	assert.NotEqual(t, block.Header.MinerHash, remoteBc.GenesisBlock.Header.MinerHash, "")
	assert.Equal(t, block.Header.VoterHash, remoteBc.GenesisBlock.Header.VoterHash, "")
	assert.Equal(t, block.Header.TransactionHash, remoteBc.GenesisBlock.Header.TransactionHash, "")
}
