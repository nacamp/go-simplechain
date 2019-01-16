package rpc

import (
	"net/http"

	"github.com/nacamp/go-simplechain/log"

	"github.com/osamingo/jsonrpc"
)

type JsonHandler interface {
	jsonrpc.Handler
	Name() string
	Params() interface{}
	Result() interface{}
}

type RpcServer struct {
	mr      *jsonrpc.MethodRepository
	address string
}

func NewRpcServer(address string) *RpcServer {
	return &RpcServer{
		mr:      jsonrpc.NewMethodRepository(),
		address: address,
	}
}

func (js *RpcServer) RegisterHandler(handler JsonHandler) {
	if err := js.mr.RegisterMethod(handler.Name(), handler, handler.Params, handler.Result); err != nil {
		log.CLog().Warning(err)
	}

}

func (js *RpcServer) Start() {

	http.Handle("/jrpc", js.mr)
	// http.HandleFunc("/jrpc/debug", mr.ServeDebug)
	go func() {
		if err := http.ListenAndServe(js.address, http.DefaultServeMux); err != nil {
			log.CLog().Warning(err)
		}
	}()
}
