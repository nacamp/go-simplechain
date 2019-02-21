package core

import (
	"bytes"
	"encoding/binary"
	"errors"
	"sort"
	"sync"

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

var GenesisCoinbaseAddress = string("0xc6d40a9bf9fe9d90019511a2147dc0958657da97463ca59d2594d5536dcdfd30ed93707d")

type BlockChain struct {
	mu                  sync.RWMutex
	GenesisBlock        *Block
	futureBlocks        *lru.Cache
	Storage             storage.Storage
	TxPool              *TransactionPool
	Consensus           Consensus
	Lib                 *Block
	Tail                *Block
	MessageToRandomNode chan *net.Message
	NewTXMessage        chan *Transaction
	tailGroup           *sync.Map

	//poa
	Signers []common.Address
}

func NewBlockChain(storage storage.Storage) *BlockChain {
	futureBlocks, _ := lru.New(maxFutureBlocks)
	bc := BlockChain{
		Storage:             storage,
		futureBlocks:        futureBlocks,
		tailGroup:           new(sync.Map),
		MessageToRandomNode: make(chan *net.Message, 1),
		NewTXMessage:        make(chan *Transaction, 1),
	}
	return &bc
}

func (bc *BlockChain) Setup(consensus Consensus, voters []*Account) {
	consensus.AddBlockChain(bc)
	bc.Consensus = consensus
	err := bc.LoadBlockChainFromStorage()
	if err != nil {
		if err == storage.ErrKeyNotFound {
			err = bc.MakeGenesisBlock(voters)
			if err != nil {
				log.CLog().WithFields(logrus.Fields{
					"Error": err,
				}).Panic("MakeGenesisBlock")
			}
			bc.PutBlockByCoinbase(bc.GenesisBlock)
		} else {
			log.CLog().WithFields(logrus.Fields{
				"Error": err,
			}).Panic("")
		}
	} else {
		err = bc.LoadLibFromStorage()
		if err != nil {
			log.CLog().WithFields(logrus.Fields{
				"Error": err,
			}).Panic("LoadLibFromStorage")
		}
		err = bc.LoadTailFromStorage()
		if err != nil {
			log.CLog().WithFields(logrus.Fields{
				"Error": err,
			}).Panic("LoadTailFromStorage")
		}
	}
	bc.TxPool = NewTransactionPool()

}

func (bc *BlockChain) LoadBlockChainFromStorage() error {
	block := bc.GetBlockByHeight(0)
	if block == nil {
		return storage.ErrKeyNotFound
	}
	var err error
	//status
	block.AccountState, err = NewAccountStateRootHash(block.Header.AccountHash, bc.Storage)
	if err != nil {
		return err
	}
	block.TransactionState, err = NewTransactionStateRootHash(block.Header.TransactionHash, bc.Storage)
	if err != nil {
		return err
	}

	err = bc.Consensus.LoadConsensusStatus(block)
	if err != nil {
		return err
	}
	bc.GenesisBlock = block
	return nil

}

func (bc *BlockChain) MakeGenesisBlock(voters []*Account) error {
	common.FromHex(GenesisCoinbaseAddress)
	header := &Header{
		Coinbase: common.HexToAddress(GenesisCoinbaseAddress),
		Height:   0,
		Time:     0,
	}
	block := &Block{
		BaseBlock: BaseBlock{Header: header},
	}

	//AccountState
	accs, err := NewAccountState(bc.Storage)
	if err != nil {
		return err
	}
	account := Account{}
	copy(account.Address[:], common.FromHex(GenesisCoinbaseAddress))
	account.AddBalance(new(big.Int).SetUint64(100)) //FIXME: amount 0
	accs.PutAccount(&account)
	block.AccountState = accs
	header.AccountHash = accs.RootHash()

	//TransactionState
	txs, err := NewTransactionState(bc.Storage)
	if err != nil {
		return err
	}
	txs.PutTransaction(&Transaction{})
	block.TransactionState = txs
	header.TransactionHash = txs.RootHash()
	err = bc.Consensus.MakeGenesisBlock(block, voters)
	if err != nil {
		return err
	}
	bc.SetLib(bc.GenesisBlock)
	bc.SetTail(bc.GenesisBlock)
	return nil
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
	//FIXME: verify genesis block
	if block.Header.Height == uint64(0) {
		return nil
	}
	var err error
	parentBlock := bc.GetBlockByHash(block.Header.ParentHash)
	block.AccountState, err = NewAccountStateRootHash(parentBlock.Header.AccountHash, bc.Storage)
	if err != nil {
		return err
	}
	block.TransactionState, err = NewTransactionStateRootHash(parentBlock.Header.TransactionHash, bc.Storage)
	if err != nil {
		return err
	}

	block.ConsensusState, err = bc.Consensus.LoadState(parentBlock)
	if err != nil {
		return err
	}
	// TODO: parent maybe not have ConsensusState
	// block.ConsensusState, err = parentBlock.ConsensusState.Clone()

	bc.RewardForCoinbase(block)

	// if err := bc.Consensus.SaveMiners(block); err != nil {
	// 	return err
	// }

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

	if block.ConsensusState.RootHash() != block.Header.ConsensusHash {
		return errors.New("block.ConsensusState.RootHash() != block.Header.ConsensusHash")
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
	firstVote := true
	for _, tx := range block.Transactions {
		fromAccount := accs.GetAccount(tx.From)
		if fromAccount.Nonce+1 != tx.Nonce {
			return ErrTransactionNonce
		}
		fromAccount.Nonce += uint64(1)
		if len(tx.Payload) == 0 {
			toAccount := accs.GetAccount(tx.To)
			if err := fromAccount.SubBalance(tx.Amount); err != nil {
				return err
			}
			toAccount.AddBalance(tx.Amount)
			accs.PutAccount(toAccount)
		} else {
			if tx.From == block.Header.Coinbase && firstVote {
				firstVote = false
			} else {
				return errors.New("This tx is not validated")
			}
		}
		accs.PutAccount(fromAccount)
		txs.PutTransaction(tx)
	}
	return nil
}

func (bc *BlockChain) PutBlock(block *Block) error {
	var err error
	//1. verify block.hash
	if block.Hash() != block.CalcHash() {
		return errors.New("block.Hash() != block.CalcHash()")
	}

	//2.signer check
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

	//5. verify consen
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
	bc.futureBlocks.Add(block.Header.ParentHash, block)
	//FIXME: temporarily, must send hash
	if block.Header.Height > uint64(1) {
		msg, err := net.NewRLPMessage(net.MsgMissingBlock, block.Header.Height-uint64(1))
		if err != nil {
			return err
		}
		bc.MessageToRandomNode <- &msg
		log.CLog().WithFields(logrus.Fields{
			"Height": block.Header.Height - uint64(1),
		}).Info("Request missing block")
	}
	return nil
}

func encodeBlockHeight(number uint64) []byte {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, number)
	return enc
}

func (bc *BlockChain) NewBlockFromParent(parentBlock *Block) (block *Block, err error) {
	h := &Header{
		ParentHash: parentBlock.Hash(),
		Height:     parentBlock.Header.Height + 1,
	}
	block = &Block{
		BaseBlock: BaseBlock{Header: h},
	}
	//state
	//TODO: test in poa
	// err = bc.Consensus.CloneFromParentBlock(block, parentBlock)
	// if err != nil {
	// 	return nil, err
	// }

	block.ConsensusState, err = parentBlock.ConsensusState.Clone()
	if err != nil {
		return nil, err
	}

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

//TODO: use code temporarily
func (bc *BlockChain) RequestMissingBlock() error {
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
		return nil
	}
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	for i := bc.Tail.Header.Height + 1; i < uint64(keys[0]); i++ {
		msg, err := net.NewRLPMessage(net.MsgMissingBlock, uint64(i))
		if err != nil {
			return err
		}
		bc.MessageToRandomNode <- &msg
		log.CLog().WithFields(logrus.Fields{
			"Height": i,
		}).Info("Request missing block")
	}
	return nil
}

func (bc *BlockChain) RemoveOrphanBlock() {
	bc.mu.RLock()
	TailTxs := bc.Tail.TransactionState
	bc.mu.RUnlock()
	bc.tailGroup.Range(func(key, value interface{}) bool {
		tail := value.(*Block)
		// var err error
		if bc.Lib.Header.Height >= tail.Header.Height {
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
	block := bc.Tail
	if block.Header.Height == 0 {
		return nil
	}
	var err error
	for {
		if block.Hash() == bc.Lib.Hash() {
			break
		}
		block = bc.GetBlockByHash(block.Header.ParentHash)
		if err != nil {
			return err
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

func (bc *BlockChain) SetLib(block *Block) {
	bc.Lib = block
	bc.Storage.Put([]byte(libKey), block.Header.Hash[:])
}

func (bc *BlockChain) LoadLibFromStorage() error {
	hash, err := bc.Storage.Get([]byte(libKey))
	if err != nil {
		return err
	}
	block := bc.GetBlockByHash(common.BytesToHash(hash))
	if err != nil {
		return err
	}
	block.AccountState, err = NewAccountStateRootHash(block.Header.AccountHash, bc.Storage)
	if err != nil {
		return err
	}
	block.TransactionState, err = NewTransactionStateRootHash(block.Header.TransactionHash, bc.Storage)
	if err != nil {
		return err
	}
	err = bc.Consensus.LoadConsensusStatus(block)
	if err != nil {
		return err
	}

	bc.Lib = block
	return nil
}

func (bc *BlockChain) SetTail(block *Block) {
	if bc.Tail == nil {
		bc.Tail = block
		bc.Storage.Put([]byte(tailKey), block.Header.Hash[:])
	}
	if block.Header.Height >= bc.Tail.Header.Height {
		bc.mu.Lock()
		bc.Tail = block
		bc.mu.Unlock()
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
	block := bc.GetBlockByHash(common.BytesToHash(hash))
	if err != nil {
		return err
	}
	block.AccountState, err = NewAccountStateRootHash(block.Header.AccountHash, bc.Storage)
	if err != nil {
		return err
	}
	block.TransactionState, err = NewTransactionStateRootHash(block.Header.TransactionHash, bc.Storage)
	if err != nil {
		return err
	}
	err = bc.Consensus.LoadConsensusStatus(block)
	if err != nil {
		return err
	}

	bc.Tail = block
	return nil
}

func (bc *BlockChain) RemoveTxInPool(block *Block) {
	for _, tx := range block.Transactions {
		bc.TxPool.Del(tx.Hash)
	}
}
