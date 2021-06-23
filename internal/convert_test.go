package toml

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvert(t *testing.T) {

	tests := []struct {
		value    string
		expected int
	}{
		{
			value:    `0x12AFE`,
			expected: 76542,
		},
		{
			value:    `0x00012AFE`,
			expected: 76542,
		},
		{
			value:    `0o67`,
			expected: 55,
		},
		{
			value:    `0o25677`,
			expected: 11199,
		},
		{
			value:    `0b11`,
			expected: 3,
		},
		{
			value:    `0b101`,
			expected: 5,
		},
		{
			value:    `0b0011`,
			expected: 3,
		},
	}

	var err error
	for _, ts := range tests {

		var total int

		for _, r := range ts.value[2:] {
			if ts.value[:2] == `0x` {
				total, err = addNumber(r, total, HexNumberType)
				require.NoError(t, err)
			}
			if ts.value[:2] == `0o` {
				total, err = addNumber(r, total, OctalNumberType)
				require.NoError(t, err)
			}
			if ts.value[:2] == `0b` {
				total, err = addNumber(r, total, BinNumberType)
				require.NoError(t, err)
			}
		}
		assert.Equal(t, ts.expected, total)
	}
}
