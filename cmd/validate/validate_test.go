package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {

	err := run(bytes.NewBufferString(`[table]
	key=1
	key2=2`))
	require.NoError(t, err)
}
