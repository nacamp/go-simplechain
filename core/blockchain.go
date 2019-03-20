package core

import (
	"bytes"
	"encoding/binary"
	"sync"

	"github.com/pkg/errors"

	"math/big"

	lru "github.com/hashicorp/golang-lru"
	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/log"
	"github.com/nacamp/go-simplechain/net"
	"github.com/nacamp/go-simplechain/rlp"
	"github.com/nacamp/go-simplechain/storage"
	"github.com/sirupsen/logrus"
)

const (
	libKey          = "lib"
	tailKey         = "tail"
	maxFutureBlocks = 256
)

type BlockChain struct {
	mu                  sync.RWMutex
	GenesisBlock        *Block
	futureBlocks        *lru.Cache
	Storage             storage.Storage
	TxPool              *TransactionPool
	Consensus           Consensus
	lib                 *Block
	tail                *Block
	MessageToRandomNode chan *net.Message
	BroadcastMessage    chan *net.Message
	NewTXMessage        chan *Transaction
	LibCh               chan struct{}
	tailGroup           *sync.Map
	coinbase            common.Address
	miningReward        uint64
	//poa
	Signers []common.Address
}

func NewBlockChain(storage storage.Storage, coinbase common.Address, miningReward uint64) *BlockChain {
	futureBlocks, _ := lru.New(maxFutureBlocks)
	bc := BlockChain{
		Storage:             storage,
		futureBlocks:        futureBlocks,
		tailGroup:           new(sync.Map),
		MessageToRandomNode: make(chan *net.Message, 1),
		BroadcastMessage:    make(chan *net.Message, 1),
		NewTXMessage:        make(chan *Transaction, 1),
		LibCh:               make(chan struct{}, 1),
		coinbase:            coinbase,
		miningReward:        miningReward,
	}
	return &bc
}

func (bc *BlockChain) Setup(consensus Consensus, voters []*Account) {
	consensus.AddBlockChain(bc)
	bc.Consensus = consensus
	if ok := bc.LoadBlockChainFromStorage(); ok {
		bc.LoadLibFromStorage()
		bc.LoadTailFromStorage()
	} else {
		bc.MakeGenesisBlock(voters)
		bc.PutBlockByCoinbase(bc.GenesisBlock)
	}
	bc.TxPool = NewTransactionPool()
}

func (bc *BlockChain) LoadBlockChainFromStorage() bool {
	block := bc.GetBlockByHeight(0)
	if block == nil {
		return false
	}
	var err error
	//status
	block.AccountState, err = NewAccountStateRootHash(block.Header.AccountHash, bc.Storage)
	if err != nil {
		log.CLog().WithFields(logrus.Fields{}).Panic(err)
	}
	block.TransactionState, err = NewTransactionStateRootHash(block.Header.TransactionHash, bc.Storage)
	if err != nil {
		log.CLog().WithFields(logrus.Fields{}).Panic(err)
	}

	consensusState, err := bc.Consensus.LoadState(block)
	if err != nil {
		log.CLog().WithFields(logrus.Fields{}).Panic(err)
	}
	block.SetConsensusState(consensusState)

	bc.GenesisBlock = block
	return true

}

func (bc *BlockChain) MakeGenesisBlock(voters []*Account) {
	header := &Header{
		Coinbase: bc.coinbase,
		Height:   0,
		Time:     0,
	}
	block := &Block{
		BaseBlock: BaseBlock{Header: header},
	}

	//AccountState
	accs, err := NewAccountState(bc.Storage)
	if err != nil {
		log.CLog().WithFields(logrus.Fields{}).Panic(err)
	}
	account := NewAccount()
	copy(account.Address[:], bc.coinbase[:])
	account.AddBalance(new(big.Int).SetUint64(bc.miningReward))
	accs.PutAccount(account)
	block.AccountState = accs
	header.AccountHash = accs.RootHash()

	//TransactionState
	txs, err := NewTransactionState(bc.Storage)
	if err != nil {
		log.CLog().WithFields(logrus.Fields{}).Panic(err)
	}
	txs.PutTransaction(&Transaction{})
	block.TransactionState = txs
	header.TransactionHash = txs.RootHash()
	err = bc.Consensus.MakeGenesisBlock(block, voters)
	if err != nil {
		log.CLog().WithFields(logrus.Fields{}).Panic(err)
	}
	bc.SetLib(bc.GenesisBlock)
	bc.SetTail(bc.GenesisBlock)
}

