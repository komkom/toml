package toml

import (
	"fmt"
	"io"

	toml "github.com/komkom/toml/internal"
	"github.com/pkg/errors"
)

type Reader struct {
	filter     *toml.Filter
	filterDone bool
	reader     io.Reader
	readerDone bool
}

// New wraps an io.Reader around an io.Reader.
// Reading data from this Reader reads data from
// its underlying wrapped io.Reader, parses and
// encodes it as a JSON stream.
func New(reader io.Reader) *Reader {
	return &Reader{
		filter: toml.NewFilter(),
		reader: reader,
	}
}

func (r *Reader) Read(p []byte) (int, error) {

	if !r.readerDone {
		for r.filter.State.Buf.Len() < len(p) {
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

	if r.readerDone && r.filterDone && len(r.filter.State.Buf.Bytes()) == 0 {
		if len(r.filter.State.Scopes) != 0 {
			return 0, fmt.Errorf(`invalid EOF`)
		}
		return 0, io.EOF
	}

	ln := len(p)
	if len(r.filter.State.Buf.Bytes()) < ln {
		ln = len(r.filter.State.Buf.Bytes())
	}

	n, err := r.filter.State.Buf.Read(p)
	r.filter.State.Buf.Truncate(len(r.filter.State.Buf.Bytes()))
	return n, err
}
