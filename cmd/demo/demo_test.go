package main

import (
	// "fmt"
	"fmt"
	"os"

	"testing"

	"github.com/nacamp/go-simplechain/account"
	"github.com/nacamp/go-simplechain/common"
	"github.com/stretchr/testify/assert"
)

func TestAccountImportAction(t *testing.T) {
	path := "./test_keystore.dat"
	os.Remove(path)
	defer os.Remove(path)
	accountImportAction(path, "0x8a21cd44e684dd2d8d9205b0bfb69339435c7bd016ebc21fddaddffd0d47ed63", "test")
	w := account.NewWallet(path)

	key, err := w.GetKey(common.HexToAddress("0xc6d40a9bf9fe9d90019511a2147dc0958657da97463ca59d2594d5536dcdfd30ed93707d"), "test")
	if err != nil {
		fmt.Println(err)
		return
	}
	assert.Equal(t, common.HexToAddress("0xc6d40a9bf9fe9d90019511a2147dc0958657da97463ca59d2594d5536dcdfd30ed93707d"), key.Address)
}