func (bc *BlockChain) GetBlockByHash(hash common.Hash) *Block {
	encodedBytes, err := bc.Storage.Get(hash[:])
	if err != nil {
		if err == storage.ErrKeyNotFound {
			return nil
		}
		log.CLog().WithFields(logrus.Fields{
			"Hash": common.HashToHex(hash),
		}).Panic("")
		return nil
	}
	block := Block{}
	rlp.NewStream(bytes.NewReader(encodedBytes), 0).Decode(&block)
	return &block
}

func (bc *BlockChain) GetBlockByHeight(height uint64) *Block {

	hash, err := bc.Storage.Get(encodeBlockHeight(height))
	if err != nil {
		if err == storage.ErrKeyNotFound {
			return nil
		}
		log.CLog().WithFields(logrus.Fields{
			"Height": height,
		}).Panic("")
		return nil
	}
	return bc.GetBlockByHash(common.BytesToHash(hash))
}

func (bc *BlockChain) PutState(block *Block) error {
	//the state save here except genesis block
	if block.Header.Height == uint64(0) {
		return nil
	}
	var err error
	parentBlock := bc.GetBlockByHash(block.Header.ParentHash)
	if parentBlock == nil {
		return errors.New("ParentBlock is nil")
	}
	block.AccountState, err = NewAccountStateRootHash(parentBlock.Header.AccountHash, bc.Storage)
	if err != nil {
		return err
	}
	block.TransactionState, err = NewTransactionStateRootHash(parentBlock.Header.TransactionHash, bc.Storage)
	if err != nil {
		return err
	}

	// parent maybe not have ConsensusState
	// block.ConsensusState, err = parentBlock.ConsensusState.Clone()
	consensusState, err := bc.Consensus.LoadState(parentBlock)
	if err != nil {
		return err
	}
	block.SetConsensusState(consensusState)

	bc.RewardForCoinbase(block)

	//TODO: check double spending ?
	if err := bc.ExecuteTransaction(block); err != nil {
		return err
	}

	if err := bc.Consensus.SaveState(block); err != nil {
		return err
	}

	//check rootHash
	if block.AccountState.RootHash() != block.Header.AccountHash {
		return errors.New("block.AccountState.RootHash() != block.Header.AccountHash")
	}
	if block.TransactionState.RootHash() != block.Header.TransactionHash {
		return errors.New("block.TransactionState.RootHash() != block.Header.TransactionHash")
	}

	if block.ConsensusState().RootHash() != block.Header.ConsensusHash {
		return errors.New("block.ConsensusState.RootHash() != block.Header.ConsensusHash")
	}

	return nil
}

func (bc *BlockChain) RewardForCoinbase(block *Block) {
	//FIXME: return nil when using Clone
	accs := block.AccountState //NewAccountStateRootHash(parentBlock.Header.AccountHash, bc.Storage)
	account := accs.GetAccount(block.Header.Coinbase)
	if account == nil { // At first, genesisblock
		account = NewAccount()
		account.Address = block.Header.Coinbase
	}
	account.AddBalance(new(big.Int).SetUint64(bc.miningReward))
	accs.PutAccount(account)
}

func (bc *BlockChain) ExecuteTransaction(block *Block) error {
	accs := block.AccountState
	txs := block.TransactionState
	// firstVote := true
	for i, tx := range block.Transactions {
		fromAccount := accs.GetAccount(tx.From)
		if fromAccount.Nonce+1 != tx.Nonce {
			return ErrTransactionNonce
		}
		fromAccount.Nonce += uint64(1)
		if tx.Payload == nil {
			toAccount := accs.GetAccount(tx.To)
			if err := fromAccount.SubBalance(tx.Amount); err != nil {
				return err
			}
			toAccount.AddBalance(tx.Amount)
			accs.PutAccount(toAccount)
		} else {
			block.ConsensusState().ExecuteTransaction(block, i, fromAccount)
		}
		accs.PutAccount(fromAccount)
		txs.PutTransaction(tx)
	}
	return nil
}

