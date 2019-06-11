package rpc

import (
	"bytes"
	"context"
	"math/big"
	"strconv"
	"time"

	"github.com/nacamp/go-simplechain/account"
	"github.com/nacamp/go-simplechain/cmd"
	"github.com/nacamp/go-simplechain/rlp"

	"github.com/intel-go/fastjson"
	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/core"
	"github.com/osamingo/jsonrpc"
)

type JsonTx struct {
	From    string       `json:"from"`
	To      string       `json:"to"`
	Amount  string       `json:"amount"`
	Nonce   string       `json:"nonce"`
	Payload *JsonPayload `json:"payload"`
}

type JsonPayload struct {
	Code string `json:"code"`
	Data string `json:"data"`
}

type JsonAccount struct {
	Address  string `json:"address"`
	Password string `json:"password"`
	Timeout  int    `json:"timeout"`
}

type JsonStatus struct {
	Message string `json:"message"`
	Status  bool   `json:"status"`
}

type AccountsHandler struct {
	w *account.Wallet
}

func (h *AccountsHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {
	addresses := []string{}
	for _, address := range h.w.Addresses() {
		addresses = append(addresses, common.AddressToHex(address))
	}
	return addresses, nil
}

type GetBalanceHandler struct {
	bc *core.BlockChain
}

func (h *GetBalanceHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {
	p := []string{}
	if err := jsonrpc.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	account := h.bc.Tail().AccountState.GetAccount(common.HexToAddress(p[0]))
	return account.AvailableBalance().String(), nil
}

type GetTransactionCountHandler struct {
	bc *core.BlockChain
}

func (h *GetTransactionCountHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {
	p := []string{}
	if err := jsonrpc.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	account := h.bc.Tail().AccountState.GetAccount(common.HexToAddress(p[0]))
	return strconv.FormatUint(account.Nonce, 10), nil
}

type SendTransactionHandler struct {
	bc        *core.BlockChain
	w         *account.Wallet
	consensus string
}

func (h *SendTransactionHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {
	var p JsonTx
	if err := jsonrpc.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	var from common.Address
	for _, address := range h.w.Addresses() {
		if common.HexToAddress(p.From) == address {
			from = common.HexToAddress(p.From)
			break
		}
	}
	if from == (common.Address{}) {
		return "", &jsonrpc.Error{Code: 0, Message: "The from is not our address"}
	}

	if !h.w.IsUnlockedAddress(from) {
		return "", &jsonrpc.Error{Code: 0, Message: "The from is locked"}
	}

	amount, _ := new(big.Int).SetString(p.Amount, 10)
	usedAmount := new(big.Int)
	if p.Payload == nil {
		usedAmount = usedAmount.Add(usedAmount, amount)
	} else if h.consensus == "dpos" && p.Payload.Code == "1" {
		_amount, _ := new(big.Int).SetString(p.Payload.Data, 10)
		usedAmount = usedAmount.Add(usedAmount, _amount)
	}
	nonce, _ := strconv.ParseUint(p.Nonce, 10, 64)
	account := h.bc.Tail().AccountState.GetAccount(from)

	if nonce <= account.Nonce {
		return "", &jsonrpc.Error{Code: 0, Message: "This transaction have wrong nonce"}
	}

	txs := h.bc.TxPool.FromTransactions(from)
	for _, tx := range txs {
		if tx.Payload.Code == uint64(0) {
			usedAmount = usedAmount.Add(usedAmount, tx.Amount)
		} else if h.consensus == "dpos" && tx.Payload.Code == uint64(1) {
			_amount := new(big.Int)
			err := rlp.Decode(bytes.NewReader(tx.Payload.Data), _amount)
			if err != nil {
				return "", &jsonrpc.Error{Code: 0, Message: err.Error()}
			}
			usedAmount = usedAmount.Add(usedAmount, _amount)
		}
	}

	if usedAmount.Cmp(account.AvailableBalance()) > 0 {
		return "", &jsonrpc.Error{Code: 0, Message: "There is insufficient amount."}
	}

	var tx *core.Transaction
	if p.Payload == nil {
		tx = core.NewTransaction(from, common.HexToAddress(p.To), amount, nonce)
	} else {
		txPayload := new(core.Payload)

		code, _ := strconv.ParseUint(p.Payload.Code, 10, 64)
		txPayload.Code = code
		if p.Payload.Data == "" {
			data, _ := strconv.ParseUint(p.Payload.Data, 10, 64)
			bytePayload, _ := rlp.EncodeToBytes(data)
			txPayload.Data = bytePayload
		}

		tx = core.NewTransactionPayload(from, common.HexToAddress(p.To), amount, nonce, txPayload)
	}
	tx.MakeHash()
	sig, err := h.w.SignHash(from, tx.Hash[:])
	if err != nil {
		return "", &jsonrpc.Error{Code: 0, Message: err.Error()}
	}
	tx.SignWithSignature(sig)
	h.bc.TxPool.Put(tx)
	h.bc.NewTXMessage <- tx
	return common.HashToHex(tx.Hash), nil
}

type GetTransactionByHashHandler struct {
	bc *core.BlockChain
}

func (h *GetTransactionByHashHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {
	p := []string{}
	if err := jsonrpc.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	tx, err := h.bc.Tail().TransactionState.GetTransaction(common.HexToHash(p[0]))
	if err != nil {
		return "", &jsonrpc.Error{Code: 0, Message: err.Error()}
	}

	rtx := &JsonTx{}
	rtx.From = common.AddressToHex(tx.From)
	rtx.To = common.AddressToHex(tx.To)
	rtx.Nonce = strconv.FormatUint(tx.Nonce, 10)
	rtx.Amount = tx.Amount.String()
	rtx.Payload = &JsonPayload{}
	rtx.Payload.Code = strconv.FormatUint(tx.Payload.Code, 10)
	if len(tx.Payload.Data) != 0 {
		data := new(uint64)
		err = rlp.Decode(bytes.NewReader(tx.Payload.Data), data)
		if err != nil {
			return "", &jsonrpc.Error{Code: 0, Message: err.Error()}
		}
		rtx.Payload.Data = strconv.FormatUint(*data, 10)
	}
	return rtx, nil
}

type NewAccountHandler struct {
	w *account.Wallet
}

func (h *NewAccountHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {
	p := []string{}
	if err := jsonrpc.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	key := account.NewKey()
	h.w.StoreKey(key, p[0])

	return common.AddressToHex(key.Address), nil
}

type UnlockHandler struct {
	w *account.Wallet
}

func (h *UnlockHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {
	var p JsonAccount
	if err := jsonrpc.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	err := h.w.TimedUnlock(common.HexToAddress(p.Address), p.Password, time.Duration(p.Timeout)*time.Second)
	if err != nil {
		return "error", &jsonrpc.Error{Code: 0, Message: err.Error()}
	}
	return "success", nil
}

type RpcService struct {
	server *RpcServer
}

func (rs *RpcService) Setup(server *RpcServer, config *cmd.Config, bc *core.BlockChain, w *account.Wallet) {
	rs.server = server
	rs.server.RegisterHandler("accounts", &AccountsHandler{w: w}, []string{}, []string{})
	rs.server.RegisterHandler("getBalance", &GetBalanceHandler{bc: bc}, []string{}, *new(string))
	rs.server.RegisterHandler("getTransactionCount", &GetTransactionCountHandler{bc: bc}, []string{}, "")                               //same *new(string)
	rs.server.RegisterHandler("sendTransaction", &SendTransactionHandler{bc: bc, w: w, consensus: config.Consensus.Name}, JsonTx{}, "") //same *new(string)
	rs.server.RegisterHandler("getTransactionByHash", &GetTransactionByHashHandler{bc: bc}, []string{}, JsonTx{})
	rs.server.RegisterHandler("newAccount", &NewAccountHandler{w: w}, []string{}, "") //same *new(string)
	rs.server.RegisterHandler("unlock", &UnlockHandler{w: w}, JsonAccount{}, "")      //same *new(string)
}

/*
https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_accounts

curl -X POST -H "Content-Type: application/json" -d '{"jsonrpc": "2.0",   "method": "accounts", "params":[]}' http://localhost:8080/jrpc
curl -X POST -H "Content-Type: application/json" -d '{"jsonrpc": "2.0",   "method": "getBalance", "params":["0x036407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0"]}' http://localhost:8080/jrpc
curl -X POST -H "Content-Type: application/json" -d '{"jsonrpc": "2.0",   "method": "getTransactionCount", "params":["0x036407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0"]}' http://localhost:8080/jrpc
curl -X POST -H "Content-Type: application/json" -d '{"jsonrpc": "2.0",   "method": "sendTransaction", "params": {"from": "0x3068c6c17a079f67b3f29a9844cbf6137a2bd7a3a58f0d0eac11b8afcd4564b8e4173af7","to": "0x03e864b08b08f632c61c6727cde0e23d125f7784b5a5a188446fc5c91ffa51faa1","amount": "1", "nonce": "1"}}' http://localhost:8080/jrpc
curl -X POST -H "Content-Type: application/json" -d '{"jsonrpc": "2.0",   "method": "getTransactionByHash", "params":["0xb017e021b8b2deba156941f32ee2e6c53c767a13749fba2533c7a30616ff48c3"]}' http://localhost:8080/jrpc
curl -X POST -H "Content-Type: application/json" -d '{"jsonrpc": "2.0",   "method": "newAccount", "params":["password"]}' http://localhost:8080/jrpc
curl -X POST -H "Content-Type: application/json" -d '{"jsonrpc": "2.0",   "method": "unlock", "params": {"address": "0x3068c6c17a079f67b3f29a9844cbf6137a2bd7a3a58f0d0eac11b8afcd4564b8e4173af7","password": "password","timeout": 300}}' http://localhost:8080/jrpc
*/
