package jsonredact

import (
	"bytes"
	"fmt"
	"reflect"
)

type dfa map[string]dfa

func newDFA(expressions ...string) dfa {
	if len(expressions) == 0 {
		return dfa{}
	}
	automata := dfa{}
	for _, exp := range expressions {
		automata = merge(automata, build(expression(exp).splitByPoint()))
	}
	return automata
}

func (a dfa) next(input string) dfa {
	automata := a[input]
	if automata != nil {
		return automata
	}
	return a["#"]
}

func (a dfa) isInTerminalState() bool {
	return a["terminal"] != nil
}

func merge(left, right dfa) dfa {
	if left == nil {
		return right
	}
	if right == nil {
		return left
	}
	automata := dfa{}
	for _, a := range []dfa{left, right} {
		for k := range a {
			r := right.next(k)
			l := left.next(k)
			if r.isInTerminalState() {
				automata[k] = r
				continue
			}
			if l.isInTerminalState() {
				automata[k] = l
				continue
			}
			automata[k] = merge(r, l)
		}
	}
	return automata
}

func build(expressions []string) dfa {
	if len(expressions) == 0 {
		return dfa{"terminal": dfa{}}
	}
	a := dfa{}
	if expressions[0] == "*" {
		return buildRecursive(expressions)
	}
	a[expressions[0]] = build(expressions[1:])
	return a
}

// buildRecursive builds dfa from recursive expression, (example *.a.b, *.a.*.b)
func buildRecursive(expressions []string) dfa {
	root := dfa{}
	root["#"] = root
	//root[expressions[1]] = dfa{}
	a := root
	for i := 1; i < len(expressions); i++ {
		if expressions[i] == "*" {
			a[expressions[i-1]] = buildRecursive(expressions[i:])
			return root
		}
		next := dfa{"#": root}
		if i > 1 {
			next[expressions[1]] = root[expressions[1]]
		}
		if i == len(expressions)-1 {
			next["terminal"] = dfa{}
		}
		a[expressions[i]] = next
		a = next
	}
	return root
}

func (a dfa) string(been map[uintptr]bool) string {
	buffer := bytes.Buffer{}
	ptr := reflect.ValueOf(a).Pointer()
	if been[ptr] {
		return ""
	}
	been[ptr] = true
	if len(a) == 0 {
		return ""
	}
	buffer.WriteString(fmt.Sprintf("state(%p) ", a))
	for k, v := range a {
		buffer.WriteString(fmt.Sprintf("%s -> %p ", k, v))
	}
	buffer.WriteByte('\n')
	for _, v := range a {
		buffer.WriteString(v.string(been))
	}
	return buffer.String()
}

func (a dfa) String() string {
	been := map[uintptr]bool{}
	return a.string(been)
}
