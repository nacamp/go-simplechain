package poa

import (
	"testing"

	"github.com/nacamp/go-simplechain/tests"
	"github.com/nacamp/go-simplechain/trie"

	"github.com/stretchr/testify/assert"

	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/storage"
)

func signers(state *PoaState) []common.Address {
	addresses, _ := state.signers()
	return addresses
}

func voters(state *PoaState) [][]byte {
	doubleAddress := make([][]byte, 0)
	iter, err := state.Voter.Iterator(nil)
	if err != nil {
		return doubleAddress
	}
	exist, _ := iter.Next()
	for exist {
		doubleAddress = append(doubleAddress, iter.Key())
		exist, _ = iter.Next()
	}
	return doubleAddress
}

func TestState(t *testing.T) {
	_storage, _ := storage.NewMemoryStorage()
	state, err := NewInitState(common.Hash{}, 0, _storage)
	assert.NoError(t, err)
	_ = state

	state.Signer.Put(common.FromHex(tests.AddressHex0), []byte{})
	state.Signer.Put(common.FromHex(tests.AddressHex1), []byte{})
	state.Signer.Put(common.FromHex(tests.AddressHex2), []byte{})

	//test signers
	addresses, _ := state.signers()
	assert.Equal(t, common.FromHex(tests.AddressHex0), addresses[0][:])
	assert.Equal(t, common.FromHex(tests.AddressHex1), addresses[1][:])
	assert.Equal(t, common.FromHex(tests.AddressHex2), addresses[2][:])

	//test ValidVote
	var newAddr = "0x1df75c884f7f1d1537177a3a35e783236739a426ee649fa3e2d8aed598b4f29e838170e2"
	assert.False(t, state.ValidVote(common.HexToAddress(newAddr), false))
	assert.False(t, state.ValidVote(common.HexToAddress(tests.AddressHex0), true))
	assert.True(t, state.ValidVote(common.HexToAddress(newAddr), true))
	assert.True(t, state.ValidVote(common.HexToAddress(tests.AddressHex0), false))

	//test Vote & RefreshSigner
	//FIXME: ban self vote, check if voter is signer
	assert.False(t, state.Vote(common.HexToAddress(tests.AddressHex0), common.HexToAddress(newAddr), false))

	assert.True(t, state.Vote(common.HexToAddress(tests.AddressHex0), common.HexToAddress(newAddr), true))
	assert.Equal(t, 1, len(voters(state)))
	_ = state.RefreshSigner()
	assert.Equal(t, 3, len(signers(state)))

	assert.True(t, state.Vote(common.HexToAddress(tests.AddressHex1), common.HexToAddress(newAddr), true))
	assert.Equal(t, 2, len(voters(state)))
	_ = state.RefreshSigner()
	assert.Equal(t, 4, len(signers(state)))
	assert.Equal(t, 0, len(voters(state)))

	//test put & get
	err = state.Put(1)
	assert.NoError(t, err)
	signersHash, votersHash, err := state.Get(1)
	//if voters size is 0, return common.Hash{}
	assert.Equal(t, common.Hash{}, votersHash)
	//if votersHash is common.Hash{}, use nil
	tr, _ := trie.NewTrie(nil, _storage, false)
	assert.Equal(t, state.Voter.RootHash(), tr.RootHash())
	tr, _ = trie.NewTrie(signersHash[:], _storage, false)
	assert.Equal(t, state.Signer.RootHash(), tr.RootHash())

	//test NewInitState
	state2, err := NewInitState(state.RootHash(), 1, _storage)
	assert.NoError(t, err)
	assert.Equal(t, state.Signer.RootHash(), state2.Signer.RootHash())

	//test sort and size in GetMiners
	signers, err := state.GetMiners()
	assert.Equal(t, "0x1df75c884f7f1d1537177a3a35e783236739a426ee649fa3e2d8aed598b4f29e838170e2", common.AddressToHex(signers[0]))
	assert.Equal(t, tests.AddressHex0, common.AddressToHex(signers[1]))
	assert.Equal(t, tests.AddressHex1, common.AddressToHex(signers[2]))
	assert.Equal(t, tests.AddressHex2, common.AddressToHex(signers[3]))
	assert.Equal(t, 4, len(signers))

	//test Clone
	state3, err := state.Clone()
	state4 := state3.(*PoaState)
	assert.Equal(t, state.Signer.RootHash(), state4.Signer.RootHash())
	assert.Equal(t, state.Voter.RootHash(), state4.Voter.RootHash())
	assert.Equal(t, state.Snapshot.RootHash(), state4.Snapshot.RootHash())
}
