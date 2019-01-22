package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommon(t *testing.T) {
	a := Address{0x1, 0x1, 0x2}
	assert.Equal(t, "0x010102000000000000000000000000000000000000000000000000000000000000000000",
		AddressToHex(a),
	)
	h := Hash{0x1, 0x1, 0x2}
	assert.Equal(t, "0x0101020000000000000000000000000000000000000000000000000000000000",
		HashToHex(h),
	)
}
