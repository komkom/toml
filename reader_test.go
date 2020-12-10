package toml

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReader(t *testing.T) {

	tests := []struct {
		doc      string
		expected string
		err      string
	}{
		{
			doc:      `a."b".d=2`,
			expected: `{"a":{"b":{"d":2}}}`,
		},
		{
			doc: `a.'b.c.d'.d=2
			a.b.c.d=2`,
			expected: `{"a":{"b.c.d":{"d":2},"b":{"c":{"d":2}}}}`,
		},
		{
			doc:      `a."\uFFFF".c=1`,
			expected: `{"a":{"\uFFFF":{"c":1}}}`,
		},
		{
			doc:      `a."\UD7FF16".c=1`,
			expected: `{"a":{"\\UD7FF16":{"c":1}}}`,
		},
		{
			doc:      `key = """\uFFFF"""`,
			expected: `{"key":"\uFFFF"}`,
		},
		{
			doc:      `key = """\UD7FF16"""`,
			expected: `{"key":"\\UD7FF16"}`,
		},
	}

	for _, ts := range tests {

		buf := bytes.NewBufferString(ts.doc)

		rd := New(buf)

		data, err := ioutil.ReadAll(rd)

		if ts.err != `` {
			require.Error(t, err)
			assert.Contains(t, err.Error(), ts.err)
			continue
		}

		require.NoError(t, err)
		assert.True(t, json.Valid(data))
		assert.Equal(t, ts.expected, string(data))
	}
}
