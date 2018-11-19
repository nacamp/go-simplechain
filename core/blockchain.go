package core

import (
	"bytes"
	"encoding/binary"
	"errors"
	"sort"
	"sync"
	"time"

	"math/big"

	lru "github.com/hashicorp/golang-lru"
	"github.com/najimmy/go-simplechain/common"
	"github.com/najimmy/go-simplechain/log"
	"github.com/najimmy/go-simplechain/net"
	"github.com/najimmy/go-simplechain/rlp"
	"github.com/najimmy/go-simplechain/storage"
	"github.com/sirupsen/logrus"
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
	mu              sync.RWMutex
	GenesisBlock    *Block
	futureBlocks    *lru.Cache
	Storage         storage.Storage
	TransactionPool *TransactionPool
	Consensus       Consensus
	Lib             *Block
	Tail            *Block
	node            *net.Node
	tailGroup       *sync.Map
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
		tailGroup:    new(sync.Map),
	}, nil

	bc.GenesisBlock, err = GetGenesisBlock(storage)
	bc.Consensus = consensus

	//MinerState
	ms, _ := bc.Consensus.NewMinerState(common.Hash{}, storage)
	bc.GenesisBlock.MinerState = ms
	minerGroup, _, _ := ms.GetMinerGroup(&bc, bc.GenesisBlock)
	ms.Put(minerGroup, bc.GenesisBlock.VoterState.RootHash())
	bc.GenesisBlock.Header.MinerHash = ms.RootHash()
	bc.GenesisBlock.Header.SnapshotVoterTime = bc.GenesisBlock.Header.Time

	bc.GenesisBlock.MakeHash()
	bc.Lib = bc.GenesisBlock
	bc.SetTail(bc.GenesisBlock)
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

	// MinerState
	//FIXME: current in NewBlockChain

	//-------

	// block.MakeHash()
	return block, nil
}

func (bc *BlockChain) SetNode(node *net.Node) {
	bc.node = node
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

	hash, err := bc.Storage.Get(encodeBlockHeight(height))
	if err != nil {
		return nil, nil
	}
	return bc.GetBlockByHash(common.BytesToHash(hash))
}

func (bc *BlockChain) PutState(block *Block) error {
	//the state save here except genesis block
	//FIXME: verify genesis block
	if block.Header.Height == uint64(0) {
		return nil
	}
	parentBlock, _ := bc.GetBlockByHash(block.Header.ParentHash)
	block.AccountState, _ = NewAccountStateRootHash(parentBlock.Header.AccountHash, bc.Storage)
	block.TransactionState, _ = NewTransactionStateRootHash(parentBlock.Header.TransactionHash, bc.Storage)
	block.VoterState, _ = NewAccountStateRootHash(parentBlock.Header.VoterHash, bc.Storage)
	block.MinerState, _ = bc.Consensus.NewMinerState(parentBlock.Header.MinerHash, bc.Storage)

	bc.RewardForCoinbase(block)

	err := bc.PutMinerState(block)
	if err != nil {
		log.CLog().Info(err)
		return err
	}

	if err := bc.ExecuteTransaction(block); err != nil {
		return err
	}

	//check rootHash
	if block.AccountState.RootHash() != block.Header.AccountHash {
		return errors.New("block.AccountState.RootHash() != block.Header.AccountHash")
	}
	if block.TransactionState.RootHash() != block.Header.TransactionHash {
		return errors.New("block.TransactionState.RootHash() != block.Header.TransactionHash")
	}
	if block.VoterState.RootHash() != block.Header.VoterHash {
		return errors.New("block.VoterState.RootHash() != block.Header.VoterHash")
	}

	if block.MinerState.RootHash() != block.Header.MinerHash {
		return errors.New("block.MinerState.RootHash() != block.Header.MinerHash")
	}
	return nil
}

