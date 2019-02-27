package dpos

import (
	"bytes"
	"errors"
	"math/big"
	"math/rand"
	"sort"

	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/core"
	"github.com/nacamp/go-simplechain/crypto"
	"github.com/nacamp/go-simplechain/log"
	"github.com/nacamp/go-simplechain/rlp"
	"github.com/nacamp/go-simplechain/storage"
	"github.com/nacamp/go-simplechain/trie"
	"github.com/sirupsen/logrus"
)

type Candidate struct {
	Address common.Address
	Balance *big.Int
}

type DposState struct {
	Candidate   *trie.Trie
	Miner       *trie.Trie
	Voter       *trie.Trie
	MinersHash  common.Hash
	ElectedTime uint64
}

//TODO:prefix for candidate address
func (ds *DposState) Stake(voter, candidate common.Address, amount *big.Int) error {
	encodedBytes, err := ds.Candidate.Get(candidate[:])
	if err != nil {
		if err == trie.ErrNotFound {
			encodedBytes, err := rlp.EncodeToBytes(amount)
			if err != nil {
				return err
			}
			ds.Voter.Put(voter[:], []byte{})
			ds.Candidate.Put(candidate[:], encodedBytes)
			return nil
		}
		return err
	}
	balance := new(big.Int)
	err = rlp.Decode(bytes.NewReader(encodedBytes), balance)
	if err != nil {
		return err
	}
	balance.Add(balance, amount)
	encodedBytes, err = rlp.EncodeToBytes(balance)
	if err != nil {
		return err
	}
	ds.Voter.Put(voter[:], []byte{})
	ds.Candidate.Put(candidate[:], encodedBytes)
	return nil
}

func (ds *DposState) Unstake(voter, candidate common.Address, amount *big.Int) error {
	encodedBytes, err := ds.Candidate.Get(candidate[:])
	if err != nil {
		if err == trie.ErrNotFound {
			return errors.New("Staking is insufficient for candidate")
		}
		return err
	}
	balance := new(big.Int)
	err = rlp.Decode(bytes.NewReader(encodedBytes), balance)
	if err != nil {
		return err
	}
	if balance.Cmp(amount) < 0 {
		return errors.New("Staking is insufficient for candidate")
	}
	balance.Sub(balance, amount)
	encodedBytes, err = rlp.EncodeToBytes(balance)
	if err != nil {
		return err
	}
	ds.Voter.Put(voter[:], []byte{})
	ds.Candidate.Put(candidate[:], encodedBytes)
	return nil
}

func (ds *DposState) GetNewElectedTime(parentElectedTime, now uint64, cycle, round, totalMiners int) uint64 {
	// electedTime, err := ds.GetElectedTime(parentBlockHash)
	if now < parentElectedTime+uint64(cycle*round*totalMiners) {
		return now
	}
	return parentElectedTime
}

func (ds *DposState) GetMiners(minerHash common.Hash) ([]common.Address, error) {
	miner := []common.Address{}
	decodedBytes, _ := ds.Miner.Get(minerHash[:])
	rlp.NewStream(bytes.NewReader(decodedBytes), 0).Decode(&miner)
	return miner, nil
}

func (ds *DposState) GetNewRoundMiners(electedTime uint64, totalMiners int) ([]common.Address, error) {
	iter, err := ds.Candidate.Iterator(nil)
	if err != nil {
		return nil, err
	}
	exist, _ := iter.Next()
	candidates := []core.BasicAccount{}
	for exist {
		account := core.BasicAccount{Address: common.Address{}}

		// encodedBytes1 := iter.Key()
		// key := new([]byte)
		// rlp.NewStream(bytes.NewReader(encodedBytes1), 0).Decode(key)
		account.Address = common.BytesToAddress(iter.Key())

		encodedBytes2 := iter.Value()
		value := new(big.Int)
		rlp.NewStream(bytes.NewReader(encodedBytes2), 0).Decode(value)
		account.Balance = value

		candidates = append(candidates, account)
		exist, err = iter.Next()
	}

	if len(candidates) < totalMiners {
		return nil, errors.New("The number of candidated miner is smaller than the minimum miner number.")
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Balance.Cmp(candidates[j].Balance) > 0
	})

	candidates = candidates[:totalMiners]
	candidateAddrs := []common.Address{}
	for _, v := range candidates {
		candidateAddrs = append(candidateAddrs, v.Address)
	}
	shuffle(candidateAddrs, int64(electedTime))
	return candidateAddrs, nil
}

