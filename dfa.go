package jsonredact

import (
	"bytes"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
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
	if len(left) == 0 {
		return right
	}
	if len(right) == 0 {
		return left
	}
	automata := dfa{}
	fmt.Println(left)
	fmt.Println(right)
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
			if k == "#" && reflect.ValueOf(a).Pointer() == reflect.ValueOf(a[k]).Pointer() {
				automata[k] = automata
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
	a := root
	for i := 1; i < len(expressions); i++ {
		if len(expressions) > i+1 && expressions[i+1] == "*" {
			a[expressions[i]] = buildRecursive(expressions[i+1:])
			return root
		}
		next := dfa{"#": root}
		if i == len(expressions)-1 {
			next["terminal"] = dfa{}
		} else if i == 1 {
			next[expressions[i]] = next
		} else {
			next[expressions[1]] = root[expressions[1]]
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
	if a.isInTerminalState() {
		return ""
	}
	buffer.WriteString(fmt.Sprintf("state(%p) ", a))
	for k, v := range a {
		if v.isInTerminalState() {
			buffer.WriteString(fmt.Sprintf("%s -> terminal ", k))
			continue
		}
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
	s := a.string(been)
	//0x1400012c9c0
	re, err := regexp.Compile(`0x.{11}`)
	if err != nil {
		panic(err)
	}
	//counter := 0
	pointers := re.FindAllString(s, -1)
	replace := make([]string, 0, len(pointers)*2)
	for i := range pointers {
		replace = append(replace, pointers[i], strconv.Itoa(i))
	}
	return strings.NewReplacer(replace...).Replace(s)
}
