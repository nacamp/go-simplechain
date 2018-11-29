package rpc

import (
	"context"

	"github.com/intel-go/fastjson"

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

type RpcService struct {
	server *RpcServer
}

func (rs *RpcService) Setup(server *RpcServer, config *core.Config) {
	rs.server = server
	rs.server.RegisterHandler(NewAccountsHandler(config))
}

/*
curl -X POST -H "Content-Type: application/json" -d '{"jsonrpc": "2.0",   "method": "accounts", "params":[]}' http://localhost:8080/jrpc
https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_accounts
curl -X POST --data '{"jsonrpc":"2.0","method":"eth_accounts","params":[],"id":1}'
{
  "id":1,
  "jsonrpc": "2.0",
  "result": ["0xc94770007dda54cF92009BFF0dE90c06F603a09f"]
}
*/