func (ds *DposState) PutMiners(miners []common.Address) (hash common.Hash, err error) {
	encodedBytes, err := rlp.EncodeToBytes(miners)
	if err != nil {
		return common.Hash{}, err
	}
	hashBytes := crypto.Sha3b256(encodedBytes)
	ds.Miner.Put(hashBytes, encodedBytes)
	copy(hash[:], hashBytes)
	return hash, nil
}

func (ds *DposState) Put(blockNumber, electedTime uint64, minersHash common.Hash) error {
	vals := make([]byte, 0)
	keyEncodedBytes, err := rlp.EncodeToBytes(blockNumber)
	if err != nil {
		return err
	}
	encodedTimeBytes, err := rlp.EncodeToBytes(electedTime)
	if err != nil {
		return err
	}

	vals = append(vals, ds.Candidate.RootHash()...)
	vals = append(vals, ds.Voter.RootHash()...)
	vals = append(vals, minersHash[:]...)
	vals = append(vals, encodedTimeBytes...)
	_, err = ds.Miner.Put(crypto.Sha3b256(keyEncodedBytes), vals)
	if err != nil {
		return err
	}

	return nil
}

/* return candidateHash, minersHash, electedTime*/
func (ds *DposState) Get(blockNumber uint64) (common.Hash, common.Hash, common.Hash, uint64, error) {
	keyEncodedBytes, err := rlp.EncodeToBytes(blockNumber)
	if err != nil {
		return common.Hash{}, common.Hash{}, common.Hash{}, 0, err
	}
	//TODO: check minimum key size
	encbytes, err := ds.Miner.Get(crypto.Sha3b256(keyEncodedBytes))
	if err != nil {
		return common.Hash{}, common.Hash{}, common.Hash{}, 0, err
	}
	if len(encbytes) < common.HashLength*3 {
		return common.Hash{}, common.Hash{}, common.Hash{}, 0, errors.New("Bytes lenght must be more than 64 bits")
	}

	electedTime := uint64(0)
	err = rlp.Decode(bytes.NewReader(encbytes[common.HashLength*3:]), &electedTime)
	if err != nil {
		return common.Hash{}, common.Hash{}, common.Hash{}, 0, err
	}
	return common.BytesToHash(encbytes[:common.HashLength]),
		common.BytesToHash(encbytes[common.HashLength : common.HashLength*2]),
		common.BytesToHash(encbytes[common.HashLength*2 : common.HashLength*3]),
		electedTime, nil
}

func (ds *DposState) RootHash() (hash common.Hash) {
	copy(hash[:], ds.Miner.RootHash())
	return hash
}

func (ds *DposState) Clone() (core.ConsensusState, error) {
	tr1, err1 := ds.Candidate.Clone()
	if err1 != nil {
		return nil, err1
	}
	tr2, err2 := ds.Miner.Clone()
	if err2 != nil {
		return nil, err2
	}
	tr3, err3 := ds.Voter.Clone()
	if err3 != nil {
		return nil, err3
	}
	return &DposState{
		Candidate:   tr1,
		Miner:       tr2,
		Voter:       tr3,
		MinersHash:  ds.MinersHash,
		ElectedTime: ds.ElectedTime,
	}, nil
}

func (cs *DposState) ExecuteTransaction(block *core.Block, txIndex int, account *core.Account) (err error) {

	tx := block.Transactions[txIndex]
	amount := new(big.Int)
	err = rlp.Decode(bytes.NewReader(tx.Payload.Data), amount)
	if err != nil {
		return err
	}
	if tx.Payload.Code == core.TxCVoteStake {
		err = account.Stake(tx.To, amount)
		if err != nil {
			return err
		}
		return cs.Stake(account.Address, tx.To, amount)
	} else if tx.Payload.Code == core.TxCVoteUnStake {
		err = account.UnStake(tx.To, amount)
		if err != nil {
			return err
		}
		return cs.Unstake(account.Address, tx.To, amount)
	}
	return nil
}

