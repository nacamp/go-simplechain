package rpc

import (
	"fmt"
	"net/http"

	"github.com/nacamp/go-simplechain/log"
	"github.com/sirupsen/logrus"

	"github.com/osamingo/jsonrpc"
)

type JsonHandler interface {
	jsonrpc.Handler
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

func (js *RpcServer) RegisterHandler(name string, handler JsonHandler, params interface{}, result interface{}) {
	if err := js.mr.RegisterMethod(name, handler, params, result); err != nil {
		log.CLog().WithFields(logrus.Fields{}).Warning(fmt.Sprintf("%+v", err))
	}

}

func (js *RpcServer) Start() {

	http.Handle("/jrpc", js.mr)
	// http.HandleFunc("/jrpc/debug", mr.ServeDebug)
	go func() {
		if err := http.ListenAndServe(js.address, http.DefaultServeMux); err != nil {
			log.CLog().WithFields(logrus.Fields{}).Warning(fmt.Sprintf("%+v", err))
		}
	}()
}