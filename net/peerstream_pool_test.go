package net

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPeerStreamPool(t *testing.T) {
	pool := NewPeerStreamPool()

	ps, err := pool.GetStream("id")
	assert.Error(t, err)
	_ = ps
}
