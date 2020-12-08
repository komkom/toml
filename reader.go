package toml

import (
	"fmt"
	"io"

	"github.com/pkg/errors"
)

type Reader struct {
	filter     *Filter
	filterDone bool
	reader     io.Reader
	readerDone bool
}

func New(reader io.Reader) *Reader {
	return &Reader{
		filter: NewFilter(),
		reader: reader,
	}
}

func (r *Reader) Read(p []byte) (int, error) {

	if !r.readerDone {
		for r.filter.buf.Len() < len(p) {
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
		r.filter.WriteRune(EOF)
		r.filter.Close()
	}

	if r.readerDone && r.filterDone && len(r.filter.state.buf.Bytes()) == 0 {
		if len(r.filter.state.scopes) != 0 {
			return 0, fmt.Errorf(`invalid EOF`)
		}
		return 0, io.EOF
	}

	ln := len(p)
	if len(r.filter.state.buf.Bytes()) < ln {
		ln = len(r.filter.state.buf.Bytes())
	}

	n, err := r.filter.state.buf.Read(p)
	r.filter.state.buf.Truncate(len(r.filter.state.buf.Bytes()))
	return n, err
}
