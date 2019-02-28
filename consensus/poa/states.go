package poa

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"sort"

	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/common/hexutil"
	"github.com/nacamp/go-simplechain/core"
	"github.com/nacamp/go-simplechain/crypto"
	"github.com/nacamp/go-simplechain/log"
	"github.com/nacamp/go-simplechain/rlp"
	"github.com/nacamp/go-simplechain/storage"
	"github.com/nacamp/go-simplechain/trie"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/sha3"
)

type DoubleAddress [common.AddressLength * 2]byte

var (
	doubleAddressT = reflect.TypeOf(DoubleAddress{})
)

type PoaState struct {
	BlockHash common.Hash                 `json:"-"`
	Signers   map[common.Address]struct{} `json:"signers"`
	//voter address+candidate address
	Votes      map[DoubleAddress]VoteData       `json:"votes"`
	Candidates map[common.Address]CandidateData `json:"candidates"`

	Snapshot *trie.Trie
	// Candidate *trie.Trie
	Voter     *trie.Trie
	Signer    *trie.Trie
	firstVote bool
}
type Vote struct {
	Signer common.Address `json:"signer"`
	VoteData
}
type VoteData struct {
	Address   common.Address `json:"address"`
	Authorize bool           `json:"authorize"`
}

type Candidate struct {
	Address common.Address `json:"address"`
	CandidateData
}

type CandidateData struct {
	Authorize bool `json:"authorize"`
	Votes     int  `json:"votes"`
}

func NewSnapshot(hash common.Hash, signers []common.Address) *PoaState {
	snap := &PoaState{
		BlockHash:  hash,
		Signers:    make(map[common.Address]struct{}),
		Votes:      make(map[DoubleAddress]VoteData),
		Candidates: make(map[common.Address]CandidateData),
	}
	for _, signer := range signers {
		snap.Signers[signer] = struct{}{}
	}
	return snap
}

func (cs *PoaState) Put(blockNumber uint64) error {
	vals := make([]byte, 0)
	keyEncodedBytes, err := rlp.EncodeToBytes(blockNumber)
	if err != nil {
		return err
	}

	vals = append(vals, cs.Signer.RootHash()...)
	vals = append(vals, cs.Voter.RootHash()...)
	_, err = cs.Snapshot.Put(crypto.Sha3b256(keyEncodedBytes), vals)
	if err != nil {
		return err
	}

	return nil
}

func (cs *PoaState) Get(blockNumber uint64) (common.Hash, common.Hash, error) {
	keyEncodedBytes, err := rlp.EncodeToBytes(blockNumber)
	if err != nil {
		return common.Hash{}, common.Hash{}, err
	}
	//TODO: check minimum key size
	encbytes, err := cs.Snapshot.Get(crypto.Sha3b256(keyEncodedBytes))
	if err != nil {
		return common.Hash{}, common.Hash{}, err
	}
	if len(encbytes) < common.HashLength*2 {
		return common.Hash{}, common.Hash{}, errors.New("Bytes lenght must be more than 64 bits")
	}

	return common.BytesToHash(encbytes[:common.HashLength]),
		common.BytesToHash(encbytes[common.HashLength:]),
		nil
}

/*
초기값 dpos 스타일
	//TODO: who voter?
	for _, v := range voters {
		state.Stake(v.Address, v.Address, v.Balance)
	}
*/

/* Make new state by rootHash and initialized by blockNumber*/
func NewInitState(rootHash common.Hash, blockNumber uint64, storage storage.Storage) (state *PoaState, err error) {
	var rootHashByte []byte
	if rootHash == (common.Hash{}) {
		rootHashByte = nil
	} else {
		rootHashByte = rootHash[:]
	}

	tr, err := trie.NewTrie(rootHashByte, storage, false)
	if err != nil {
		return nil, err
	}

	state = new(PoaState)
	state.Snapshot = tr
	signersHash, votersHash, err := state.Get(blockNumber)
	if err != nil {
		if err == trie.ErrNotFound {
			tr2, err := trie.NewTrie(nil, storage, false)
			state.Signer = tr2
			tr3, err := trie.NewTrie(nil, storage, false)
			state.Voter = tr3
			return state, err
		}
		return nil, err
	}

	tr2, err := trie.NewTrie(signersHash[:], storage, false)
	state.Signer = tr2
	tr3, err := trie.NewTrie(votersHash[:], storage, false)
	state.Voter = tr3
	state.firstVote = true
	return state, err
}

