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
	peer "github.com/libp2p/go-libp2p-peer"
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
	libKey          = "lib"
	tailKey         = "tail"
	maxFutureBlocks = 256
)

var GenesisCoinbaseAddress = string("0x036407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0")

type BlockChain struct {
	mu           sync.RWMutex
	GenesisBlock *Block
	futureBlocks *lru.Cache
	Storage      storage.Storage
	TxPool       *TransactionPool
	Consensus    Consensus
	Lib          *Block
	Tail         *Block
	node         net.INode
	tailGroup    *sync.Map
}

func NewBlockChain(consensus Consensus, storage storage.Storage) *BlockChain {
	futureBlocks, _ := lru.New(maxFutureBlocks)
	bc := BlockChain{
		Storage:      storage,
		futureBlocks: futureBlocks,
		tailGroup:    new(sync.Map),
	}

	bc.Consensus = consensus
	return &bc
}

func (bc *BlockChain) Setup(voters []*Account) {
	err := bc.LoadBlockChainFromStorage()
	if err != nil {
		bc.MakeGenesisBlock(voters)
		bc.PutBlockByCoinbase(bc.GenesisBlock)
	} else {
		bc.LoadLibFromStorage()
		bc.LoadTailFromStorage()
	}
	bc.TxPool = NewTransactionPool()

}

func (bc *BlockChain) LoadBlockChainFromStorage() error {
	block, err := bc.GetBlockByHeight(0)
	if err != nil {
		return err
	}
	//status
	block.AccountState, _ = NewAccountStateRootHash(block.Header.AccountHash, bc.Storage)
	block.TransactionState, _ = NewTransactionStateRootHash(block.Header.TransactionHash, bc.Storage)
	block.VoterState, _ = NewAccountStateRootHash(block.Header.VoterHash, bc.Storage)
	block.MinerState, _ = bc.Consensus.NewMinerState(block.Header.MinerHash, bc.Storage)
	bc.GenesisBlock = block
	return nil

}

func (bc *BlockChain) MakeGenesisBlock(voters []*Account) {
	common.Hex2Bytes(GenesisCoinbaseAddress)
	header := &Header{
		Coinbase: common.BytesToAddress(common.FromHex(GenesisCoinbaseAddress)),
		Height:   0,
		Time:     0,
	}
	block := &Block{
		Header: header,
	}

	//AccountState
	accs, _ := NewAccountState(bc.Storage)
	account := Account{}
	copy(account.Address[:], common.FromHex(GenesisCoinbaseAddress))
	account.AddBalance(new(big.Int).SetUint64(100)) //FIXME: amount 0
	accs.PutAccount(&account)
	block.AccountState = accs
	header.AccountHash = accs.RootHash()

	//TransactionState
	txs, _ := NewTransactionState(bc.Storage)
	txs.PutTransaction(&Transaction{})
	block.TransactionState = txs
	header.TransactionHash = txs.RootHash()

	//VoterState
	vs, _ := NewAccountState(bc.Storage)
	for _, account := range voters {
		vs.PutAccount(account)
	}
	block.VoterState = vs
	header.VoterHash = vs.RootHash()
	bc.GenesisBlock = block

	// MinerState
	ms, _ := bc.Consensus.NewMinerState(common.Hash{}, bc.Storage)
	bc.GenesisBlock.MinerState = ms
	minerGroup, _, _ := ms.GetMinerGroup(bc, block)
	ms.Put(minerGroup, bc.GenesisBlock.VoterState.RootHash())

	bc.GenesisBlock = block
	bc.GenesisBlock.Header.MinerHash = ms.RootHash()
	bc.GenesisBlock.Header.SnapshotVoterTime = bc.GenesisBlock.Header.Time
	bc.GenesisBlock.MakeHash()

	bc.SetLib(bc.GenesisBlock)
	bc.SetTail(bc.GenesisBlock)
}