func (bc *BlockChain) PutBlock(block *Block) error {
	// if block := bc.GetBlockByHash(block.Hash()); block != nil {
	// 	log.CLog().WithFields(logrus.Fields{
	// 		"Height": block.Header.Height,
	// 	}).Info("There is same block in blockchain")
	// 	// return nil
	// }

	var err error
	//1. verify block.hash
	if block.Hash() != block.CalcHash() {
		return errors.New("block.Hash() != block.CalcHash()")
	}

	//2. check signer
	err = block.VerifySign()
	if err != nil {
		return err
	}

	//3. verify transaction
	err = block.VerifyTransacion()
	if err != nil {
		return err
	}

	//4. save status and verify hash
	err = bc.PutState(block)
	if err != nil {
		return err
	}

	//5. verify consensus
	err = bc.Consensus.Verify(block)
	if err != nil {
		return err
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
	return nil
}

func (bc *BlockChain) AddTailToGroup(block *Block) {
	bc.tailGroup.Store(block.Hash(), block)
	//if parent exist
	bc.tailGroup.Delete(block.Header.ParentHash)
}

func (bc *BlockChain) PutBlockByCoinbase(block *Block) {
	bc.mu.Lock()
	bc.putBlockToStorage(block)
	bc.mu.Unlock()
	bc.SetTail(block)

	log.CLog().WithFields(logrus.Fields{
		"Height":   block.Header.Height,
		"Tx count": len(block.Transactions),
	}).Info("Mined block")
	bc.AddTailToGroup(block)
}

func (bc *BlockChain) HasParentInBlockChain(block *Block) bool {
	if block.Header.ParentHash[:] != nil {
		b := bc.GetBlockByHash(block.Header.ParentHash)
		if b != nil {
			return true
		}
	}
	return false
}

func (bc *BlockChain) putBlockIfParentExistInFutureBlocks(block *Block) error {
	if bc.futureBlocks.Contains(block.Hash()) {
		block, _ := bc.futureBlocks.Get(block.Hash())
		futureBlock := block.(*Block)
		if err := bc.PutBlock(futureBlock); err != nil {
			bc.futureBlocks.Remove(futureBlock.Hash())
			return err
		}
		return bc.putBlockIfParentExistInFutureBlocks(futureBlock)
	}
	return nil
}

func (bc *BlockChain) PutBlockIfParentExist(block *Block) error {
	if bc.HasParentInBlockChain(block) {
		if err := bc.PutBlock(block); err != nil {
			return err
		}
		return bc.putBlockIfParentExistInFutureBlocks(block)
	}
	return bc.AddFutureBlock(block)
}

func (bc *BlockChain) AddFutureBlock(block *Block) error {
	log.CLog().WithFields(logrus.Fields{
		"Height": block.Header.Height,
		"hash":   common.HashToHex(block.Hash()),
	}).Debug("Inserted block into  future blocks")
	if _, ok := bc.futureBlocks.Peek(block.Header.ParentHash); ok {
		return nil
	}
	if block.Header.Height > bc.Lib().Header.Height {
		bc.futureBlocks.Add(block.Header.ParentHash, block)
		if block.Header.Height > uint64(1) { //if parent hash is not genesis block
			msg, err := net.NewRLPMessage(net.MsgMissingBlock, block.Header.ParentHash)
			if err != nil {
				return err
			}
			bc.MessageToRandomNode <- &msg
			log.CLog().WithFields(logrus.Fields{
				"Hash": common.HashToHex(block.Header.ParentHash),
			}).Info("Request missing block")
		}
	}
	return nil
}

//for test
func (bc *BlockChain) FutureBlockSize() int {
	return bc.futureBlocks.Len()
}

func (bc *BlockChain) RemoveFutureBlock() {
	for _, k := range bc.futureBlocks.Keys() {
		if v, ok := bc.futureBlocks.Get(k); ok {
			block := v.(*Block)
			if bc.Lib().Header.Height >= block.Header.Height {
				bc.futureBlocks.Remove(k)
				log.CLog().WithFields(logrus.Fields{
					"Height": block.Header.Height,
				}).Info("Remove old futureBlock")
			}
		}
	}
}

func encodeBlockHeight(number uint64) []byte {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, number)
	return enc
}

