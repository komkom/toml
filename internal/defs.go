package toml

import (
	"strings"
)

type Var string

var (
	NodefVar         Var = `nodef-var`
	BasicVar         Var = `basic-var`
	TableVar         Var = `table-var`
	ImplicitTableVar Var = `implicit-table-var`
	ArrayVar         Var = `array-var`
)

type DefineFunc func(key []string, v Var) bool

type Defs struct {
	m             Map
	arrayKeyStack *ArrayKeyStack
	keyFilter     *KeyFilter
}

func MakeDefs() Defs {
	return Defs{m: Map{m: make(map[string]Map)},
		arrayKeyStack: &ArrayKeyStack{},
		keyFilter:     &KeyFilter{},
	}
}

func (d Defs) Define(key []string, insertTable []string, v Var) bool {

	if len(key) == 0 {
		return false
	}

	ok := d.m.Set(key, insertTable, v)
	if !ok {
		return false
	}

	fullKey := strings.Join(key, "\n")

	toClose := d.arrayKeyStack.Push(fullKey, v)
	for _, k := range toClose {

		key := strings.Split(k, "\n")
		ok = d.m.Clear(key)
		if !ok {
			return false
		}
	}

	return true
}