// func (ds *DposState) ExecuteTransaction(tx *core.Transaction, account *core.Account) (err error) {
// 	amount := new(big.Int)
// 	err = rlp.Decode(bytes.NewReader(tx.Payload.Data), amount)
// 	if err != nil {
// 		return err
// 	}
// 	if tx.Payload.Code == core.TxCVoteStake {
// 		err = account.Stake(tx.To, amount)
// 		if err != nil {
// 			return err
// 		}
// 		return ds.Stake(account.Address, tx.To, amount)
// 	} else if tx.Payload.Code == core.TxCVoteUnStake {
// 		err = account.UnStake(tx.To, amount)
// 		if err != nil {
// 			return err
// 		}
// 		return ds.Unstake(account.Address, tx.To, amount)
// 	}
// 	return nil
// }

/* Make new state by rootHash and initialized by blockNumber*/
func NewInitState(rootHash common.Hash, blockNumber uint64, storage storage.Storage) (*DposState, error) {
	var rootHashByte []byte
	if rootHash == (common.Hash{}) {
		rootHashByte = nil
	} else {
		rootHashByte = rootHash[:]
	}

	tr, err := trie.NewTrie(rootHashByte, storage, false)
	if err != nil {
		log.CLog().WithFields(logrus.Fields{"BlockNumber": blockNumber}).Panic(err)
		//return nil, err
	}

	state := new(DposState)
	state.Miner = tr
	candidateHash, votersHash, minersHash, electedTime, err := state.Get(blockNumber)
	if err != nil {
		if err == trie.ErrNotFound {
			tr2, err := trie.NewTrie(nil, storage, false)
			if err != nil {
				log.CLog().WithFields(logrus.Fields{}).Panic(err)
			}
			state.Candidate = tr2
			tr3, err := trie.NewTrie(nil, storage, false)
			if err != nil {
				log.CLog().WithFields(logrus.Fields{}).Panic(err)
			}
			state.Voter = tr3
			return state, nil
		}
		//return nil, err
		log.CLog().WithFields(logrus.Fields{}).Panic(err)
	}

	tr2, err := trie.NewTrie(candidateHash[:], storage, false)
	if err != nil {
		log.CLog().WithFields(logrus.Fields{}).Panic(err)
	}
	state.Candidate = tr2
	tr3, err := trie.NewTrie(votersHash[:], storage, false)
	if err != nil {
		log.CLog().WithFields(logrus.Fields{}).Panic(err)
	}
	state.Voter = tr3
	state.MinersHash = minersHash
	state.ElectedTime = electedTime
	return state, nil
}

func shuffle(slice []common.Address, seed int64) {
	r := rand.New(rand.NewSource(seed))
	for len(slice) > 0 {
		n := len(slice)
		randIndex := r.Intn(n)
		slice[n-1], slice[randIndex] = slice[randIndex], slice[n-1]
		slice = slice[:n-1]
	}
}

// func (ds *DposState) PutElectedTime(blockHash common.Hash, time uint64) error {
// 	encodedBytes, err := rlp.EncodeToBytes(time)
// 	if err != nil {
// 		return err
// 	}
// 	ds.Miner.Put(blockHash[:], encodedBytes)
// 	return nil
// }

// func (ds *DposState) GetElectedTime(blockHash common.Hash) (uint64, error) {
// 	encodedBytes, err := ds.Candidate.Get(blockHash[:])
// 	if err != nil {
// 		return 0, err
// 	}
// 	electedTime := uint64(0)
// 	err = rlp.Decode(bytes.NewReader(encodedBytes), &electedTime)
// 	if err != nil {
// 		return 0, err
// 	}
// 	return electedTime, nil
// }

// func (ds *DposState) PutMiners(electedTime uint64, miners []common.Address) error {
// 	// miner := Miner{MinerGroup: minerGroup, SnapshotVoterHash: snapshotVoterHash}
// 	encodedBytes1, err := rlp.EncodeToBytes(electedTime)
// 	if err != nil {
// 		return err
// 	}
// 	encodedBytes2, err := rlp.EncodeToBytes(miners)
// 	if err != nil {
// 		return err
// 	}
// 	ds.Miner.Put(encodedBytes1, encodedBytes2)
// 	return nil
// }