func (bc *BlockChain) SetNode(node net.INode) {
	bc.node = node
	node.RegisterSubscriber(net.MSG_NEW_BLOCK, bc)
	node.RegisterSubscriber(net.MSG_MISSING_BLOCK, bc)
	node.RegisterSubscriber(net.MSG_MISSING_BLOCK_ACK, bc)
	node.RegisterSubscriber(net.MSG_NEW_TX, bc)
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
		return nil, err
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
		log.CLog().Warning(err)
		return err
	}
	//TODO: check double spending ?
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
		if fromAccount.Nonce+1 != tx.Nonce {
			return ErrTransactionNonce
		}
		fromAccount.Nonce += uint64(1)
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
		log.CLog().Warning("Error VerifyTransacion")
		return
	}

	//2. save status and verify hash
	err = bc.PutState(block)
	if err != nil {
		log.CLog().Warning("Error PutState")
		return
	}

	//4. verify block.hash
	if block.Hash() != block.CalcHash() {
		log.CLog().Info("block.Hash() != block.CalcHash()")
		return
	}

	//5.signer check
	v, _ := block.VerifySign()
	if !v || err != nil {
		log.CLog().WithFields(logrus.Fields{
			"Height": block.Header.Height,
			"Err":    err,
		}).Warning("Signature is invalid")
		return
	}

	bc.putBlockToStorage(block)
	log.CLog().WithFields(logrus.Fields{
		"Height": block.Header.Height,
		//"hash":   common.Hash2Hex(block.Hash()),
	}).Info("Imported block")

	//set tail
	bc.SetTail(block)

	bc.tailGroup.Store(block.Hash(), block)
	//if parent exist
	bc.tailGroup.Delete(block.Header.ParentHash)

	//remove tx
	bc.RemoveTxInPool(block)
}

func (bc *BlockChain) AddTailToGroup(block *Block) {
	bc.tailGroup.Store(block.Hash(), block)
	//if parent exist
	bc.tailGroup.Delete(block.Header.ParentHash)
}

func (bc *BlockChain) PutBlockByCoinbase(block *Block) {
	bc.mu.Lock()
	bc.putBlockToStorage(block)
	bc.SetTail(block)
	bc.mu.Unlock()
	log.CLog().WithFields(logrus.Fields{
		"Height":   block.Header.Height,
		"Tx count": len(block.Transactions),
	}).Info("Mined block")
	bc.AddTailToGroup(block)
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
		bc.AddFutureBlock(block)
	}
}

