package core

import (
	"bytes"
	"encoding/binary"
	"math/big"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/najimmy/go-simplechain/common"
	"github.com/najimmy/go-simplechain/storage"
)

/*
First time
search the height or the hash
add block
ignore block validity
*/
type BlockChain struct {
	GenesisBlock *Block
	storage      storage.Storage
}

func NewBlockChain() (*BlockChain, error) {
	storage, err := storage.NewMemoryStorage()
	if err != nil {
		return nil, err
	}
	blockChain, err := BlockChain{
		storage: storage,
	}, nil
	blockChain.GenesisBlock, err = GetGenesisBlock()

	return &blockChain, err
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

func (bc *BlockChain) PutBlock(block Block) {
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

func encodeBlockHeight(number uint64) []byte {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, number)
	return enc
}