/*
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

func (ds *DposState) GetMiners(minerHash common.Hash) ([]common.Address, error) {
	// encodedBytes1, err := rlp.EncodeToBytes(electedTime)
	// if err != nil {
	// 	return nil, err
	// }
	miner := []common.Address{}
	decodedBytes, _ := ds.Miner.Get(minerHash[:])
	rlp.NewStream(bytes.NewReader(decodedBytes), 0).Decode(&miner)
	return miner, nil
}
*/

// func (ds *DposState) GetMinerss(newRound bool, electedTime uint64, totalMiners int) ([]common.Address, error) {
// 	if newRound {
// 		iter, err := ds.Candidate.Iterator(nil)
// 		if err != nil {
// 			return nil, err
// 		}
// 		exist, _ := iter.Next()
// 		candidates := []core.BasicAccount{}
// 		for exist {
// 			account := core.BasicAccount{Address: common.Address{}}

// 			encodedBytes1 := iter.Key()
// 			key := []byte{}
// 			rlp.NewStream(bytes.NewReader(encodedBytes1), 0).Decode(&key)
// 			account.Address = common.BytesToAddress(key)

// 			encodedBytes2 := iter.Value()
// 			value := new(big.Int)
// 			rlp.NewStream(bytes.NewReader(encodedBytes2), 0).Decode(value)
// 			account.Balance = value

// 			candidates = append(candidates, account)
// 			exist, err = iter.Next()
// 		}

// 		if len(candidates) < totalMiners {
// 			return nil, errors.New("The number of candidated miner is smaller than the minimum miner number.")
// 		}

// 		sort.Slice(candidates, func(i, j int) bool {
// 			return candidates[i].Balance.Cmp(candidates[j].Balance) > 0
// 		})

// 		candidates = candidates[:totalMiners]
// 		candidateAddrs := []common.Address{}
// 		for _, v := range candidates {
// 			candidateAddrs = append(candidateAddrs, v.Address)
// 		}
// 		shuffle(candidateAddrs, int64(electedTime))
// 		return candidateAddrs, nil
// 	}
// 	miners, err := ds.GetMiners(electedTime)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return miners, nil

// }

// func (ds *DposState) GetMinersAndElectedTime(blockTime, electedTime uint64, cycle, round, totalMiners int) (uint64, []common.Address, error) {
// 	if blockTime < electedTime+uint64(cycle*round*totalMiners) {
// 		iter, err := ds.Candidate.Iterator(nil)
// 		if err != nil {
// 			return 0, nil, err
// 		}
// 		exist, _ := iter.Next()
// 		candidates := []core.BasicAccount{}
// 		for exist {
// 			account := core.BasicAccount{Address: common.Address{}}

// 			encodedBytes1 := iter.Key()
// 			key := []byte{}
// 			rlp.NewStream(bytes.NewReader(encodedBytes1), 0).Decode(&key)
// 			account.Address = common.BytesToAddress(key)

// 			encodedBytes2 := iter.Value()
// 			value := new(big.Int)
// 			rlp.NewStream(bytes.NewReader(encodedBytes2), 0).Decode(value)
// 			account.Balance = value

// 			candidates = append(candidates, account)
// 			exist, err = iter.Next()
// 		}

// 		if len(candidates) < totalMiners {
// 			return 0, nil, errors.New("The number of candidated miner is smaller than the minimum miner number.")
// 		}

// 		sort.Slice(candidates, func(i, j int) bool {
// 			return candidates[i].Balance.Cmp(candidates[j].Balance) > 0
// 		})

// 		candidates = candidates[:totalMiners]
// 		candidateAddrs := []common.Address{}
// 		for _, v := range candidates {
// 			candidateAddrs = append(candidateAddrs, v.Address)
// 		}
// 		shuffle(candidateAddrs, int64(blockTime))
// 		return blockTime, candidateAddrs, nil
// 	}
// 	miners, err := ds.GetMiners(electedTime)
// 	if err != nil {
// 		return 0, nil, err
// 	}
// 	return electedTime, miners, nil

// }
