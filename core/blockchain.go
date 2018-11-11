package core

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"math/big"

	lru "github.com/hashicorp/golang-lru"
	"github.com/najimmy/go-simplechain/common"
	"github.com/najimmy/go-simplechain/rlp"
	"github.com/najimmy/go-simplechain/storage"
)

/*
First time
search the height or the hash
add block
ignore block validity
*/
const (
	maxFutureBlocks = 256
)

var GenesisCoinbaseAddress = string("0x036407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0")
var keystore = map[string]string{
	GenesisCoinbaseAddress: "0xe68fb0a479c495910c8351c3593667028b45d679f55ce22b0514c4a8a6bcbdd1",
	"0x03fdefdefbb2478f3d1ed3221d38b8bad6d939e50f17ffda40f0510b4d28506bd3": "0xf390e256b6ed8a1b283d3ea80b103b868c14c31e5b7114fc32fff21c4cb263eb",
	"0x03e864b08b08f632c61c6727cde0e23d125f7784b5a5a188446fc5c91ffa51faa1": "0xb385aca81e134722cca902bf85443528c3d3a783cf54008cfc34a2ca563fc5b6",
}

type BlockChain struct {
	GenesisBlock    *Block
	futureBlocks    *lru.Cache
	Storage         storage.Storage
	TransactionPool *TransactionPool
	Consensus       Consensus
}

func NewBlockChain(consensus Consensus) (*BlockChain, error) {
	storage, err := storage.NewMemoryStorage()
	if err != nil {
		return nil, err
	}
	futureBlocks, _ := lru.New(maxFutureBlocks)
	bc, err := BlockChain{
		Storage:      storage,
		futureBlocks: futureBlocks,
	}, nil

	bc.GenesisBlock, err = GetGenesisBlock(storage)
	bc.Consensus = consensus

	//MinerState
	ms, _ := bc.Consensus.NewMinerState(common.Hash{}, storage)
	bc.GenesisBlock.MinerState = ms
	minerGroup, _ := ms.GetMinerGroup(&bc, bc.GenesisBlock)
	ms.Put(minerGroup, bc.GenesisBlock.VoterState.RootHash())
	bc.GenesisBlock.Header.MinerHash = ms.RootHash()
	bc.GenesisBlock.Header.SnapshotVoterTime = bc.GenesisBlock.Header.Time

	bc.GenesisBlock.MakeHash()
	return &bc, err
}

//make genesisblock and save state
func GetGenesisBlock(storage storage.Storage) (*Block, error) {
	//TODO: load genesis block from config or db
	// var coinbaseAddress = "0x036407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0"
	common.Hex2Bytes(GenesisCoinbaseAddress)
	header := &Header{
		Coinbase: common.BytesToAddress(common.FromHex(GenesisCoinbaseAddress)),
		Height:   0,
		Time:     0,
	}
	block := &Block{
		Header: header,
	}
	//FIXME: change location to save genesis state
	//-------
	//AccountState
	accs, _ := NewAccountState(storage)
	account := Account{}
	copy(account.Address[:], common.FromHex(GenesisCoinbaseAddress))
	account.AddBalance(new(big.Int).SetUint64(100))
	accs.PutAccount(&account)
	block.AccountState = accs
	header.AccountHash = accs.RootHash()

	//TransactionState
	txs, _ := NewTransactionState(storage)
	txs.PutTransaction(&Transaction{})
	block.TransactionState = txs
	header.TransactionHash = txs.RootHash()

	//VoterState
	vs, _ := NewAccountState(storage)
	account1 := Account{}
	copy(account1.Address[:], common.FromHex(GenesisCoinbaseAddress))
	account1.AddBalance(new(big.Int).SetUint64(100))
	vs.PutAccount(&account1)

	account2 := Account{}
	copy(account2.Address[:], common.FromHex("0x03fdefdefbb2478f3d1ed3221d38b8bad6d939e50f17ffda40f0510b4d28506bd3"))
	account2.AddBalance(new(big.Int).SetUint64(20))
	vs.PutAccount(&account2)

	account3 := Account{}
	copy(account3.Address[:], common.FromHex("0x03e864b08b08f632c61c6727cde0e23d125f7784b5a5a188446fc5c91ffa51faa1"))
	account3.AddBalance(new(big.Int).SetUint64(50))
	vs.PutAccount(&account3)

	block.VoterState = vs
	header.VoterHash = vs.RootHash()

	// MinderState
	//FIXME: current in NewBlockChain

	//-------

	//block.MakeHash()
	return block, nil
}

func (bc *BlockChain) GetBlockByHash(hash common.Hash) (*Block, error) {
	encodedBytes, err := bc.Storage.Get(hash[:])
	if err != nil {
		return nil, err
	}
	block := Block{}
	rlp.NewStream(bytes.NewReader(encodedBytes), 0).Decode(&block)
	return &block, nil
}

func (bc *BlockChain) GetBlockByHeight(height uint64) (*Block, error) {
	encodedBytes, err := bc.Storage.Get(encodeBlockHeight(height))
	if err != nil {
		return nil, err
	}

	block := Block{}
	rlp.NewStream(bytes.NewReader(encodedBytes), 0).Decode(&block)
	return &block, nil
}