func (bc *BlockChain) NewBlockFromTail() (block *Block, err error) {
	parentBlock := bc.Tail()
	bc.mu.Lock()
	defer bc.mu.Unlock()
	h := &Header{
		ParentHash: parentBlock.Hash(),
		Height:     parentBlock.Header.Height + 1,
	}
	block = &Block{
		BaseBlock: BaseBlock{Header: h},
	}

	//state
	//Tail block always have state, but We can not guarantee that another block will have a state.
	consensusState, err := parentBlock.ConsensusState().Clone()
	if err != nil {
		return nil, err
	}
	block.SetConsensusState(consensusState)

	block.AccountState, err = parentBlock.AccountState.Clone()
	if err != nil {
		return nil, err
	}
	block.TransactionState, err = parentBlock.TransactionState.Clone()
	if err != nil {
		return nil, err
	}
	return block, nil
}

func (bc *BlockChain) RequestMissingBlocks() error {
	maxHeight := uint64(18446744073709551615)
	olderHeight := maxHeight
	for _, k := range bc.futureBlocks.Keys() {
		v, _ := bc.futureBlocks.Peek(k)
		block := v.(*Block)
		if bc.Tail().Header.Height < block.Header.Height && olderHeight > block.Header.Height {
			olderHeight = block.Header.Height
		}
	}
	if olderHeight == maxHeight {
		return nil
	}
	var blockRange [2]uint64
	blockRange[0] = bc.Tail().Header.Height + 1
	blockRange[1] = olderHeight
	//request max 100 blocks
	if olderHeight-bc.Tail().Header.Height > 100 {
		blockRange[1] = bc.Tail().Header.Height + 100
	}

	msg, err := net.NewRLPMessage(net.MsgMissingBlocks, blockRange)
	if err != nil {
		return err
	}
	bc.MessageToRandomNode <- &msg
	log.CLog().WithFields(logrus.Fields{
		"Height Start": blockRange[0],
		"Height End":   blockRange[1],
	}).Info("Request missing blocks")

	return nil
}

//TODO: change to request block chunk
// func (bc *BlockChain) RequestMissingBlock() error {
// 	//comment
// 	// missigBlock := make(map[uint64]bool)
// 	// for _, k := range bc.futureBlocks.Keys() {
// 	// 	v, _ := bc.futureBlocks.Peek(k)
// 	// 	block := v.(*Block)
// 	// 	missigBlock[block.Header.Height] = true
// 	// }
// 	// var keys []int
// 	// for k := range missigBlock {
// 	// 	keys = append(keys, int(k))
// 	// }
// 	// sort.Ints(keys)
// 	// if len(keys) == 0 {
// 	// 	return nil
// 	// }
// 	// bc.mu.RLock()
// 	// defer bc.mu.RUnlock()
// 	// //TODO: request chunk blocks
// 	// for i := bc.Tail.Header.Height + 1; i < uint64(keys[0]); i++ {
// 	// 	msg, err := net.NewRLPMessage(net.MsgMissingBlock, uint64(i))
// 	// 	if err != nil {
// 	// 		return err
// 	// 	}
// 	// 	bc.MessageToRandomNode <- &msg
// 	// 	log.CLog().WithFields(logrus.Fields{
// 	// 		"Height": i,
// 	// 	}).Info("Request missing block")
// 	// }
// 	return nil
// }

func (bc *BlockChain) RemoveOrphanBlock() {
	TailTxs := bc.Tail().TransactionState
	bc.tailGroup.Range(func(key, value interface{}) bool {
		tail := value.(*Block)
		// var err error
		if bc.Lib().Header.Height >= tail.Header.Height {
			validBlock := bc.GetBlockByHeight(tail.Header.Height)
			if validBlock == nil {
				return true
			}
			for validBlock.Hash() != tail.Hash() {
				removableBlock := tail
				validBlock = bc.GetBlockByHash(validBlock.Header.ParentHash)
				tail = bc.GetBlockByHash(tail.Header.ParentHash)
				for _, tx := range removableBlock.Transactions {
					_tx := TailTxs.GetTransaction(tx.Hash)
					if _tx == nil {
						bc.TxPool.Put(tx)
					}
				}
				bc.Storage.Del(common.HashToBytes(removableBlock.Hash()))
				//already removed during for loop
				// if err != nil {
				// 	break
				// }
				if tail == nil {
					break
				}
			}
		}
		return true
	})
}

