package tests

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"

	"github.com/btcsuite/btcd/btcec"
	"github.com/nacamp/go-simplechain/cmd"
	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/core"
)

var Addr0 = string("0xc6d40a9bf9fe9d90019511a2147dc0958657da97463ca59d2594d5536dcdfd30ed93707d")
var Addr1 = string("0xd182458d4f299f73f496b7025912b0688653dbef74bc98638cd73e7e9ca01f8e9d416e44")
var Addr2 = string("0xfdf75c884f7f1d1537177a3a35e783236739a426ee649fa3e2d8aed598b4f29e838170e2")

var Keystore = map[string]string{ //0, 2, 1
	Addr0: "0x8a21cd44e684dd2d8d9205b0bfb69339435c7bd016ebc21fddaddffd0d47ed63",
	Addr1: "0xd7573bb27684e1911b5e8bfb3a553f860ce873562e64016fec0974a6163a5cff",
	Addr2: "0x47661aa6cccada84454842404ec0cca83760254191232f1d4cc11653d397ac2e",
}

type trick int

const (
	None trick = iota
	CHANGE_COINBASE
	LangObjC
)

func MakeConfig() *cmd.Config {
	configStr := `
	{
		"host_id" : "/ip4/127.0.0.1/tcp/9990/ipfs/16Uiu2HAkwR1pV8ZR8ApcZWrMSw5iNMwaJHFpKr91H9a1a65WGehk",
		"db_path" : "/test/db",
		"miner_address" : "0xd182458d4f299f73f496b7025912b0688653dbef74bc98638cd73e7e9ca01f8e9d416e44",
		"miner_private_key" : "0xd7573bb27684e1911b5e8bfb3a553f860ce873562e64016fec0974a6163a5cff",
		"node_private_key"  : "08021220a178bc3f8ee6738af0139d9784519e5aa1cb256c12c54444bd63296502f29e94",
		"node_key_path" : "/test/nodekey",
		"seeds" :  ["080212201afa45f64cd5a28cd40e178889ed2e9f987658bc4d48d376ef6ecb1ab1b26211"],
		
		"voters" : [{"address":"0xc6d40a9bf9fe9d90019511a2147dc0958657da97463ca59d2594d5536dcdfd30ed93707d", "balance":100 },
		{"address":"0xfdf75c884f7f1d1537177a3a35e783236739a426ee649fa3e2d8aed598b4f29e838170e2", "balance":20 },
		{"address":"0xd182458d4f299f73f496b7025912b0688653dbef74bc98638cd73e7e9ca01f8e9d416e44", "balance":50 }]
	}
	`
	contents := []byte(configStr)
	config := &cmd.Config{}
	json.Unmarshal([]byte(contents), config)
	return config
}

type signersAscending []common.Address

func (s signersAscending) Len() int           { return len(s) }
func (s signersAscending) Less(i, j int) bool { return bytes.Compare(s[i][:], s[j][:]) < 0 }
func (s signersAscending) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func SignerSlice(signers []common.Address) []common.Address {
	sigs := make([]common.Address, 0, len(signers))
	for _, sig := range signers {
		sigs = append(sigs, sig)
	}
	sort.Sort(signersAscending(sigs))
	return sigs
}

func MakeBlock(bc *core.BlockChain, parentBlock *core.Block, coinbase, from, to string, amount *big.Int, trickId trick, trickValue interface{}) *core.Block {
	h := &core.Header{}
	h.ParentHash = parentBlock.Hash()
	h.Height = parentBlock.Header.Height + 1
	h.Time = parentBlock.Header.Time + 3
	//set time to turn coinbase mining
	if bc.Consensus.ConsensusType() == "DPOS" {
		gMs, _ := bc.GenesisBlock.MinerState.Clone()
		gMinerGroup, _ := gMs.MakeMiner(bc.GenesisBlock.VoterState, 3)
		for {
			index := (h.Time % 9) / 3
			if gMinerGroup[index] == common.HexToAddress(coinbase) {
				break
			}
			h.Time++
		}
	} else {
		gMinerGroup := SignerSlice(bc.Signers)
		for {
			index := (h.Time % 9) / 3
			if gMinerGroup[index] == common.HexToAddress(coinbase) {
				break
			}
			h.Time++
		}
		// snapshot, err := dpos.snapshot(block.Header.ParentHash)
	}
	block := &core.Block{BaseBlock: core.BaseBlock{Header: h}}

	if bc.Consensus.ConsensusType() == "DPOS" {
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

	accs := block.AccountState
	txs := block.TransactionState

	fromAccount := accs.GetAccount(common.HexToAddress(from))
	tx := MakeTransaction(from, to, amount, fromAccount.Nonce+uint64(1))
	// tx := MakeTransaction(from, to, new(big.Int).Div(amount, new(big.Int).SetUint64(2)), fromAccount.Nonce+uint64(1))
	block.TransactionState.PutTransaction(tx)
	block.Transactions = make([]*core.Transaction, 1)
	block.Transactions[0] = tx
	fromAccount.Nonce += uint64(1)

	toAccount := accs.GetAccount(tx.To)
	fromAccount.SubBalance(tx.Amount)
	toAccount.AddBalance(tx.Amount)
	// fromAccount.SubBalance(tx.Amount)
	// toAccount.AddBalance(tx.Amount)

	accs.PutAccount(fromAccount)
	accs.PutAccount(toAccount)
	h.AccountHash = block.AccountState.RootHash()

	txs.PutTransaction(tx)
	h.TransactionHash = block.TransactionState.RootHash()

	if bc.Consensus.ConsensusType() == "DPOS" {
		h.MinerHash = block.MinerState.RootHash()
	}

	if bc.Consensus.ConsensusType() == "POA" {
		//TODO: fix temp hash
		block.Header.SnapshotHash = bc.GenesisBlock.Header.SnapshotHash
		// cannot use below code for cycling reference
		// 	snapshot, _ := bc.Consensus.(*consensus.Poa).Snapshot(block.Header.ParentHash)
	}
	block.MakeHash()
	priv, _ := btcec.PrivKeyFromBytes(btcec.S256(), common.FromHex(Keystore[coinbase]))
	block.Sign((*ecdsa.PrivateKey)(priv))
	return block
}

func MakeTransaction(from, to string, amount *big.Int, nonce uint64) *core.Transaction {
	tx := core.NewTransaction(common.HexToAddress(from), common.HexToAddress(to), amount, nonce)
	tx.MakeHash()
	priv, _ := btcec.PrivKeyFromBytes(btcec.S256(), common.FromHex(Keystore[from]))
	tx.Sign((*ecdsa.PrivateKey)(priv))
	return tx
}