func (bc *BlockChain) PutMinerState(block *Block) error {

	// save status
	ms := block.MinerState
	minerGroup, voterBlock, err := ms.GetMinerGroup(bc, block)
	if err != nil {
		return err
	}
	//TODO: we need to test  when voter transaction make
	//make new miner group
	if voterBlock.Header.Height == block.Header.Height {

		ms.Put(minerGroup, block.Header.VoterHash) //TODO voterhash
	}
	//else use parent miner group
	//TODO: check after 3 seconds(block creation) and 3 seconds(mining order)
	index := (block.Header.Time % 9) / 3
	if minerGroup[index] != block.Header.Coinbase {
		return errors.New("minerGroup[index] != block.Header.Coinbase")
	}

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
}

func (bc *BlockChain) ExecuteTransaction(block *Block) error {
	accs := block.AccountState
	txs := block.TransactionState

	for _, tx := range block.Transactions {
		fromAccount := accs.GetAccount(tx.From)
		toAccount := accs.GetAccount(tx.To)
		if err := fromAccount.SubBalance(tx.Amount); err != nil {
			return err
		}
		toAccount.AddBalance(tx.Amount)

		accs.PutAccount(fromAccount)
		accs.PutAccount(toAccount)
		txs.PutTransaction(tx)
		//implement vote transaction later
	}
	return nil
}

func (bc *BlockChain) PutBlock(block *Block) {
	//1. verify transaction
	err := block.VerifyTransacion()
	if err != nil {
		log.CLog().Info("Error VerifyTransacion")
		return
	}

	//2. save status and verify hash
	err = bc.PutState(block)
	if err != nil {
		log.CLog().Info("Error PutState")
		return
	}

	//4. verify block.hash
	if block.Hash() != block.CalcHash() {
		log.CLog().Info("block.Hash() != block.CalcHash()")
		return
	}

	//5. TODO: signer check
	bc.putBlockToStorage(block)
	log.CLog().WithFields(logrus.Fields{
		"height": block.Header.Height,
		"hash":   common.Hash2Hex(block.Hash()),
	}).Info("New Block was inserted at Blockchain")

	//set tail
	bc.SetTail(block)

	bc.tailGroup.Store(block.Hash(), block)
	//if parent exist
	bc.tailGroup.Delete(block.Header.ParentHash)

	bc.Consensus.UpdateLIB(bc)
	bc.RemoveOrphanBlock()

}

func (bc *BlockChain) PutBlockByCoinbase(block *Block) {
	bc.mu.Lock()
	bc.putBlockToStorage(block)
	bc.SetTail(block)
	bc.mu.Unlock()
	log.CLog().WithFields(logrus.Fields{
		"Height": block.Header.Height,
		"hash":   common.Hash2Hex(block.Hash()),
	}).Info("Block was created")
	bc.Consensus.UpdateLIB(bc)
	bc.RemoveOrphanBlock()
}

