package rpc

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/intel-go/fastjson"
	"github.com/nacamp/go-simplechain/account"
	"github.com/osamingo/jsonrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type AccountsHandlerMock struct {
	w *account.Wallet
	mock.Mock
}

func (h *AccountsHandlerMock) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {
	args := h.Called()
	p := []string{}
	if err := jsonrpc.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	if args.Get(1) != nil {
		return args.Get(0).([]string), args.Get(1).(*jsonrpc.Error) //args.String(0)
	} else {
		return args.Get(0).([]string), nil
	}

}

func TestCheckParamertAtAccountsHandler1(t *testing.T) {
	mr := jsonrpc.NewMethodRepository()
	testObj := &AccountsHandlerMock{w: nil}
	require.NoError(t, mr.RegisterMethod("accounts", testObj, []string{}, []string{}))
	testObj.On("ServeJSONRPC").Return([]string{"address1", "address2"}, &jsonrpc.Error{Code: 0, Message: "This is dummy error"})
	rec := httptest.NewRecorder()
	r, err := http.NewRequest("", "", bytes.NewReader([]byte(`{"jsonrpc": "2.0",   "method": "accounts", "params":[true]}`)))
	require.NoError(t, err)
	r.Header.Set("Content-Type", "application/json")

	mr.ServeHTTP(rec, r)
	res := jsonrpc.Response{}
	err = fastjson.NewDecoder(rec.Body).Decode(&res)
	require.NoError(t, err)
	assert.NotNil(t, res.Error)
	fmt.Printf("%+v\r\n", res.Error)
	testObj.AssertExpectations(t)
}

func TestCheckParamertAtAccountsHandler2(t *testing.T) {
	mr := jsonrpc.NewMethodRepository()
	testObj := &AccountsHandlerMock{w: nil}
	require.NoError(t, mr.RegisterMethod("accounts", testObj, []string{}, []string{}))
	testObj.On("ServeJSONRPC").Return([]string{"address1", "address2"}, nil)
	rec := httptest.NewRecorder()
	r, err := http.NewRequest("", "", bytes.NewReader([]byte(`{"jsonrpc": "2.0",   "method": "accounts", "params":[]}`)))
	require.NoError(t, err)
	r.Header.Set("Content-Type", "application/json")

	mr.ServeHTTP(rec, r)
	res := jsonrpc.Response{}
	err = fastjson.NewDecoder(rec.Body).Decode(&res)
	require.NoError(t, err)
	assert.Nil(t, res.Error)
	testObj.AssertExpectations(t)
}
