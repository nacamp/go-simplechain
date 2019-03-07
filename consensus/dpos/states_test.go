package dpos

import (
	"bytes"
	"math/big"
	"reflect"
	"testing"

	"github.com/nacamp/go-simplechain/rlp"
	"github.com/nacamp/go-simplechain/tests"

	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/storage"
	"github.com/stretchr/testify/assert"
)

func candidate(state *DposState, address common.Address) (balance *big.Int) {
	encodedBytes, _ := state.Candidate.Get(address[:])
	balance = new(big.Int)
	_ = rlp.Decode(bytes.NewReader(encodedBytes), balance)
	return balance
}

func TestStakeUnstake(t *testing.T) {
	_storage, _ := storage.NewMemoryStorage()
	state, err := NewInitState(common.Hash{}, 0, _storage)
	assert.NoError(t, err)
	_ = state

	//var newAddr = "0x1df75c884f7f1d1537177a3a35e783236739a426ee649fa3e2d8aed598b4f29e838170e2"
	err = state.Stake(common.HexToAddress(tests.AddressHex0), common.HexToAddress(tests.AddressHex0), new(big.Int).SetUint64(0))
	assert.Error(t, err)
	err = state.Stake(common.HexToAddress(tests.AddressHex0), common.HexToAddress(tests.AddressHex0), new(big.Int).SetUint64(10))
	assert.NoError(t, err)
	state.Stake(common.HexToAddress(tests.AddressHex0), common.HexToAddress(tests.AddressHex2), new(big.Int).SetUint64(20))
	state.Stake(common.HexToAddress(tests.AddressHex1), common.HexToAddress(tests.AddressHex2), new(big.Int).SetUint64(30))
	assert.True(t, candidate(state, tests.Address2).Cmp(new(big.Int).SetUint64(50)) == 0)

	err = state.Unstake(tests.Address2, tests.Address2, new(big.Int).SetUint64(10))
	assert.NoError(t, err)
	err = state.Unstake(tests.Address2, tests.Address2, new(big.Int).SetUint64(50))
	assert.Error(t, err)
}

func TestStake(t *testing.T) {
	_storage, _ := storage.NewMemoryStorage()
	state, err := NewInitState(common.Hash{}, 0, _storage)
	assert.NoError(t, err)

	var newAddr = "0x1df75c884f7f1d1537177a3a35e783236739a426ee649fa3e2d8aed598b4f29e838170e2"
	state.Stake(tests.Address0, common.HexToAddress(newAddr), new(big.Int).SetUint64(10))
	_, err = state.GetNewRoundMiners(uint64(5), 3)
	assert.Error(t, err)
	state.Stake(tests.Address0, tests.Address0, new(big.Int).SetUint64(30))
	state.Stake(tests.Address0, tests.Address1, new(big.Int).SetUint64(40))
	state.Stake(tests.Address0, tests.Address2, new(big.Int).SetUint64(50))

	//test GetNewRoundMiners
	miners, _ := state.GetNewRoundMiners(uint64(5), 3)
	minerSize := 0
	for _, v := range miners {
		if v == tests.Address0 || v == tests.Address1 || v == tests.Address2 {
			minerSize++
		}
	}
	assert.Equal(t, 3, len(miners))
	assert.Equal(t, 3, minerSize)

	//test PutMiners and GetMiners
	minersHash, _ := state.PutMiners(miners)
	miners2, _ := state.GetMiners(minersHash)
	assert.True(t, reflect.DeepEqual(miners, miners2))

	//test Put and Get
	state.Put(1, 5, minersHash)
	stateHash, err := state.Get(1)
	assert.Equal(t, state.Candidate.RootHash(), stateHash.Candidate)
	assert.Equal(t, state.Voter.RootHash(), stateHash.Voter)
	assert.Equal(t, minersHash, common.BytesToHash(stateHash.Miner))
	assert.Equal(t, uint64(5), stateHash.ElectedTime)

	//test NewInitState
	state2, err := NewInitState(state.RootHash(), 1, _storage)
	assert.Equal(t, state.Candidate.RootHash(), state2.Candidate.RootHash())
	assert.Equal(t, state.Voter.RootHash(), state2.Voter.RootHash())
	assert.Equal(t, minersHash, state2.MinersHash)
	assert.Equal(t, uint64(5), state2.ElectedTime)

	//TODO test
	//ExecuteTransaction in blockchain
}

func TestGetNewElectedTime(t *testing.T) {
	assert.Equal(t, uint64(0), GetNewElectedTime(0, 26, 3, 3, 3))
	assert.Equal(t, uint64(27), GetNewElectedTime(0, 27, 3, 3, 3))
}