func (bc *BlockChain) RebuildBlockHeight() error {
	block := bc.Tail()
	if block.Header.Height == 0 {
		return nil
	}
	for {
		if bc.Lib().Header.Height+1 == block.Header.Height { //block.Hash() == bc.Lib.Hash()
			break
		}
		block = bc.GetBlockByHash(block.Header.ParentHash)
		if block == nil {
			return errors.New("ParentBlock is nil")
		}
		bc.Storage.Put(encodeBlockHeight(block.Header.Height), block.Header.Hash[:])
	}
	return nil
}

func (bc *BlockChain) putBlockToStorage(block *Block) error {
	encodedBytes, err := rlp.EncodeToBytes(block)
	if err != nil {
		return err
	}
	bc.Storage.Put(block.Header.Hash[:], encodedBytes)
	bc.Storage.Put(encodeBlockHeight(block.Header.Height), block.Header.Hash[:])
	return nil
}

func (bc *BlockChain) Lib() *Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.lib
}

func (bc *BlockChain) SetLib(block *Block) {
	bc.mu.Lock()
	bc.lib = block
	bc.Storage.Put([]byte(libKey), block.Header.Hash[:])
	bc.mu.Unlock()
	bc.LibCh <- struct{}{}
}

func (bc *BlockChain) LoadLibFromStorage() {
	hash, err := bc.Storage.Get([]byte(libKey))
	if err != nil {
		log.CLog().WithFields(logrus.Fields{}).Panic(err)
	}
	block := bc.GetBlockByHash(common.BytesToHash(hash))
	if block == nil {
		log.CLog().WithFields(logrus.Fields{}).Panic("Block is nil")
	}
	block.AccountState, err = NewAccountStateRootHash(block.Header.AccountHash, bc.Storage)
	if err != nil {
		log.CLog().WithFields(logrus.Fields{}).Panic(err)
	}
	block.TransactionState, err = NewTransactionStateRootHash(block.Header.TransactionHash, bc.Storage)
	if err != nil {
		log.CLog().WithFields(logrus.Fields{}).Panic(err)
	}
	consensusState, err := bc.Consensus.LoadState(block)
	if err != nil {
		log.CLog().WithFields(logrus.Fields{}).Panic(err)
	}
	block.SetConsensusState(consensusState)

	bc.SetLib(block)
}

func (bc *BlockChain) Tail() *Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.tail

}

func (bc *BlockChain) SetTail(block *Block) {
	if bc.tail == nil {
		bc.mu.Lock()
		bc.tail = block
		bc.Storage.Put([]byte(tailKey), block.Header.Hash[:])
		bc.mu.Unlock()
	}
	if block.Header.Height >= bc.tail.Header.Height {
		bc.mu.Lock()
		bc.tail = block
		bc.Storage.Put([]byte(tailKey), block.Header.Hash[:])
		log.CLog().WithFields(logrus.Fields{
			"Height": block.Header.Height,
		}).Debug("Tail")
		bc.mu.Unlock()
		//TODO: tow to call by channel
		bc.RebuildBlockHeight()
	}
}

func (bc *BlockChain) LoadTailFromStorage() {
	hash, err := bc.Storage.Get([]byte(tailKey))
	if err != nil {
		log.CLog().WithFields(logrus.Fields{}).Panic(err)
	}
	block := bc.GetBlockByHash(common.BytesToHash(hash))
	if block == nil {
		log.CLog().WithFields(logrus.Fields{}).Panic("Block is nil")
	}
	block.AccountState, err = NewAccountStateRootHash(block.Header.AccountHash, bc.Storage)
	if err != nil {
		log.CLog().WithFields(logrus.Fields{}).Panic(err)
	}
	block.TransactionState, err = NewTransactionStateRootHash(block.Header.TransactionHash, bc.Storage)
	if err != nil {
		log.CLog().WithFields(logrus.Fields{}).Panic(err)
	}

	consensusState, err := bc.Consensus.LoadState(block)
	if err != nil {
		log.CLog().WithFields(logrus.Fields{}).Panic(err)
	}
	block.SetConsensusState(consensusState)

	bc.SetTail(block)
}

func (bc *BlockChain) RemoveTxInPool(block *Block) {
	for _, tx := range block.Transactions {
		bc.TxPool.Del(tx.Hash)
	}
}