func (bc *BlockChain) HasParentInBlockChain(block *Block) bool {
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

func (bc *BlockChain) NewBlockFromParent(parentBlock *Block) *Block {
	h := &Header{
		ParentHash: parentBlock.Hash(),
		//Coinbase        common.Address
		Height: parentBlock.Header.Height + 1,
		// Time              uint64
		// Hash              common.Hash
		// AccountHash       common.Hash
		// TransactionHash   common.Hash
		// MinerHash         common.Hash
		// VoterHash         common.Hash
		// SnapshotVoterTime: parentBlock.Header.SnapshotVoterTime,
	}
	// h.ParentHash = parentBlock.Hash()
	// h.Height = parentBlock.Header.Height + 1
	// h.Time = time
	block := &Block{
		Header: h,
		// Transactions []*Transaction
		// AccountState     *AccountState
		// TransactionState *TransactionState
		// MinerState       MinerState
		// VoterState       *AccountState
	}
	//state
	block.VoterState, _ = parentBlock.VoterState.Clone()
	block.MinerState, _ = parentBlock.MinerState.Clone()
	block.AccountState, _ = parentBlock.AccountState.Clone()
	block.TransactionState, _ = parentBlock.TransactionState.Clone()
	return block
}

// func (bc *BlockChain) Start() {
// 	//go bc.Loop()
// }

func (bc *BlockChain) HandleMessage(message *net.Message) error {
	if message.Code == net.MSG_NEW_BLOCK {
		block := &Block{}
		rlp.DecodeBytes(message.Payload, block)
		log.CLog().WithFields(logrus.Fields{
			"height": block.Header.Height,
		}).Info("new block arrrived")

		if block.Header.Height == 1 {
			log.CLog().WithFields(logrus.Fields{
				"height": block.Header.Height,
			}).Info("new block arrrived")
		}
		bc.PutBlockIfParentExist(block)
	} else if message.Code == net.MSG_MISSING_BLOCK {
		height := uint64(0)
		rlp.DecodeBytes(message.Payload, &height)
		log.CLog().WithFields(logrus.Fields{
			"Height": height,
		}).Info("missing block request arrived")
		bc.SendMissingBlock(height)
	}
	return nil
}

// func (sp *SubsriberPool) handleMessage(message *Message) {
// 	sp.messageCh <- message
// }

//TODO: use code temporarily
func (bc *BlockChain) RequestMissingBlock() {
	missigBlock := make(map[uint64]bool)
	for _, k := range bc.futureBlocks.Keys() {
		v, _ := bc.futureBlocks.Peek(k)
		block := v.(*Block)
		missigBlock[block.Header.Height] = true
	}
	var keys []int
	for k := range missigBlock {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	if len(keys) == 0 {
		return
	}
	for i := bc.Tail.Header.Height + 1; i < uint64(keys[0]); i++ {
		msg, _ := net.NewRLPMessage(net.MSG_MISSING_BLOCK, uint64(i))
		bc.node.SendMessage(&msg)
		log.CLog().WithFields(logrus.Fields{
			"Height": i,
		}).Info("request missing block")
	}
}

func (bc *BlockChain) SendMissingBlock(height uint64) {
	block, _ := bc.GetBlockByHeight(height)
	if block != nil {
		message, _ := net.NewRLPMessage(net.MSG_NEW_BLOCK, block)
		bc.node.SendMessage(&message)
		log.CLog().WithFields(logrus.Fields{
			"Height": height,
		}).Info("missing block send")
	} else {
		log.CLog().WithFields(logrus.Fields{
			"Height": height,
		}).Info("We don't have missing block")
	}
}

func (bc *BlockChain) RemoveOrphanBlock() {
	bc.tailGroup.Range(func(key, value interface{}) bool {
		orphanBlock := value.(*Block)
		if bc.Lib.Header.Height >= orphanBlock.Header.Height {
			validBlock, _ := bc.GetBlockByHeight(orphanBlock.Header.Height)
			orphanBlockHash := orphanBlock.Hash()
			for validBlock.Hash() != orphanBlock.Hash() {
				validBlock, _ = bc.GetBlockByHash(validBlock.Header.ParentHash)
				orphanBlock, _ = bc.GetBlockByHash(orphanBlock.Header.ParentHash)
				bc.Storage.Del(orphanBlockHash[:])
			}
		}
		return true
	})
}

func (bc *BlockChain) RebuildBlockHeight() {
	block := bc.Tail
	if block.Header.Height == 0 {
		return
	}
	for {
		block, _ := bc.GetBlockByHash(block.Header.ParentHash)
		block2, _ := bc.GetBlockByHeight(block.Header.Height)
		if block.Hash() == block2.Hash() {
			break
		}
		bc.Storage.Put(encodeBlockHeight(block.Header.Height), block.Header.Hash[:])
	}
}

func (bc *BlockChain) SetTail(block *Block) {
	bc.Tail = block
	bc.RebuildBlockHeight()
}

func (bc *BlockChain) Start() {
	go bc.loop()
}

func (bc *BlockChain) loop() {
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ticker.C:
			bc.RequestMissingBlock()
		}
	}
}

func (bc *BlockChain) putBlockToStorage(block *Block) {
	encodedBytes, _ := rlp.EncodeToBytes(block)
	bc.Storage.Put(block.Header.Hash[:], encodedBytes)
	bc.Storage.Put(encodeBlockHeight(block.Header.Height), block.Header.Hash[:])
}
