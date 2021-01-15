package toml

import (
	"io"
)

type segment struct {
	S    string
	V    Var
	Head bool
}

type KeyFilterPushFunc func(key []string, v Var, w io.StringWriter)

func BaseKeyFilterPushFunc(baseKey []string, keyFilterFunc KeyFilterPushFunc) KeyFilterPushFunc {

	return func(key []string, v Var, w io.StringWriter) {
		keyFilterFunc(append(baseKey, key...), v, w)
	}
}

type KeyFilter struct {
	path        []segment
	notBaseHead bool
}

func (k KeyFilter) closeSegments(beq int, w io.StringWriter) {

	for i := len(k.path) - 1; i >= beq; i-- {

		if k.path[i].V == ArrayVar {
			w.WriteString("}]")
			continue
		}
		w.WriteString("}")
	}
}

func (k KeyFilter) renderKey(key string, w io.StringWriter) {
	w.WriteString(`"`)
	w.WriteString(key)
	w.WriteString(`":`)
}

func (k *KeyFilter) Push(key []string, v Var, w io.StringWriter) {

	if v == BasicVar {
		key = key[:len(key)-1]
	}

	min := len(key)
	if len(k.path) < min {
		min = len(k.path)
	}

	var idx int
	for i := 0; i < min; i++ {
		if key[i] != k.path[i].S {
			break
		}
		idx++
	}

	if idx < len(k.path) {
		k.closeSegments(idx, w)
		k.path = k.path[:idx]
	}

	for i := idx; i < len(key); i++ {

		tv := v
		head := true
		if i < len(key)-1 {
			tv = TableVar
			head = false
		}

		k.path = append(k.path, segment{S: key[i], V: tv, Head: head})
	}

	if v == ArrayVar && idx == len(k.path) && idx == len(key) {
		w.WriteString("},{")
		k.path[len(k.path)-1].Head = true
		return
	}

	if idx > 0 {
		if (idx != len(key) || v != TableVar || k.path[idx-1].V != TableVar) && !k.path[idx-1].Head {
			w.WriteString(",")
		}

	} else if k.notBaseHead {
		w.WriteString(",")
	}

	k.notBaseHead = true

	if v == BasicVar {
		if len(k.path) > 0 {
			k.path[len(k.path)-1].Head = false
		}
	}

	for i := idx; i < len(k.path); i++ {

		k.renderKey(k.path[i].S, w)
		if k.path[i].V == ArrayVar {
			w.WriteString("[{")
		} else {
			w.WriteString("{")
		}
	}

	for i := 0; i < idx; i++ {
		k.path[i].Head = false
	}
}

func (k *KeyFilter) Close(w io.StringWriter) {
	k.closeSegments(0, w)
}