func LoadSnapshot(db storage.Storage, hash common.Hash) (*PoaState, error) {
	blob, err := db.Get(append([]byte("snap-"), hash[:]...))
	if err != nil {
		return nil, err
	}
	snap := new(PoaState)
	if err := json.Unmarshal(blob, snap); err != nil {
		return nil, err
	}
	return snap, nil
}

func (s *PoaState) Store(db storage.Storage) error {
	blob, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return db.Put(append([]byte("snap-"), s.BlockHash[:]...), blob)
}

func (s *PoaState) CalcHash() (hash common.Hash) {
	blob, _ := json.Marshal(s)
	hasher := sha3.New256()
	hasher.Write(blob)
	hasher.Sum(hash[:0])
	return hash
}

func (s *PoaState) Copy() *PoaState {
	cpy := &PoaState{
		BlockHash:  s.BlockHash,
		Signers:    make(map[common.Address]struct{}),
		Votes:      make(map[DoubleAddress]VoteData),
		Candidates: make(map[common.Address]CandidateData),
	}
	for signer := range s.Signers {
		cpy.Signers[signer] = struct{}{}
	}
	for byteAddress, vote := range s.Votes {
		cpy.Votes[byteAddress] = VoteData{
			Address:   vote.Address,
			Authorize: vote.Authorize,
		}
	}
	for address, candidate := range s.Candidates {
		cpy.Candidates[address] = CandidateData{
			Authorize: candidate.Authorize,
			Votes:     candidate.Votes,
		}
	}
	return cpy
}

func (cs *PoaState) ValidVote2(address common.Address, join bool) bool {
	_, err := cs.Signer.Get(address[:])
	if err != nil {
		return join
	}
	return !join
}

func (s *PoaState) ValidVote(address common.Address, authorize bool) bool {
	_, signer := s.Signers[address]
	return (signer && !authorize) || (!signer && authorize)
}

func appendAddress(a common.Address, b common.Address) DoubleAddress {
	ba := append(a[:], b[:]...)
	var da DoubleAddress
	copy(da[0:], ba)
	return da
}

func (cs *PoaState) Cast2(signer, candidate common.Address, authorize bool) bool {
	// Ensure the vote is meaningful
	if !cs.ValidVote(candidate, authorize) {
		return false
	}
	cs.Voter.Put(append(signer[:], candidate[:]...), []byte{0x0})
	return true
}

func (s *PoaState) Cast(signer common.Address, address common.Address, authorize bool) bool {
	// Ensure the vote is meaningful
	if !s.ValidVote(address, authorize) {
		return false
	}
	key := appendAddress(signer, address)
	if _, ok := s.Votes[key]; !ok {
		s.Votes[key] = VoteData{
			Address:   address,
			Authorize: authorize,
		}
		if old, ok := s.Candidates[address]; ok {
			old.Votes++
			s.Candidates[address] = old
		} else {
			s.Candidates[address] = CandidateData{Authorize: authorize, Votes: 1}
		}
		return true
	}
	return false
}

func (cs *PoaState) Apply2() {
	targetAddress := common.Address{}

	iter, err := cs.Voter.Iterator(nil)
	if err != nil {
		return //0, nil, err
	}
	candidate := make(map[common.Address]int)
	exist, _ := iter.Next()
	// candidates := []core.BasicAccount{}

	for exist {
		c := common.BytesToAddress(iter.Key()[common.HashLength:])
		_, v := candidate[c]
		if v {
			candidate[c] += 1
		} else {
			candidate[c] = 1
		}
		if candidate[c] > len(cs.Signers)/2 {
			_, err := cs.Signer.Get(c[:])
			if err != nil {
				if err == trie.ErrNotFound {
					cs.Signer.Put(c[:], []byte{})
				} else {
					log.CLog().WithFields(logrus.Fields{}).Panic(err)
				}
			} else {
				cs.Signer.Del(c[:])
			}
			targetAddress = c
			break
		}
		exist, err = iter.Next()
	}
	if len(targetAddress) > 0 {
		iter, _ := cs.Voter.Iterator(nil)
		exist, _ := iter.Next()
		for exist {
			k := iter.Key()
			if common.BytesToAddress(k[:common.HashLength]) == targetAddress || common.BytesToAddress(k[common.HashLength:]) == targetAddress {
				cs.Voter.Del(iter.Key())
			}
			exist, err = iter.Next()
		}
	}
}

