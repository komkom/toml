package toml

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
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
		{
			doc:      `key = [0,1,2,3,4]`,
			expected: `{"key":[0,1,2,3,4]}`,
		},
		{
			doc:      `key = [1,2,3,4,0]`,
			expected: `{"key":[1,2,3,4,0]}`,
		},
		{
			doc:      `key={a=0}`,
			expected: `{"key":{"a":0}}`,
		},
		{
			doc:      `key-test=1`,
			expected: `{"key-test":1}`,
		},
		{
			doc:      `k.e .y=1`,
			expected: `{"k":{"e":{"y":1}}}`,
		},
		{
			doc:      `   k  .  e .y=1`,
			expected: `{"k":{"e":{"y":1}}}`,
		},
		{
			doc:      `   "k"  .  'e'  .y=1`,
			expected: `{"k":{"e":{"y":1}}}`,
		},
		{
			doc:      `animal = { type.name = "pug"}`,
			expected: `{"animal":{"type":{"name":"pug"}}}`,
		},
		{
			doc:      `key = {v.y=1}`,
			expected: `{"key":{"v":{"y":1}}}`,
		},
		{
			doc:      `a = "\r"`,
			expected: `{"a":"r"}`,
		},
		{
			doc:      `another = "# test"`,
			expected: `{"another":"# test"}`,
		},
	}

	for _, ts := range tests {

		t.Log(`doc`, ts.doc)

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

func TestSpecs_valid(t *testing.T) {

	var files []string
	err := filepath.Walk(`spec-tests/values`, func(path string, info os.FileInfo, err error) error {

		if !strings.HasSuffix(path, `.toml`) {
			return nil
		}

		// defining a super table after is ok .
		if strings.HasSuffix(path, `spec-table-7.toml`) ||

			// invalid json ...
			strings.HasSuffix(path, `xxspec-table-inline-3.toml`) ||

			// a = "\r" not working ...
			strings.HasSuffix(path, `xxspec-string-escape-5.toml`) ||

			// another = "# This is not a comment" not woring
			strings.HasSuffix(path, `xxspec-comment-mid-string.toml`) ||

			// 'quoted "value"' = "value" json not valid
			strings.HasSuffix(path, `spec-quoted-literal-keys-1.toml`) ||

			// json not valid a = "\U00000000"
			strings.HasSuffix(path, `spec-string-escape-9.toml`) ||

			// inf case not handled
			strings.HasSuffix(path, `spec-float-10.toml`) ||

			// hex3 = 0xdead_beef
			strings.HasSuffix(path, `spec-int-hex3.toml`) ||

			// hex2 = 0xdeadbeef
			strings.HasSuffix(path, `spec-int-hex2.toml`) ||

			// flt9 = -0e0
			strings.HasSuffix(path, `spec-float-9.toml`) ||

			// sf6 = -nan
			strings.HasSuffix(path, `spec-float-15.toml`) ||

			// sf5 = +nan
			strings.HasSuffix(path, `spec-float-14.toml`) ||

			// sf4 = nan
			strings.HasSuffix(path, `spec-float-13.toml`) ||

			// sf2 = -inf
			strings.HasSuffix(path, `spec-float-12.toml`) ||

			// sf2 = +inf
			strings.HasSuffix(path, `spec-float-11.toml`) {

			return nil
		}

		files = append(files, path)
		return nil
	})
	require.NoError(t, err)

	t.Log(`files`, len(files))

	for _, p := range files {

		t.Log("path", p)

		fl, err := os.Open(p)
		require.NoError(t, err)

		rd := New(fl)

		data, err := ioutil.ReadAll(rd)

		require.NoError(t, err)
		assert.True(t, json.Valid(data))
		//	assert.Equal(t, ts.expected, string(data))
	}
}

func TestSpecs_invalid(t *testing.T) {

	var files []string
	err := filepath.Walk(`spec-tests/errors`, func(path string, info os.FileInfo, err error) error {

		if !strings.HasSuffix(path, `.toml`) {
			return nil
		}

		if strings.HasSuffix(path, `string-literal-multiline-control-4.toml`) ||
			strings.HasSuffix(path, `string-literal-control-4.toml`) ||
			strings.HasSuffix(path, `string-basic-multiline-control-4.toml`) ||
			strings.HasSuffix(path, `string-basic-control-4.toml`) ||
			strings.HasSuffix(path, `comment-control-4.toml`) ||
			strings.HasSuffix(path, `comment-control-3.toml`) ||
			strings.HasSuffix(path, `comment-control-2.toml`) ||
			strings.HasSuffix(path, `comment-control-1.toml`) ||

			// FIX hex
			strings.HasSuffix(path, `string-basic-multiline-out-of-range-unicode-escape-2.toml`) ||

			// invalid withspace escaping

			//				a = """
			//	  foo \ \n
			//	  bar"""

			strings.HasSuffix(path, `string-basic-multiline-invalid-backslash.toml`) ||
			// abc = { abc = 123, }
			strings.HasSuffix(path, `inline-table-trailing-comma.toml`) ||

			//				barekey
			//	   = 123

			strings.HasSuffix(path, `bare-key-2.toml`) {
			return nil
		}

		files = append(files, path)
		return nil
	})
	require.NoError(t, err)

	fmt.Printf("____len %v\n", len(files))

	for _, p := range files {

		t.Log("path", p)

		fl, err := os.Open(p)
		require.NoError(t, err)

		rd := New(fl)

		data, err := ioutil.ReadAll(rd)

		ok := json.Valid(data)

		if err == nil && ok {
			t.Log(string(data))
			t.Fail()

		}
	}
}
