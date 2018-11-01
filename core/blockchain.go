package core

import (
	"bytes"
	"encoding/binary"

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
	storage storage.Storage
}

func NewBlockChain() (*BlockChain, error) {
	storage, err := storage.NewMemoryStorage()
	if err != nil {
		return nil, err
	}
	return &BlockChain{
		storage: storage,
	}, nil
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

func encodeBlockHeight(number uint64) []byte {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, number)
	return enc
}
