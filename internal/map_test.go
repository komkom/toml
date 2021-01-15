package toml

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMap(t *testing.T) {

	m := Map{m: make(map[string]Map)}

	key := []string{`1`, `2`, `3`}

	ok := m.Set(key, BasicVar)
	assert.True(t, ok)

	r, ok := m.Get(key)
	assert.True(t, ok)

	assert.Equal(t, BasicVar, r.Var)
	assert.Nil(t, r.m)

	ok = m.Set(key[:2], BasicVar)
	assert.False(t, ok)

	ok = m.Clear(key[:2])
	assert.True(t, ok)

	_, ok = m.Get(key)
	assert.False(t, ok)

	ok = m.Set(key, BasicVar)
	assert.True(t, ok)

	_, ok = m.Get(key)
	assert.True(t, ok)
}
