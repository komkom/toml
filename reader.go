package toml

import (
	"bytes"
	"fmt"
	"io"

	toml "github.com/komkom/toml/internal"
	"github.com/pkg/errors"
)

type Reader struct {
	filter     *toml.Filter
	filterDone bool
	reader     io.Reader
	out        *bytes.Buffer
	readerDone bool
}

// New wraps an io.Reader around an io.Reader.
// Reading data from this Reader reads data form
// its underlying wrapped io.Reader, parses and
// encodes it as a JSON stream.
func New(reader io.Reader) *Reader {
	var out bytes.Buffer
	return &Reader{
		out:    &out,
		filter: toml.NewFilter(&out),
		reader: reader,
	}
}

func (r *Reader) Read(p []byte) (int, error) {

	if !r.readerDone {
		for r.out.Len() < len(p) {
			n, err := r.reader.Read(p)
			_, err = r.filter.Write(p[:n])
			if err != nil {
				return 0, err
			}

			if errors.Is(err, io.EOF) || n == 0 {
				r.readerDone = true
				break
			}

			if err != nil {
				return 0, err
			}
		}
	}

	if r.readerDone && !r.filterDone {
		r.filterDone = true
		r.filter.WriteRune('\n')
		r.filter.WriteRune(toml.EOF)
		r.filter.Close()
	}

	if r.readerDone && r.filterDone && len(r.out.Bytes()) == 0 {
		if len(r.filter.State.Scopes) != 0 {
			return 0, fmt.Errorf(`invalid EOF`)
		}
		return 0, io.EOF
	}

	ln := len(p)
	if len(r.out.Bytes()) < ln {
		ln = len(r.out.Bytes())
	}

	n, err := r.out.Read(p)
	r.out.Truncate(len(r.out.Bytes()))
	return n, err
}
