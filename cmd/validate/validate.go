package main

import (
	"fmt"
	"io"
	"os"

	toml "github.com/komkom/toml/internal"
)

type NoopsWriter struct{}

func (NoopsWriter) WriteString(s string) (int, error) {
	return 0, nil
}

func (NoopsWriter) WriteRune(r rune) (int, error) {
	return 0, nil
}

func mail() {
	err := run(os.Stdin)
	if err != nil {
		fmt.Printf("error: %v\n", err.Error())
		os.Exit(1)
	}
	fmt.Printf("toml is valid")
}

func run(reader io.Reader) error {
	f := toml.NewFilter(NoopsWriter{})
	_, err := io.Copy(f, reader)
	if err != nil {
		return err
	}

	err = f.WriteRune('\n')
	if err != nil {
		return err
	}

	err = f.WriteRune(toml.EOF)
	if err != nil {
		return err
	}

	return nil
}
