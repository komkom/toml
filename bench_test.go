package toml

import (
	"bytes"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

type TmplBuffer struct {
	buf     bytes.Buffer
	tmpl    string
	counter int
	stop    bool
	n       int64
}

func (t *TmplBuffer) fillBuffer() {

	for i := 0; i < 100; i++ {
		tmp := strings.Replace(t.tmpl, `$1`, strconv.Itoa(t.counter), -1)
		t.buf.WriteString(tmp)
		t.counter++
	}

}

func (t *TmplBuffer) Read(p []byte) (int, error) {

	for {
		n, err := t.buf.Read(p)
		if err != nil && !errors.Is(err, io.EOF) {
			return n, err
		}
		if t.stop && (errors.Is(err, io.EOF) || n == 0) {
			return n, io.EOF
		}

		if n == 0 {
			t.fillBuffer()
			continue
		}

		t.n += int64(n)

		return n, nil
	}
}

var (
	tmpl = `
	key1_$1 = "tt"
	key2_$1 = "tt"
	[table_$1]
	1=1.389740932847e101917
	2="some longer string"
	[[array_$1]]
	a.b.c.d.e = "test value"
	[[array_$1]]
	a.b.c.d.e = "test value"
	[[array_$1]]
	a.b = "test value"`
)

func BenchmarkParserThroughput(b *testing.B) {

	buf := &TmplBuffer{tmpl: tmpl}
	p := make([]byte, 128)
	rd := New(buf)

	var n int64
	for i := 0; i < b.N; i++ {

		rn, err := rd.Read(p)
		require.NoError(b, err)

		n += int64(rn)
	}

	b.SetBytes(buf.n / int64(b.N))
}

func BenchmarkMemoryThroughput(b *testing.B) {

	buf := &TmplBuffer{tmpl: tmpl}
	p := make([]byte, 128)

	var n int64
	for i := 0; i < b.N; i++ {

		rn, err := buf.Read(p)
		require.NoError(b, err)

		n += int64(rn)
	}

	b.SetBytes(buf.n / int64(b.N))
}
