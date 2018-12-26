package consensus

import (
	"bytes"
	"encoding/json"
	"reflect"
	"sort"

	"github.com/najimmy/go-simplechain/common"
	"github.com/najimmy/go-simplechain/common/hexutil"
	"github.com/najimmy/go-simplechain/storage"
)

type DoubleAddress [common.AddressLength * 2]byte

var (
	doubleAddressT = reflect.TypeOf(DoubleAddress{})
)

type Snapshot struct {
	BlockHash common.Hash                 `json:"hash"`
	Signers   map[common.Address]struct{} `json:"signers"`
	//voter address+candidate address
	Votes      map[DoubleAddress]VoteData       `json:"votes"`
	Candidates map[common.Address]CandidateData `json:"candidates"`
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

func NewSnapshot(hash common.Hash, signers []common.Address) *Snapshot {
	snap := &Snapshot{
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

func LoadSnapshot(db storage.Storage, hash common.Hash) (*Snapshot, error) {
	blob, err := db.Get(append([]byte("snap-"), hash[:]...))
	if err != nil {
		return nil, err
	}
	snap := new(Snapshot)
	if err := json.Unmarshal(blob, snap); err != nil {
		return nil, err
	}
	return snap, nil
}

func (s *Snapshot) Store(db storage.Storage) error {
	blob, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return db.Put(append([]byte("snap-"), s.BlockHash[:]...), blob)
}

func (s *Snapshot) Copy() *Snapshot {
	cpy := &Snapshot{
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

func (s *Snapshot) ValidVote(address common.Address, authorize bool) bool {
	_, signer := s.Signers[address]
	return (signer && !authorize) || (!signer && authorize)
}

func appendAddress(a common.Address, b common.Address) DoubleAddress {
	ba := append(a[:], b[:]...)
	var da DoubleAddress
	copy(da[0:], ba)
	return da
}

func (s *Snapshot) Cast(signer common.Address, address common.Address, authorize bool) bool {
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

func (s *Snapshot) Apply() {
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

func (s *Snapshot) SignerSlice() []common.Address {
	sigs := make([]common.Address, 0, len(s.Signers))
	for sig := range s.Signers {
		sigs = append(sigs, sig)
	}
	sort.Sort(signersAscending(sigs))
	return sigs
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
