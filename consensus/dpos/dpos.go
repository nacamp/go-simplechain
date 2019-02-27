package dpos

import (
	"errors"
	"time"

	"github.com/nacamp/go-simplechain/account"
	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/trie"

	"github.com/nacamp/go-simplechain/core"
	"github.com/nacamp/go-simplechain/log"
	"github.com/nacamp/go-simplechain/net"
	"github.com/sirupsen/logrus"
)

//Demo 3 accounts
var GenesisCoinbaseAddress = string("0x036407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0")
var keystore = map[string]string{
	GenesisCoinbaseAddress: "0xe68fb0a479c495910c8351c3593667028b45d679f55ce22b0514c4a8a6bcbdd1",
	"0x03fdefdefbb2478f3d1ed3221d38b8bad6d939e50f17ffda40f0510b4d28506bd3": "0xf390e256b6ed8a1b283d3ea80b103b868c14c31e5b7114fc32fff21c4cb263eb",
	"0x03e864b08b08f632c61c6727cde0e23d125f7784b5a5a188446fc5c91ffa51faa1": "0xb385aca81e134722cca902bf85443528c3d3a783cf54008cfc34a2ca563fc5b6",
}

type Dpos struct {
	bc           *core.BlockChain
	coinbase     common.Address
	wallet       *account.Wallet
	enableMining bool
	streamPool   *net.PeerStreamPool
}

func NewDpos(streamPool *net.PeerStreamPool) *Dpos {
	return &Dpos{streamPool: streamPool}
}

func (cs *Dpos) Setup(address common.Address, wallet *account.Wallet, period int) {
	cs.enableMining = true
	cs.coinbase = address
	cs.wallet = wallet
}

func (cs *Dpos) MakeBlock(now uint64) *core.Block {
	bc := cs.bc
	//TODO: check after 3 seconds(block creation) and 3 seconds(mining order)
	//Fix: when ticker is 1 second, server mining...
	turn := (now % 9) / 3
	block, err := bc.NewBlockFromTail()
	if err != nil {
		log.CLog().Warning(err)
	}
	block.Header.Time = now
	state := block.ConsensusState.(*DposState)
	electedTime := state.GetNewElectedTime(state.ElectedTime, block.Header.Time, 3, 3, 3)

	//electedTime := cs.state.GetNewElectedTime(bc.Tail.Hash(), now, 3, 3, 3)
	var minerGroup []common.Address
	if electedTime != now {
		minerGroup, err = state.GetMiners(state.MinersHash)
		//minerGroup, _, err := block.MinerState.GetMinerGroup(bc, block)
		if err != nil {
			log.CLog().Warning(err)
		}
	}
	if electedTime == now || minerGroup[turn] == cs.coinbase {
		parent := bc.GetBlockByHash(bc.Tail.Header.ParentHash)

		if (parent != nil) && (now-parent.Header.Time < 3) { //(3 * 3)
			log.CLog().WithFields(logrus.Fields{
				"address": common.AddressToHex(cs.coinbase),
			}).Warning("Interval is short")
			return nil
		}

		log.CLog().WithFields(logrus.Fields{
			"address": common.BytesToHex(cs.coinbase[:]),
		}).Debug("my turn")
		block.Header.Coinbase = cs.coinbase
		//block.Header.SnapshotVoterTime = bc.Tail.Header.SnapshotVoterTime // voterBlock.Header.Time
		//because PutMinerState recall GetMinerGroup , here assign  bc.Tail.Header.SnapshotVoterTime , not voterBlock.Header.Time

		//TODO: check double spending ?
		block.Transactions = make([]*core.Transaction, 0)
		accs := block.AccountState
		for i := 0; i < bc.TxPool.Len(); i++ {
			tx := bc.TxPool.Pop()
			if tx == nil {
				break
			}
			//TODO: remove code duplicattion in ExecuteTransaction
			fromAccount := accs.GetAccount(tx.From)
			//TODO: check at txpool
			if fromAccount == nil {
				log.CLog().WithFields(logrus.Fields{
					"Address": common.AddressToHex(tx.From),
				}).Warning("Not found account")
			} else if fromAccount.Nonce+1 == tx.Nonce {
				block.Transactions = append(block.Transactions, tx)
			} else if fromAccount.Nonce+1 < tx.Nonce {
				//use in future
				bc.TxPool.Put(tx)
			} else {
				log.CLog().WithFields(logrus.Fields{
					"Address": common.AddressToHex(tx.From),
				}).Warning("cannot accept a transaction with wrong nonce")
			}
		}
		bc.RewardForCoinbase(block)
		bc.ExecuteTransaction(block)
		block.Header.AccountHash = block.AccountState.RootHash()
		block.Header.TransactionHash = block.TransactionState.RootHash()
		cs.SaveState(block)
		if err := cs.Verify(block); err != nil {
			log.CLog().WithFields(logrus.Fields{
				"address": common.BytesToHex(cs.coinbase[:]),
			}).Debug("not my turn")
		}
		block.Header.ConsensusHash = state.RootHash()
		block.MakeHash()
		return block
	} else {
		log.CLog().WithFields(logrus.Fields{
			"address": common.BytesToHex(cs.coinbase[:]),
		}).Debug("not my turn")
		return nil
	}
}