func (s *PoaState) Apply() {
	devictedAddress := common.Address{}
	for address, candidate := range s.Candidates {
		if candidate.Votes > len(s.Signers)/2 {
			if candidate.Authorize {
				//join
				s.Signers[address] = struct{}{}
			} else {
				//evict
				delete(s.Signers, address)
				devictedAddress = address
			}
		}
	}

	if devictedAddress != (common.Address{}) {
		for address, candidate := range s.Candidates {
			if _, ok := s.Votes[appendAddress(devictedAddress, address)]; ok {
				//TODO: test
				candidate.Votes--
				s.Candidates[address] = candidate
				delete(s.Votes, appendAddress(devictedAddress, address))
			}
		}
		for address := range s.Signers {
			if _, ok := s.Votes[appendAddress(address, devictedAddress)]; ok {
				delete(s.Votes, appendAddress(address, devictedAddress))
			}
		}
		delete(s.Candidates, devictedAddress)
	}
}

type signersAscending []common.Address

func (s signersAscending) Len() int           { return len(s) }
func (s signersAscending) Less(i, j int) bool { return bytes.Compare(s[i][:], s[j][:]) < 0 }
func (s signersAscending) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func (s *PoaState) SignerSlice() []common.Address {
	sigs := make([]common.Address, 0, len(s.Signers))
	for sig := range s.Signers {
		sigs = append(sigs, sig)
	}
	sort.Sort(signersAscending(sigs))
	return sigs
}

func (cs *PoaState) GetMiners() (signers []common.Address, err error) {
	signers = []common.Address{}
	iter, err := cs.Signer.Iterator(nil)
	if err != nil {
		return nil, err
	}
	exist, _ := iter.Next()
	for exist {
		k := iter.Key()
		signers = append(signers, common.BytesToAddress(k))
		exist, err = iter.Next()
	}
	return signers, nil
}

// MarshalText returns the hex representation of a.
func (a DoubleAddress) MarshalText() ([]byte, error) {
	return hexutil.Bytes(a[:]).MarshalText()
}

// UnmarshalText parses a hash in hex syntax.
func (a *DoubleAddress) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedText("DoubleAddress", input, a[:])
}

// UnmarshalJSON parses a hash in hex syntax.
func (a *DoubleAddress) UnmarshalJSON(input []byte) error {
	return hexutil.UnmarshalFixedJSON(doubleAddressT, input, a[:])
}

func (cs *PoaState) Clone() (core.ConsensusState, error) {
	tr1, err1 := cs.Voter.Clone()
	if err1 != nil {
		return nil, err1
	}
	tr2, err2 := cs.Signer.Clone()
	if err2 != nil {
		return nil, err2
	}
	tr3, err3 := cs.Snapshot.Clone()
	if err3 != nil {
		return nil, err3
	}
	return &PoaState{
		Voter:     tr1,
		Signer:    tr2,
		Snapshot:  tr3,
		firstVote: true,
	}, nil
}

func (cs *PoaState) ExecuteTransaction(block *core.Block, txIndex int, account *core.Account) (err error) {
	tx := block.Transactions[txIndex]
	if tx.From == block.Header.Coinbase && cs.firstVote {
		cs.firstVote = false
	} else {
		return errors.New("This tx is not validated")
	}
	if tx.Payload.Code == core.TxCVoteStake {
		cs.Cast2(tx.From, tx.To, true)
	} else if tx.Payload.Code == core.TxCVoteUnStake {
		cs.Cast2(tx.From, tx.To, false)
	}
	return nil
}

func (cs *PoaState) RootHash() (hash common.Hash) {
	copy(hash[:], cs.Snapshot.RootHash())
	return hash
}
