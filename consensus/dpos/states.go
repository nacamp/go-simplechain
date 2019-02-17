package dpos

import (
	"bytes"
	"errors"
	"math/big"
	"math/rand"
	"sort"

	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/core"
	"github.com/nacamp/go-simplechain/rlp"
	"github.com/nacamp/go-simplechain/trie"
)

type Candidate struct {
	Address common.Address
	Balance *big.Int
}

type DposState struct {
	Candidate *trie.Trie
	Miner     *trie.Trie
}

//TODO:prefix for candidate address
func (ds *DposState) Stake(candidate common.Address, amount *big.Int) error {
	encodedBytes, err := ds.Candidate.Get(candidate[:])
	if err != nil {
		if err == trie.ErrNotFound {
			encodedBytes, err := rlp.EncodeToBytes(amount)
			if err != nil {
				return err
			}
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
	ds.Candidate.Put(candidate[:], encodedBytes)
	return nil
}

func (ds *DposState) Unstake(candidate common.Address, amount *big.Int) error {
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
	ds.Candidate.Put(candidate[:], encodedBytes)
	return nil
}

func (ds *DposState) GetNewElectedTime(blockTime, electedTime uint64, cycle, round, totalMiners int) uint64 {
	if blockTime < electedTime+uint64(cycle*round*totalMiners) {
		return blockTime
	}
	return electedTime
}

func (ds *DposState) GetMinersAndElectedTime(blockTime, electedTime uint64, cycle, round, totalMiners int) (uint64, []common.Address, error) {
	if blockTime < electedTime+uint64(cycle*round*totalMiners) {
		iter, err := ds.Candidate.Iterator(nil)
		if err != nil {
			return 0, nil, err
		}
		exist, _ := iter.Next()
		candidates := []core.BasicAccount{}
		for exist {
			account := core.BasicAccount{Address: common.Address{}}

			encodedBytes1 := iter.Key()
			key := []byte{}
			rlp.NewStream(bytes.NewReader(encodedBytes1), 0).Decode(&key)
			account.Address = common.BytesToAddress(key)

			encodedBytes2 := iter.Value()
			value := new(big.Int)
			rlp.NewStream(bytes.NewReader(encodedBytes2), 0).Decode(value)
			account.Balance = value

			candidates = append(candidates, account)
			exist, err = iter.Next()
		}

		if len(candidates) < totalMiners {
			return 0, nil, errors.New("The number of candidated miner is smaller than the minimum miner number.")
		}

		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].Balance.Cmp(candidates[j].Balance) > 0
		})

		candidates = candidates[:totalMiners]
		candidateAddrs := []common.Address{}
		for _, v := range candidates {
			candidateAddrs = append(candidateAddrs, v.Address)
		}
		shuffle(candidateAddrs, int64(blockTime))
		return blockTime, candidateAddrs, nil
	}
	miners, err := ds.GetMiners(electedTime)
	if err != nil {
		return 0, nil, err
	}
	return electedTime, miners, nil

}

func (ds *DposState) PutElectedTime(blockHash common.Hash, time uint64) error {
	encodedBytes, err := rlp.EncodeToBytes(time)
	if err != nil {
		return err
	}
	ds.Miner.Put(blockHash[:], encodedBytes)
	return nil
}

func (ds *DposState) GetElectedTime(blockHash common.Hash) (uint64, error) {
	encodedBytes, err := ds.Candidate.Get(blockHash[:])
	if err != nil {
		return 0, err
	}
	electedTime := uint64(0)
	err = rlp.Decode(bytes.NewReader(encodedBytes), &electedTime)
	if err != nil {
		return 0, err
	}
	return electedTime, nil
}

func (ds *DposState) PutMiners(electedTime uint64, miners []common.Address) error {
	// miner := Miner{MinerGroup: minerGroup, SnapshotVoterHash: snapshotVoterHash}
	encodedBytes1, err := rlp.EncodeToBytes(electedTime)
	if err != nil {
		return err
	}
	encodedBytes2, err := rlp.EncodeToBytes(miners)
	if err != nil {
		return err
	}
	ds.Miner.Put(encodedBytes1, encodedBytes2)
	return nil
}

func (ds *DposState) GetMiners(electedTime uint64) ([]common.Address, error) {
	encodedBytes1, err := rlp.EncodeToBytes(electedTime)
	if err != nil {
		return nil, err
	}
	miner := []common.Address{}
	decodedBytes, _ := ds.Miner.Get(encodedBytes1)
	rlp.NewStream(bytes.NewReader(decodedBytes), 0).Decode(&miner)
	return miner, nil
}

func (ds *DposState) Clone() (*DposState, error) {
	tr1, err1 := ds.Candidate.Clone()
	if err1 != nil {
		return nil, err1
	}
	tr2, err2 := ds.Miner.Clone()
	if err2 != nil {
		return nil, err2
	}
	return &DposState{
		Candidate: tr1,
		Miner:     tr2,
	}, nil
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
