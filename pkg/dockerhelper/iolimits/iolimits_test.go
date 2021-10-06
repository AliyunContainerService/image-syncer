package iolimits

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadAtMost(t *testing.T) {
	for _, c := range []struct {
		input, limit  int
		shouldSucceed bool
	}{
		{0, 0, true},
		{0, 1, true},
		{1, 0, false},
		{1, 1, true},
		{bytes.MinRead*5 - 1, bytes.MinRead * 5, true},
		{bytes.MinRead * 5, bytes.MinRead * 5, true},
		{bytes.MinRead*5 + 1, bytes.MinRead * 5, false},
	} {
		input := make([]byte, c.input)
		_, err := rand.Read(input)
		require.NoError(t, err)
		result, err := ReadAtMost(bytes.NewReader(input), c.limit)
		if c.shouldSucceed {
			assert.NoError(t, err)
			assert.Equal(t, result, input)
		} else {
			assert.Error(t, err)
		}
	}
}
