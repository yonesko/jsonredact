package jsonredact

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type state struct {
	isTerminal  bool
	transitions map[string]*state
}

func newNode() *state {
	return &state{transitions: map[string]*state{}}
}

func newDFA(expressions ...string) *state {
	if len(expressions) == 0 {
		return newNode()
	}
	automata := build(expression(expressions[0]).splitByPoint())
	fmt.Println(expressions[0])
	fmt.Println(automata)
	fmt.Println("-----")
	for i := 1; i < len(expressions); i++ {
		rightToAutomata := map[*state]*state{}
		leftToAutomata := map[*state]*state{}
		automata = merge(automata, build(expression(expressions[i]).splitByPoint()), rightToAutomata, leftToAutomata)
		fmt.Println(expressions[i])
		fmt.Println(automata)
		fmt.Println("-----")
	}
	return automata
}

func (n *state) next(input string) *state {
	automata := n.transitions[input]
	if automata != nil {
		return automata
	}
	automata = n.transitions[`\`+input]
	if (input == "*" || input == "#") && automata != nil {
		return automata
	}
	return n.transitions["#"]
}

func merge(right, left *state, rightToAutomata map[*state]*state, leftToAutomata map[*state]*state) *state {
	commonKeys := make(map[string]bool, len(left.transitions)+len(right.transitions))
	automata := newNode()
	for k := range left.transitions {
		if right.transitions[k] == nil {
			automata.transitions[k] = left.transitions[k]
		} else {
			commonKeys[k] = true
		}
	}
	for k := range right.transitions {
		if left.transitions[k] == nil {
			automata.transitions[k] = right.transitions[k]
		} else {
			commonKeys[k] = true
		}
	}
	rightToAutomata[right] = automata
	leftToAutomata[left] = automata
	for k := range commonKeys {
		r := right.next(k)
		l := left.next(k)
		if r.isTerminal {
			automata.transitions[k] = r
			continue
		}
		if l.isTerminal {
			automata.transitions[k] = l
			continue
		}
		//check recursion
		if rightToAutomata[r] != nil && rightToAutomata[r] == leftToAutomata[l] {
			automata.transitions[k] = rightToAutomata[r]
			continue
		}
		automata.transitions[k] = merge(r, l, rightToAutomata, leftToAutomata)
	}
	return automata
}

func build(expressions []string) *state {
	if len(expressions) == 0 {
		return &state{isTerminal: true}
	}
	a := newNode()
	if expressions[0] == "*" {
		return buildRecursive(expressions)
	}
	a.transitions[expressions[0]] = build(expressions[1:])
	return a
}

// buildRecursive builds *state from recursive expression, (example *.a.b, *.a.*.b)
func buildRecursive(expressions []string) *state {
	root := newNode()
	root.transitions["#"] = root
	a := root
	for i := 1; i < len(expressions); i++ {
		if len(expressions) > i+1 && expressions[i+1] == "*" {
			a.transitions[expressions[i]] = buildRecursive(expressions[i+1:])
			return root
		}
		next := newNode()
		if i == len(expressions)-1 {
			next.isTerminal = true
		} else if i == 1 {
			next.transitions[expressions[i]] = next
			next.transitions["#"] = root
		} else {
			next.transitions[expressions[1]] = root.transitions[expressions[1]]
			next.transitions["#"] = root
		}
		a.transitions[expressions[i]] = next
		a = next
	}
	return root
}

func (n *state) string(been map[*state]bool) string {
	buffer := bytes.Buffer{}
	if been[n] {
		return ""
	}
	been[n] = true
	if n.isTerminal {
		return ""
	}
	buffer.WriteString(fmt.Sprintf("state(%p) ", n))
	for k, v := range n.transitions {
		if v.isTerminal {
			buffer.WriteString(fmt.Sprintf("%s -> terminal ", k))
			continue
		}
		buffer.WriteString(fmt.Sprintf("%s -> %p ", k, v))
	}
	buffer.WriteByte('\n')
	for _, v := range n.transitions {
		buffer.WriteString(v.string(been))
	}
	return buffer.String()
}

func (n *state) String() string {
	been := map[*state]bool{}
	s := n.string(been)
	//0x1400012c9c0
	re, err := regexp.Compile(`0x.{11}`)
	if err != nil {
		panic(err)
	}
	pointers := re.FindAllString(s, -1)
	replace := make([]string, 0, len(pointers)*2)
	for i := range pointers {
		replace = append(replace, pointers[i], strconv.Itoa(i))
	}
	return strings.NewReplacer(replace...).Replace(s)
}
