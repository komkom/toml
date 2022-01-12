package toml

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
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
			expected: `{"hex3":1192227}`,
		},
		{
			doc:      `hex3 = 0xdead_beef`,
			expected: `{"hex3":3735928559}`,
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
			expected: `{"sf6":"+nan"}`,
		},
		{
			doc:      `k = 0e0`,
			expected: `{"k":0e0}`,
		},
		{
			doc:      `sf6 = +inf`,
			expected: `{"sf6":"+inf"}`,
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
			expected: `{"key":"value\r\n"}`,
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
		{
			doc:      `multiline_end_esc = """When will it end? \"""...""\" should be here\""""`,
			expected: `{"multiline_end_esc":"When will it end? \"\"\"...\"\"\" should be here\""}`,
		},
		{
			doc:      `multiline_not_unicode = """\\u0041"""`,
			expected: `{"multiline_not_unicode":"\\u0041"}`,
		},
		{
			doc:      `winpath  = 'C:\Users\nodejs\templates'`,
			expected: `{"winpath":"C:\\Users\\nodejs\\templates"}`,
		},
		{
			doc:      `string_escape = "\U00000000"`,
			expected: `{"string_escape":"\\U00000000"}`,
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

			strings.HasSuffix(path, `spec-tests/errors/string-basic-out-of-range-unicode-escape-2.toml`) ||
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

func TestSpecTests_invalid(t *testing.T) {

	var counter int
	var failed int
	filepath.Walk(`spec-tests/tests/invalid`, func(path string, info os.FileInfo, e error) error {

		counter++
		if info.IsDir() || !strings.HasSuffix(info.Name(), `.toml`) {
			return nil
		}

		data, err := os.ReadFile(path)
		require.NoError(t, err)
		reader := bytes.NewReader(data)
		_, err = io.ReadAll(New(reader))
		if err == nil {
			failed++
			fmt.Printf("path_toml = %+v\n", path)
		}

		return nil
	})
	assert.Equal(t, 221, counter)
	assert.Equal(t, 7, failed)
}

func TestSpecTests_valid(t *testing.T) {

	var counter int
	var failed int
	filepath.Walk(`spec-tests/tests/valid`, func(path string, info os.FileInfo, e error) error {

		counter++
		if info.IsDir() || !strings.HasSuffix(info.Name(), `.toml`) {
			return nil
		}

		data, err := os.ReadFile(path)
		require.NoError(t, err)
		reader := bytes.NewReader(data)
		_, err = io.ReadAll(New(reader))
		if err != nil {
			failed++
			fmt.Printf("path_toml = %+v %v\n", path, err)
		}

		return nil
	})
	assert.Equal(t, 207, counter)
	assert.Equal(t, 2, failed)
}

func TestSpec_compareJSON(t *testing.T) {

	exclude := map[string]bool{
		`spec-tests/tests/valid/comment/everywhere.json`:    true,
		`spec-tests/tests/valid/datetime/datetime.json`:     true,
		`spec-tests/tests/valid/datetime/local.json`:        true,
		`spec-tests/tests/valid/datetime/milliseconds.json`: true,
		`spec-tests/tests/valid/float/inf-and-nan.json`:     true,
		`spec-tests/tests/valid/string/unicode-escape.json`: true,
	}

	filepath.Walk(`spec-tests/tests/valid`, func(path string, info os.FileInfo, e error) error {

		if info.IsDir() || !strings.HasSuffix(info.Name(), `.json`) {
			return nil
		}

		fmt.Printf("path %v\n", path)

		if _, ok := exclude[path]; ok {

			fmt.Printf("excluded\n")
			return nil
		}

		data, err := os.ReadFile(path)
		require.NoError(t, err)

		m := map[string]interface{}{}
		err = json.Unmarshal(data, &m)
		require.NoError(t, err)

		res, err := taggedToUntagged(m)
		require.NoError(t, err)

		path = path[:len(path)-5] + `.toml`

		data, err = os.ReadFile(path)
		require.NoError(t, err)

		reader := bytes.NewReader(data)
		parsedJSON, err := io.ReadAll(New(reader))
		require.NoError(t, err)

		parsedMap := map[string]interface{}{}

		err = json.Unmarshal(parsedJSON, &parsedMap)
		require.NoError(t, err)

		require.Equal(t, res, parsedMap)

		return nil
	})
}

func taggedToUntagged(obj interface{}) (interface{}, error) {

	switch o := obj.(type) {
	case map[string]interface{}:

		res, ok, err := unwrap(o)
		if err != nil {
			return nil, err
		}
		if ok {
			return res, nil
		}

		for key, value := range o {

			res, err := taggedToUntagged(value)
			if err != nil {
				return nil, err
			}

			o[key] = res
		}

		return obj, nil

	case []interface{}:

		arr := []interface{}{}
		for _, item := range o {

			res, err := taggedToUntagged(item)
			if err != nil {
				return nil, err
			}
			arr = append(arr, res)
		}

		return arr, nil
	default:
		return obj, nil
	}
}

func unwrap(m map[string]interface{}) (interface{}, bool, error) {

	var tagType string
	var tagValue string
	var counter, keyCounter int
	for key, value := range m {

		if counter > 2 {
			break
		}
		counter++

		if key == `type` {
			keyCounter++
			tagType = value.(string)
		}

		if key == `value` {
			keyCounter++
			tagValue = value.(string)
		}
	}

	if keyCounter < 2 {
		return nil, false, nil
	}

	switch tagType {
	case `integer`:
		v, err := strconv.Atoi(tagValue)
		if err != nil {
			return nil, false, fmt.Errorf(`unwarp integer failed %w`, err)
		}
		return float64(v), true, nil
	case `float`:

		switch tagValue {
		case `nan`, `-nan`, `+nan`, `inf`, `-inf`, `+inf`:
			return tagValue, true, nil
		}

		f, err := strconv.ParseFloat(tagValue, 64)
		if err != nil {
			return nil, false, fmt.Errorf("unwrap float failed %w", err)
		}
		return f, true, nil

	case `bool`:

		switch tagValue {
		case `true`:
			return true, true, nil
		case `false`:
			return false, true, nil
		}

		return nil, false, fmt.Errorf("unwrap bool failed %v", tagValue)
	case `string`, `datetime`, `date-local`, `time-local`, `datetime-local`:
		return tagValue, true, nil

	}

	panic(`no such type ` + tagType)
}
