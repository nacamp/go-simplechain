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

type BlockChain struct {
	GenesisBlock    *Block
	futureBlocks    *lru.Cache
	Storage         storage.Storage
	TransactionPool *TransactionPool
}

func NewBlockChain() (*BlockChain, error) {
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
	return &bc, err
}

//make genesisblock and save state
func GetGenesisBlock(storage storage.Storage) (*Block, error) {
	//TODO: load genesis block from config or db
	var coinbaseAddress = "0x036407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0"
	common.Hex2Bytes(coinbaseAddress)
	header := &Header{
		Coinbase: common.BytesToAddress(common.FromHex(coinbaseAddress)),
		Height:   0,
		Time:     new(big.Int).SetUint64(1541072021),
	}
	block := &Block{
		Header: header,
	}
	//FIXME: change location to save genesis state
	//-------
	accs, _ := NewAccountState(storage)
	txs, _ := NewTransactionState(storage)
	account := Account{}
	copy(account.Address[:], common.FromHex(coinbaseAddress))
	account.AddBalance(new(big.Int).SetUint64(100))
	accs.PutAccount(&account)
	header.AccountHash = accs.RootHash()
	block.AccountState = accs

	txs.PutTransaction(&Transaction{})
	header.TransactionHash = txs.RootHash()
	block.TransactionState = txs
	//-------

	block.MakeHash()
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
	if block.Header.Height == uint64(0) {
		return
	}
	bc.RewardForCoinbase(block)
	if err := bc.ExecuteTransaction(block); err != nil {
		return
	}
}

func (bc *BlockChain) RewardForCoinbase(block *Block) {
	parentBlock, _ := bc.GetBlockByHash(block.Header.ParentHash)
	//FIXME: return nil when using Clone
	accs, _ := NewAccountStateRootHash(parentBlock.Header.AccountHash, bc.Storage)
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
	block.AccountState = accs
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

	//3. verify block.hash
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
