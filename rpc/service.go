package rpc

import (
	"context"
	"math/big"
	"strconv"

	"github.com/najimmy/go-simplechain/cmd"
	"github.com/najimmy/go-simplechain/rlp"

	"github.com/intel-go/fastjson"

	"github.com/najimmy/go-simplechain/common"
	"github.com/najimmy/go-simplechain/core"
	"github.com/osamingo/jsonrpc"
)

//FIXME: temporary service

// accounts >>>>>>>>>>>
type AccountsHandler struct {
	config *cmd.Config
}

func NewAccountsHandler(config *cmd.Config) *AccountsHandler {
	return &AccountsHandler{config: config}
}
func (h *AccountsHandler) Name() string {
	return "accounts"
}
func (h *AccountsHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {
	return []string{h.config.MinerAddress}, nil
}
func (h *AccountsHandler) Params() interface{} {
	return []string{}
}
func (h *AccountsHandler) Result() interface{} {
	return []string{}
}

// accounts <<<<<<<<<<

// getBalance >>>>>>>>>>>
type GetBalanceHandler struct {
	bc *core.BlockChain
}

func NewGetBalanceHandler(bc *core.BlockChain) *GetBalanceHandler {

	return &GetBalanceHandler{bc: bc}
}

func (h *GetBalanceHandler) Name() string {
	return "getBalance"
}

func (h *GetBalanceHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {
	p := []string{}
	if err := jsonrpc.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	account := h.bc.Tail.AccountState.GetAccount(common.HexToAddress(p[0]))
	return account.Balance.String(), nil
}

func (h *GetBalanceHandler) Params() interface{} {
	return []string{}
}
func (h *GetBalanceHandler) Result() interface{} {
	return ""
}

// getBalance <<<<<<<<<<

// getTransactionCount >>>>>>>>>>>
type GetTransactionCountHandler struct {
	bc *core.BlockChain
}

func NewGetTransactionCountHandler(bc *core.BlockChain) *GetTransactionCountHandler {

	return &GetTransactionCountHandler{bc: bc}
}

func (h *GetTransactionCountHandler) Name() string {
	return "getTransactionCount"
}

func (h *GetTransactionCountHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {
	p := []string{}
	if err := jsonrpc.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	account := h.bc.Tail.AccountState.GetAccount(common.HexToAddress(p[0]))
	return strconv.FormatUint(account.Nonce, 10), nil
}

func (h *GetTransactionCountHandler) Params() interface{} {
	return []string{}
}
func (h *GetTransactionCountHandler) Result() interface{} {
	return ""
}

// getTransactionCount <<<<<<<<<<

// sendTransaction >>>>>>>>>>>
type TempTx struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Amount  string `json:"amount"`
	Nonce   string `json:"nonce"`
	Payload string `json:"payload"`
}

type SendTransactionHandler struct {
	bc *core.BlockChain
}

func NewSendTransactionHandler(bc *core.BlockChain) *SendTransactionHandler {

	return &SendTransactionHandler{bc: bc}
}

func (h *SendTransactionHandler) Name() string {
	return "sendTransaction"
}

func (h *SendTransactionHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {
	var p TempTx
	if err := jsonrpc.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	amount, _ := new(big.Int).SetString(p.Amount, 10)
	nonce, _ := strconv.ParseUint(p.Nonce, 10, 64)
	var tx *core.Transaction
	if p.Payload == "" {
		tx = core.MakeTransaction(p.From, p.To, amount, nonce)
	} else {
		payload, _ := strconv.ParseBool(p.Payload)
		bytePpayload, _ := rlp.EncodeToBytes(payload)
		tx = core.MakeTransactionPayload(p.From, p.To, amount, nonce, bytePpayload)
	}

	h.bc.TxPool.Put(tx)
	h.bc.NewTXMessage <- tx
	return common.Hash2Hex(tx.Hash), nil
}

func (h *SendTransactionHandler) Params() interface{} {
	return TempTx{}
}
func (h *SendTransactionHandler) Result() interface{} {
	return ""
}

// sendTransaction <<<<<<<<<<

// getTransactionByHash >>>>>>>>>>>
type GetTransactionByHashHandler struct {
	bc *core.BlockChain
}

func NewGetTransactionByHashHandler(bc *core.BlockChain) *GetTransactionByHashHandler {

	return &GetTransactionByHashHandler{bc: bc}
}

func (h *GetTransactionByHashHandler) Name() string {
	return "getTransactionByHash"
}

func (h *GetTransactionByHashHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {
	p := []string{}
	if err := jsonrpc.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	tx := h.bc.Tail.TransactionState.GetTransaction(common.HexToHash(p[0]))

	rtx := &TempTx{}
	rtx.From = common.Address2Hex(tx.From)
	rtx.To = common.Address2Hex(tx.To)
	rtx.Nonce = strconv.FormatUint(tx.Nonce, 10)
	rtx.Amount = tx.Amount.String()
	return rtx, nil
}

func (h *GetTransactionByHashHandler) Params() interface{} {
	return []string{}
}
func (h *GetTransactionByHashHandler) Result() interface{} {
	return TempTx{}
}

// getTransactionByHash <<<<<<<<<<

type RpcService struct {
	server *RpcServer
}

func (rs *RpcService) Setup(server *RpcServer, config *cmd.Config, bc *core.BlockChain) {
	rs.server = server
	rs.server.RegisterHandler(NewAccountsHandler(config))
	rs.server.RegisterHandler(NewGetBalanceHandler(bc))
	rs.server.RegisterHandler(NewGetTransactionCountHandler(bc))
	rs.server.RegisterHandler(NewSendTransactionHandler(bc))
	rs.server.RegisterHandler(NewGetTransactionByHashHandler(bc))
}

/*
https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_accounts

curl -X POST -H "Content-Type: application/json" -d '{"jsonrpc": "2.0",   "method": "accounts", "params":[]}' http://localhost:8080/jrpc
curl -X POST --data '{"jsonrpc":"2.0","method":"eth_accounts","params":[],"id":1}'
{
  "id":1,
  "jsonrpc": "2.0",
  "result": ["0xc94770007dda54cF92009BFF0dE90c06F603a09f"]
}

curl -X POST -H "Content-Type: application/json" -d '{"jsonrpc": "2.0",   "method": "getBalance", "params":["0x036407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0"]}' http://localhost:8080/jrpc
curl -X POST --data '{"jsonrpc":"2.0","method":"eth_getBalance","params":["0xc94770007dda54cF92009BFF0dE90c06F603a09f", "latest"],"id":1}'
{
  "id":1,
  "jsonrpc": "2.0",
  "result": "0x0234c8a3397aab58" // 158972490234375000
}

curl -X POST -H "Content-Type: application/json" -d '{"jsonrpc": "2.0",   "method": "getTransactionCount", "params":["0x036407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0"]}' http://localhost:8080/jrpc
curl -X POST --data '{"jsonrpc":"2.0","method":"eth_getTransactionCount","params":["0xc94770007dda54cF92009BFF0dE90c06F603a09f","latest"],"id":1}'
params: [
   '0xc94770007dda54cF92009BFF0dE90c06F603a09f',
   'latest' // state at the latest block
]
{
  "id":1,
  "jsonrpc": "2.0",
  "result": "0x1"
}

curl -X POST -H "Content-Type: application/json" -d '{"jsonrpc": "2.0",   "method": "sendTransaction", "params": {"from": "0x036407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0","to": "0x03e864b08b08f632c61c6727cde0e23d125f7784b5a5a188446fc5c91ffa51faa1","amount": "1", "nonce": "1"}
}' http://localhost:8080/jrpc
curl -X POST --data '{"jsonrpc":"2.0","method":"eth_sendTransaction","params":[{see above}],"id":1}'
params: [{
  "from": "0xb60e8dd61c5d32be8058bb8eb970870f07233155",
  "to": "0xd46e8dd67c5d32be8058bb8eb970870f07244567",
  "gas": "0x76c0", // 30400
  "gasPrice": "0x9184e72a000", // 10000000000000
  "value": "0x9184e72a", // 2441406250
  "data": "0xd46e8dd67c5d32be8d46e8dd67c5d32be8058bb8eb970870f072445675058bb8eb970870f072445675"
}]
{
  "id":1,
  "jsonrpc": "2.0",
  "result": "0xe670ec64341771606e55d6b4ca35a1a6b75ee3d5145a99d05921026d1527331"
}

curl -X POST -H "Content-Type: application/json" -d '{"jsonrpc": "2.0",   "method": "getTransactionByHash", "params":["0xb017e021b8b2deba156941f32ee2e6c53c767a13749fba2533c7a30616ff48c3"]}' http://localhost:8080/jrpc
curl -X POST --data '{"jsonrpc":"2.0","method":"eth_getTransactionByHash","params":["0x88df016429689c079f3b2f6ad39fa052532c56795b733da78a91ebe6a713944b"],"id":1}'
{
  "jsonrpc":"2.0",
  "id":1,
  "result":{
    "blockHash":"0x1d59ff54b1eb26b013ce3cb5fc9dab3705b415a67127a003c3e61eb445bb8df2",
    "blockNumber":"0x5daf3b", // 6139707
    "from":"0xa7d9ddbe1f17865597fbd27ec712455208b6b76d",
    "gas":"0xc350", // 50000
    "gasPrice":"0x4a817c800", // 20000000000
    "hash":"0x88df016429689c079f3b2f6ad39fa052532c56795b733da78a91ebe6a713944b",
    "input":"0x68656c6c6f21",
    "nonce":"0x15", // 21
    "to":"0xf02c1c8e6114b1dbe8937a39260b5b0a374432bb",
    "transactionIndex":"0x41", // 65
    "value":"0xf3dbb76162000", // 4290000000000000
    "v":"0x25", // 37
    "r":"0x1b5e176d927f8e9ab405058b2d2457392da3e20f328b16ddabcebc33eaac5fea",
    "s":"0x4ba69724e8f69de52f0125ad8b3c5c2cef33019bac3249e2c0a2192766d1721c"
  }
}
*/
