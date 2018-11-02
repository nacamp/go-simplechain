package core

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/rlp"
	lru "github.com/hashicorp/golang-lru"
	"github.com/najimmy/go-simplechain/common"
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
	GenesisBlock *Block
	futureBlocks *lru.Cache
	storage      storage.Storage
	AccountState *AccountState
}

func NewBlockChain() (*BlockChain, error) {
	storage, err := storage.NewMemoryStorage()
	if err != nil {
		return nil, err
	}
	futureBlocks, _ := lru.New(maxFutureBlocks)
	bc, err := BlockChain{
		storage:      storage,
		futureBlocks: futureBlocks,
	}, nil

	bc.AccountState, err = NewAccountState()
	bc.GenesisBlock, err = GetGenesisBlock()

	return &bc, err
}

func GetGenesisBlock() (*Block, error) {
	//TODO: load genesis block from config or db
	/*
		priv/pub
		e68fb0a479c495910c8351c3593667028b45d679f55ce22b0514c4a8a6bcbdd1 / 036407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0
	*/
	var coinbaseAddress = "036407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0"
	common.Hex2Bytes(coinbaseAddress)
	header := &Header{
		Coinbase: common.BytesToAddress(common.Hex2Bytes(coinbaseAddress)),
		Height:   0,
		Time:     new(big.Int).SetUint64(1541072021),
	}
	block := &Block{
		Header: header,
	}
	//FIXME: change location to save genesis state
	//-------
	account := Account{}
	copy(account.Address[:], common.Hex2Bytes(coinbaseAddress))
	account.AddBalance(new(big.Int).SetUint64(100))
	accountState, _ := NewAccountState()
	accountState.PutAccount(&account)
	copy(header.AccountHash[:], accountState.Trie.RootHash())
	//-------

	block.MakeHash()
	return block, nil
}

func (bc *BlockChain) GetBlockByHash(hash common.Hash) (*Block, error) {
	encodedBytes, err := bc.storage.Get(hash[:])
	if err != nil {
		return nil, err
	}
	block := Block{}
	rlp.NewStream(bytes.NewReader(encodedBytes), 0).Decode(&block)
	return &block, nil
}

func (bc *BlockChain) GetBlockByHeight(height uint64) (*Block, error) {
	encodedBytes, err := bc.storage.Get(encodeBlockHeight(height))
	if err != nil {
		return nil, err
	}

	block := Block{}
	rlp.NewStream(bytes.NewReader(encodedBytes), 0).Decode(&block)
	return &block, nil
}

func (bc *BlockChain) DummyReward() {
	var coinbaseAddress = "036407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0"
	var coinbaseAddress2 common.Address
	copy(coinbaseAddress2[:], common.Hex2Bytes(coinbaseAddress))
	account := bc.AccountState.GetAccount(coinbaseAddress2)
	if account == nil { // At first, genesisblock
		account = &Account{Address: coinbaseAddress2}
	}
	account.AddBalance(new(big.Int).SetUint64(100))
	bc.AccountState.PutAccount(account)
}

func (bc *BlockChain) VerifyState(block *Block) bool {
	//FIXME: where verfyState => Block
	//check reward
	var rootHash common.Hash
	copy(rootHash[:], bc.AccountState.Trie.RootHash())
	if block.Header.AccountHash != rootHash {
		return false
	} else {
		return true
	}

}

func (bc *BlockChain) PutBlock(block *Block) {
	//FIXME: check if valid state
	bc.DummyReward()
	if bc.VerifyState(block) == false {
		fmt.Println("error.....")
		return
	}
	encodedBytes, _ := rlp.EncodeToBytes(block)
	//TODO: change height , hash
	bc.storage.Put(block.Header.Hash[:], encodedBytes)
	bc.storage.Put(encodeBlockHeight(block.Header.Height), encodedBytes)
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
