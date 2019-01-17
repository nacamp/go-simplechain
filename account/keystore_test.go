package account

import (
	"fmt"
	"testing"
)

func TestNewKey(t *testing.T) {
	key := NewKey()
	fmt.Printf("%v", key)
}