func (dpos *Dpos) Start() {
	if dpos.enableMining {
		go dpos.loop()
	}
}

func (cs *Dpos) loop() {
	ticker := time.NewTicker(3 * time.Second)
	for {
		select {
		case now := <-ticker.C:
			block := cs.MakeBlock(uint64(now.Unix()))
			if block != nil {
				sig, err := cs.wallet.SignHash(cs.coinbase, block.Header.Hash[:])
				if err != nil {
					log.CLog().WithFields(logrus.Fields{
						"Msg": err,
					}).Warning("SignHash")
				}
				block.SignWithSignature(sig)
				cs.bc.PutBlockByCoinbase(block)
				cs.bc.Consensus.UpdateLIB()
				cs.bc.RemoveOrphanBlock()
				message, _ := net.NewRLPMessage(net.MsgNewBlock, block.BaseBlock)
				cs.streamPool.BroadcastMessage(&message)
			}
		}
	}
}

func (cs *Dpos) Verify(block *core.Block) (err error) {
	//block.Header.Coinbase
	state := block.ConsensusState.(*DposState)
	miners, err := state.GetMiners(state.MinersHash)
	if err != nil {
		return err
	}
	turn := (block.Header.Time % 9) / 3
	if miners[turn] != block.Header.Coinbase {
		return errors.New("This time is not your turn")
	}
	return nil
}

func (cs *Dpos) SaveState(block *core.Block) (err error) {
	state := block.ConsensusState.(*DposState)
	accs := block.AccountState
	electedTime := state.GetNewElectedTime(state.ElectedTime, block.Header.Time, 3, 3, 3)
	if electedTime == block.Header.Time {
		miners, err := state.GetNewRoundMiners(block.Header.Time, 3)
		if err != nil {
			return err
		}
		state.MinersHash, err = state.PutMiners(miners)
		state.ElectedTime = block.Header.Time

		iter, err := state.Voter.Iterator(nil)
		if err != nil {
			return err
		}
		exist, _ := iter.Next()
		for exist {
			account := accs.GetAccount(common.BytesToAddress(iter.Key()))
			account.CalcSetTotalPeggedStake()
			accs.PutAccount(account)
			exist, err = iter.Next()
		}
		//reset voter if this round is new
		state.Voter, err = trie.NewTrie(nil, cs.bc.Storage, false)
	}
	err = state.Put(block.Header.Height, state.ElectedTime, state.MinersHash)
	if err != nil {
		return err
	}
	return nil
}

/*
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
*/

//----------    Consensus  ----------------//

func (d *Dpos) UpdateLIB() {
	bc := d.bc
	block := bc.Tail
	//FIXME: consider timestamp, changed minerGroup
	miners := make(map[common.Address]bool)
	turn := 1
	for bc.Lib.Hash() != block.Hash() {
		miners[block.Header.Coinbase] = true
		//minerGroup, _, _ := block.MinerState.GetMinerGroup(bc, block)
		if turn == 3 {
			if len(miners) == 3 {
				bc.SetLib(block)
				log.CLog().WithFields(logrus.Fields{
					"Height": block.Header.Height,
					//"address": common.Hash2Hex(block.Hash()),
				}).Info("Updated Lib")
				return
			}
			miners = make(map[common.Address]bool)
			miners[block.Header.Coinbase] = true
			turn = 0
		}
		block = bc.GetBlockByHash(block.Header.ParentHash)
		turn++
	}
	return
}

func (c *Dpos) ConsensusType() string {
	return "DPOS"
}

func (cs *Dpos) LoadState(block *core.Block) (state core.ConsensusState, err error) {
	bc := cs.bc

	state, err = NewInitState(block.Header.ConsensusHash, block.Header.Height, bc.Storage)
	if err != nil {
		return nil, err
	}
	return state, nil
}

func (cs *Dpos) MakeGenesisBlock(block *core.Block, voters []*core.Account) (err error) {
	bc := cs.bc

	state, err := NewInitState(common.Hash{}, 0, bc.Storage)
	if err != nil {
		return err
	}

	//TODO: who voter?
	for _, v := range voters {
		state.Stake(v.Address, v.Address, v.Balance)
	}
	miners, err := state.GetNewRoundMiners(block.Header.Time, 3)
	if err != nil {
		return err
	}
	state.MinersHash, err = state.PutMiners(miners)
	state.ElectedTime = block.Header.Time
	state.Put(block.Header.Height, state.ElectedTime, state.MinersHash)

	block.ConsensusState = state
	block.Header.ConsensusHash = state.RootHash()
	bc.GenesisBlock = block
	bc.GenesisBlock.MakeHash()
	return nil
}

func (cs *Dpos) AddBlockChain(bc *core.BlockChain) {
	cs.bc = bc
}

// //TODO: rename clone
// func (cs *Dpos) CloneFromParentBlock(block *core.Block, parentBlock *core.Block) (err error) {
// 	block.ConsensusState, err = parentBlock.ConsensusState.Clone()
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

// func (cs *Dpos) SaveMiners(block *core.Block) (err error) {
// 	return nil
// }
