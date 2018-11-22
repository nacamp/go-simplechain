package tests

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/btcsuite/btcd/btcec"
	"github.com/najimmy/go-simplechain/common"
	"github.com/najimmy/go-simplechain/core"
)

var Addr0 = string("0x036407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0")
var Addr1 = string("0x03e864b08b08f632c61c6727cde0e23d125f7784b5a5a188446fc5c91ffa51faa1")
var Addr2 = string("0x03fdefdefbb2478f3d1ed3221d38b8bad6d939e50f17ffda40f0510b4d28506bd3")

var Keystore = map[string]string{ //0, 2, 1
	Addr0: "0xe68fb0a479c495910c8351c3593667028b45d679f55ce22b0514c4a8a6bcbdd1",
	Addr2: "0xf390e256b6ed8a1b283d3ea80b103b868c14c31e5b7114fc32fff21c4cb263eb",
	Addr1: "0xb385aca81e134722cca902bf85443528c3d3a783cf54008cfc34a2ca563fc5b6",
}

type trick int

const (
	None trick = iota
	CHANGE_COINBASE
	LangObjC
)

func MakeConfig() *core.Config {
	configStr := `
	{
		"host_id" : "/ip4/127.0.0.1/tcp/9990/ipfs/16Uiu2HAkwR1pV8ZR8ApcZWrMSw5iNMwaJHFpKr91H9a1a65WGehk",
		"db_path" : "/opt/simplechain/data",
		"miner_address" : "0x036407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0",
		"miner_private_key" : "0xe68fb0a479c495910c8351c3593667028b45d679f55ce22b0514c4a8a6bcbdd1",
		"node_private_key"  : "08021220a178bc3f8ee6738af0139d9784519e5aa1cb256c12c54444bd63296502f29e94",
		"seeds" :  ["080212201afa45f64cd5a28cd40e178889ed2e9f987658bc4d48d376ef6ecb1ab1b26211"],
		
		"voters" : [{"address":"0x036407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0", "balance":100 },
					{"address":"0x03fdefdefbb2478f3d1ed3221d38b8bad6d939e50f17ffda40f0510b4d28506bd3", "balance":20 },
					{"address":"0x03e864b08b08f632c61c6727cde0e23d125f7784b5a5a188446fc5c91ffa51faa1", "balance":50 }]
	}
	`
	contents := []byte(configStr)
	config := &core.Config{}
	json.Unmarshal([]byte(contents), config)
	return config
}
func MakeVoterAccountsFromConfig(config *core.Config) (voters []*core.Account) {
	voters = make([]*core.Account, 3)
	for i, voter := range config.Voters {
		account := &core.Account{}
		copy(account.Address[:], common.FromHex(voter.Address))
		account.Balance = voter.Balance
		voters[i] = account
	}
	return voters
}

func MakeBlock(bc *core.BlockChain, parentBlock *core.Block, coinbase, from, to string, amount *big.Int, trickId trick, trickValue interface{}) *core.Block {
	h := &core.Header{}
	h.ParentHash = parentBlock.Hash()
	h.Height = parentBlock.Header.Height + 1
	h.Time = parentBlock.Header.Time + 3
	//set time to turn coinbase mining
	gMs, _ := bc.GenesisBlock.MinerState.Clone()
	gMinerGroup, _ := gMs.MakeMiner(bc.GenesisBlock.VoterState, 3)
	for {
		index := (h.Time % 9) / 3
		if gMinerGroup[index] == common.HexToAddress(coinbase) {
			break
		}
		h.Time++
	}
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
	h.Coinbase = common.HexToAddress(coinbase)
	// index := block.Header.Height % 3
	// h.Coinbase = minerGroup[index]
	if trickId == CHANGE_COINBASE {
		h.Coinbase = common.HexToAddress((trickValue).(string))
	}
	// fmt.Printf("height,index,address : %v-%v-%v\n", block.Header.Height, index, common.Bytes2Hex(h.Coinbase[:]))

	//account, transaction
	block.AccountState, _ = parentBlock.AccountState.Clone()
	block.TransactionState, _ = parentBlock.TransactionState.Clone()
	coinbaseAccount := block.AccountState.GetAccount(h.Coinbase)
	coinbaseAccount.AddBalance(new(big.Int).SetUint64(100))
	block.AccountState.PutAccount(coinbaseAccount)
	tx := MakeTransaction(from, to, new(big.Int).Div(amount, new(big.Int).SetUint64(2)))
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

func MakeTransaction(from, to string, amount *big.Int) *core.Transaction {
	tx := core.NewTransaction(common.HexToAddress(from), common.HexToAddress(to), amount)
	tx.MakeHash()
	priv, _ := btcec.PrivKeyFromBytes(btcec.S256(), common.FromHex(Keystore[from]))
	tx.Sign((*ecdsa.PrivateKey)(priv))
	return tx
}
