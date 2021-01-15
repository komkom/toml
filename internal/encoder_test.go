package toml

import (
	"bytes"
	"fmt"
	"testing"
)

type path struct {
	segs []string
	v    Var
}

func TestKeyFilter(t *testing.T) {

	tests := []struct {
		paths []path
	}{
		{
			paths: []path{

				{segs: []string{`a`, `b`}, v: TableVar},
				{segs: []string{`a`, `b`, `c`}, v: ArrayVar},

				{segs: []string{`a`, `b`, `c`}, v: ArrayVar},

				{segs: []string{`a`, `b`, `c`, `d`}, v: ArrayVar},
			},
		},

		/*
			{
				paths: []path{

					{segs: []string{`a`, `b`}, v: TableVar},
					{segs: []string{`a`, `b`, `c`}, v: TableVar},

					{segs: []string{`a`, `b`}, v: TableVar},

					{segs: []string{`a`, `b`, `d`}, v: TableVar},

					{segs: []string{`x`}, v: TableVar},

					{segs: []string{`y`}, v: TableVar},

					{segs: []string{`x`, `y`}, v: TableVar},

					{segs: []string{``}, v: TableVar},
				},
			},
		*/
	}

	for _, ts := range tests {

		buf := &bytes.Buffer{}
		fi := &KeyFilter{}

		for idx, p := range ts.paths {
			fi.Push(p.segs, p.v, buf)

			fmt.Printf("_line%v %s\n", idx, buf.Bytes())
		}

		fmt.Printf("___ %s\n", buf.Bytes())
	}
}