func (bc *BlockChain) PutState(block *Block) {
	//the state save here except genesis block
	//FIXME: verify genesis block
	if block.Header.Height == uint64(0) {
		return
	}
	parentBlock, _ := bc.GetBlockByHash(block.Header.ParentHash)
	block.AccountState, _ = NewAccountStateRootHash(parentBlock.Header.AccountHash, bc.Storage)
	block.TransactionState, _ = NewTransactionStateRootHash(parentBlock.Header.TransactionHash, bc.Storage)
	block.VoterState, _ = NewAccountStateRootHash(parentBlock.Header.VoterHash, bc.Storage)
	block.MinerState, _ = bc.Consensus.NewMinerState(parentBlock.Header.MinerHash, bc.Storage)

	bc.RewardForCoinbase(block)

	//miner check
	//bc.PutMinerState(block)
	if err := bc.ExecuteTransaction(block); err != nil {
		return
	}
}

// func (bc *BlockChain) PutVoterState(block *Block) error {
// 	return nil
// }

func (bc *BlockChain) PutMinerState(block *Block) error {

	//1. save status and check hash
	ms := block.MinerState
	minerGroup, err := ms.GetMinerGroup(bc, block)
	if err != nil {
		return err
	}

	ms.Put(minerGroup, common.Hash{}) //TODO voterhash
	if ms.RootHash() != block.Header.MinerHash {
		return errors.New("minerState.RootHash() != block.Header.MinerHash")
	}

	//2. check the order to mine
	index := block.Header.Height % 3
	if minerGroup[index] != block.Header.Coinbase {
		return errors.New("minerGroup[index] != block.Header.Coinbase")
	}

	// //3. set state,  nil before setting state
	// block.MinderState = ms
	return nil

}

func (bc *BlockChain) RewardForCoinbase(block *Block) {
	//FIXME: return nil when using Clone
	accs := block.AccountState //NewAccountStateRootHash(parentBlock.Header.AccountHash, bc.Storage)
	account := accs.GetAccount(block.Header.Coinbase)
	if account == nil { // At first, genesisblock
		account = &Account{Address: block.Header.Coinbase}
	}
	//FIXME: 100 for reward
	account.AddBalance(new(big.Int).SetUint64(100))
	accs.PutAccount(account)
	// fmt.Printf("%v\n", account.Balance)
	// fmt.Printf("%v\n", block.Header.Coinbase)

	//set state,  nil before setting state
	// block.AccountState = accs
}

func (bc *BlockChain) ExecuteTransaction(block *Block) error {
	accs := block.AccountState
	txs := block.TransactionState

	for _, tx := range block.Transactions {
		fromAccount := accs.GetAccount(tx.From)
		toAccount := accs.GetAccount(tx.To)
		// fmt.Printf("%v\n", tx.From)
		// fmt.Printf("%v\n", fromAccount.Balance)
		if err := fromAccount.SubBalance(tx.Amount); err != nil {
			return err
		}
		toAccount.AddBalance(tx.Amount)

		accs.PutAccount(fromAccount)
		accs.PutAccount(toAccount)
		txs.PutTransaction(tx)
		//implement vote transaction later
	}
	// fmt.Printf("%v\n", accs.RootHash())
	if accs.RootHash() != block.Header.AccountHash {
		return errors.New("accs.RootHash() != block.Header.AccountHash")
	}
	if txs.RootHash() != block.Header.TransactionHash {
		return errors.New("txs.RootHash() != block.Header.TransactionHash")
	}

	return nil
}

func (bc *BlockChain) PutBlock(block *Block) {
	//1. verify transaction
	err := block.VerifyTransacion()
	if err != nil {
		fmt.Println("VerifyTransacion")
		return
	}

	//2. save status and verify hash
	bc.PutState(block)

	//4. verify block.hash
	if block.Hash() != block.CalcHash() {
		fmt.Println("block.Hash() != block.CalcHash()")
		return
	}

	encodedBytes, _ := rlp.EncodeToBytes(block)
	//TODO: change height , hash
	bc.Storage.Put(block.Header.Hash[:], encodedBytes)
	bc.Storage.Put(encodeBlockHeight(block.Header.Height), encodedBytes)
}

func (bc *BlockChain) HasParentInBlockChain(block *Block) bool {
	//TODO: check  block.Header.ParentHash[:] != nil
	if block.Header.ParentHash[:] != nil {
		b, _ := bc.GetBlockByHash(block.Header.ParentHash)
		if b != nil {
			return true
		}
	}
	return false
}

func (bc *BlockChain) putBlockIfParentExistInFutureBlocks(block *Block) {
	if bc.futureBlocks.Contains(block.Hash()) {
		block, _ := bc.futureBlocks.Get(block.Hash())
		bc.PutBlock(block.(*Block))
		bc.putBlockIfParentExistInFutureBlocks(block.(*Block))
	}
}

func (bc *BlockChain) PutBlockIfParentExist(block *Block) {
	if bc.HasParentInBlockChain(block) {
		bc.PutBlock(block)
		bc.putBlockIfParentExistInFutureBlocks(block)
	} else {
		bc.futureBlocks.Add(block.Header.ParentHash, block)
	}
}

func encodeBlockHeight(number uint64) []byte {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, number)
	return enc
}