func (bc *BlockChain) AddFutureBlock(block *Block) {
	log.CLog().WithFields(logrus.Fields{
		"Height": block.Header.Height,
		"hash":   common.Hash2Hex(block.Hash()),
	}).Debug("Inserted block into  future blocks")
	bc.futureBlocks.Add(block.Header.ParentHash, block)
	//FIXME: temporarily, must send hash
	if block.Header.Height > uint64(1) {
		msg, _ := net.NewRLPMessage(net.MSG_MISSING_BLOCK, block.Header.Height-uint64(1))
		bc.node.SendMessageToRandomNode(&msg)
		log.CLog().WithFields(logrus.Fields{
			"Height": block.Header.Height - uint64(1),
		}).Info("Request missing block")
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
	if message.Code == net.MSG_NEW_BLOCK || message.Code == net.MSG_MISSING_BLOCK_ACK {
		block := &Block{}
		rlp.DecodeBytes(message.Payload, block)
		log.CLog().WithFields(logrus.Fields{
			"height": block.Header.Height,
		}).Debug("new block arrrived")

		if block.Header.Height == 1 {
			log.CLog().WithFields(logrus.Fields{
				"height": block.Header.Height,
			}).Debug("new block arrrived")
		}
		bc.PutBlockIfParentExist(block)
		bc.Consensus.UpdateLIB(bc)
		bc.RemoveOrphanBlock()
	} else if message.Code == net.MSG_MISSING_BLOCK {
		height := uint64(0)
		rlp.DecodeBytes(message.Payload, &height)
		log.CLog().WithFields(logrus.Fields{
			"Height": height,
		}).Debug("missing block request arrived")
		bc.SendMissingBlock(height, message.PeerID)
	} else if message.Code == net.MSG_NEW_TX {
		tx := &Transaction{}
		rlp.DecodeBytes(message.Payload, &tx)
		log.CLog().WithFields(logrus.Fields{
			"From":   common.Address2Hex(tx.From),
			"To":     common.Address2Hex(tx.To),
			"Amount": tx.Amount,
		}).Info("Received tx")
		bc.TxPool.Put(tx)

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
		bc.node.SendMessageToRandomNode(&msg)
		log.CLog().WithFields(logrus.Fields{
			"Height": i,
		}).Info("Request missing block")
	}
}

func (bc *BlockChain) SendMissingBlock(height uint64, peerID peer.ID) {
	block, _ := bc.GetBlockByHeight(height)
	if block != nil {
		message, _ := net.NewRLPMessage(net.MSG_MISSING_BLOCK_ACK, block)
		bc.node.SendMessage(&message, peerID)
		log.CLog().WithFields(logrus.Fields{
			"Height": height,
		}).Info("Send missing block")
	} else {
		log.CLog().WithFields(logrus.Fields{
			"Height": height,
		}).Info("We don't have missing block")
	}
}

func (bc *BlockChain) RemoveOrphanBlock() {
	TailTxs := bc.Tail.TransactionState
	bc.tailGroup.Range(func(key, value interface{}) bool {
		tail := value.(*Block)
		var err error
		if bc.Lib.Header.Height >= tail.Header.Height {
			validBlock, _ := bc.GetBlockByHeight(tail.Header.Height)
			for validBlock.Hash() != tail.Hash() {
				removableBlock := tail
				validBlock, _ = bc.GetBlockByHash(validBlock.Header.ParentHash)
				tail, err = bc.GetBlockByHash(tail.Header.ParentHash)
				for _, tx := range removableBlock.Transactions {
					_tx := TailTxs.GetTransaction(tx.Hash)
					if _tx == nil {
						bc.TxPool.Put(tx)
					}
				}
				bc.Storage.Del(common.HashToBytes(removableBlock.Hash()))
				//already removed during for loop
				if err != nil {
					break
				}
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
		if block.Hash() == bc.Lib.Hash() {
			break
		}
		block, _ = bc.GetBlockByHash(block.Header.ParentHash)
		bc.Storage.Put(encodeBlockHeight(block.Header.Height), block.Header.Hash[:])
	}
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

func (bc *BlockChain) SetLib(block *Block) {
	bc.Lib = block
	bc.Storage.Put([]byte(libKey), block.Header.Hash[:])
}

func (bc *BlockChain) LoadLibFromStorage() error {
	hash, err := bc.Storage.Get([]byte(libKey))
	if err != nil {
		return err
	}
	block, err := bc.GetBlockByHash(common.BytesToHash(hash))
	block.AccountState, _ = NewAccountStateRootHash(block.Header.AccountHash, bc.Storage)
	block.TransactionState, _ = NewTransactionStateRootHash(block.Header.TransactionHash, bc.Storage)
	block.VoterState, _ = NewAccountStateRootHash(block.Header.VoterHash, bc.Storage)
	block.MinerState, _ = bc.Consensus.NewMinerState(block.Header.MinerHash, bc.Storage)
	bc.Lib = block
	return nil
}

func (bc *BlockChain) SetTail(block *Block) {
	if bc.Tail == nil {
		bc.Tail = block
		bc.Storage.Put([]byte(tailKey), block.Header.Hash[:])
	}
	if block.Header.Height >= bc.Tail.Header.Height {
		bc.Tail = block
		bc.Storage.Put([]byte(tailKey), block.Header.Hash[:])
		log.CLog().WithFields(logrus.Fields{
			"Height": block.Header.Height,
		}).Debug("Tail")
		bc.RebuildBlockHeight()
	}
}

func (bc *BlockChain) LoadTailFromStorage() error {
	hash, err := bc.Storage.Get([]byte(tailKey))
	if err != nil {
		return err
	}
	block, err := bc.GetBlockByHash(common.BytesToHash(hash))
	block.AccountState, _ = NewAccountStateRootHash(block.Header.AccountHash, bc.Storage)
	block.TransactionState, _ = NewTransactionStateRootHash(block.Header.TransactionHash, bc.Storage)
	block.VoterState, _ = NewAccountStateRootHash(block.Header.VoterHash, bc.Storage)
	block.MinerState, _ = bc.Consensus.NewMinerState(block.Header.MinerHash, bc.Storage)
	bc.Tail = block
	return nil
}
func (bc *BlockChain) RemoveTxInPool(block *Block) {
	for _, tx := range block.Transactions {
		bc.TxPool.Del(tx.Hash)
	}
}

func (bc *BlockChain) BroadcastNewTXMessage(tx *Transaction) {
	message, _ := net.NewRLPMessage(net.MSG_NEW_TX, tx)
	bc.node.BroadcastMessage(&message)
}
