package rpc

import (
	"context"
	"strconv"

	"github.com/intel-go/fastjson"

	"github.com/najimmy/go-simplechain/common"
	"github.com/najimmy/go-simplechain/core"
	"github.com/osamingo/jsonrpc"
)

//FIXME: temporary service

// accounts >>>>>>>>>>>
type AccountsHandler struct {
	config *core.Config
}

func NewAccountsHandler(config *core.Config) *AccountsHandler {
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

type RpcService struct {
	server *RpcServer
}

func (rs *RpcService) Setup(server *RpcServer, config *core.Config, bc *core.BlockChain) {
	rs.server = server
	rs.server.RegisterHandler(NewAccountsHandler(config))
	rs.server.RegisterHandler(NewGetBalanceHandler(bc))
	rs.server.RegisterHandler(NewGetTransactionCountHandler(bc))
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
  "result": "0x1" // 1
}
*/
