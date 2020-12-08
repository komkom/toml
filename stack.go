package toml

import (
	"strings"
)

type ArrayKeyStack struct {
	stack []string
}

func (a *ArrayKeyStack) Push(key string, v Var) (toClose []string) {

	for {
		if len(a.stack) == 0 {
			break
		}

		ck := a.stack[len(a.stack)-1]

		if strings.HasPrefix(key, ck) {
			break
		}

		toClose = append(toClose, ck)
		a.stack = a.stack[:len(a.stack)-1]
	}

	if v == ArrayVar {
		if len(a.stack) > 0 && a.stack[len(a.stack)-1] == key {
			return toClose
		}
		a.stack = append(a.stack, key)
	}
	return toClose
}
