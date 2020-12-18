package toml

import (
	"bytes"
	"encoding/json"
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
			expected: `{"a":"\r"}`,
		},
		{
			doc:      `another = "# test"`,
			expected: `{"another":"# test"}`,
		},
		{
			doc:      `'quoted "value"' = "value"`,
			expected: `{"quoted \"value\"":"value"}`,
		},
		{
			doc:      `hex3 = 0x123_123`,
			expected: `{"hex3":"0x123123"}`,
		},
		{
			doc:      `hex3 = 0xdead_beef`,
			expected: `{"hex3":"0xDEADBEEF"}`,
		},
		{
			doc:      `flt9 = -0e0`,
			expected: `{"flt9":-0e0}`,
		},
		{
			doc:      `sf6 = -nan`,
			expected: `{"sf6":"-nan"}`,
		},
		{
			doc:      `sf6 = +nan`,
			expected: `{"sf6":"nan"}`,
		},
		{
			doc:      `k = 0e0`,
			expected: `{"k":0e0}`,
		},
		{
			doc:      `sf6 = +inf`,
			expected: `{"sf6":"inf"}`,
		},
		{
			doc:      `sf6 = -inf`,
			expected: `{"sf6":"-inf"}`,
		},
		{
			doc:      `sf6 = inf`,
			expected: `{"sf6":"inf"}`,
		},
		{
			doc: `key = """a b c \
									ooo"""`,
			expected: `{"key":"a b c ooo"}`,
		},
		{
			doc: `key = """value  \
					                        """`,
			expected: `{"key":"value  "}`,
		},
		{
			doc: `[x.y.z.w] # for this to work
					[x]`,
			expected: `{"x":{"y":{"z":{"w":{}}}}}`,
		},
		{
			doc:      `"key\r\n"=1`,
			expected: `{"key\r\n":1}`,
		},
		{
			doc:      `key="""value\r\n"""`,
			expected: `{"key":"value/r/n"}`,
		},
		{
			doc: `[[arr.x]]
				[arr.x.table]
				[[arr.x]]
				[arr.x.table]
				[x]
				[[arr.x]]
				`,
			expected: `{"arr":{"x":[{"table":{}},{"table":{}}]},"x":{},"arr":{"x":[{}]}}`,
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

		// json not valid a = "\U00000000"
		if strings.HasSuffix(path, `spec-string-escape-9.toml`) {
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
			strings.HasSuffix(path, `string-basic-multiline-out-of-range-unicode-escape-2.toml`) {
			return nil
		}

		files = append(files, path)
		return nil
	})
	require.NoError(t, err)

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
