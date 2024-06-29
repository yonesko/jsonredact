package jsonredact

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type node struct {
	states     []*state
	isTerminal bool
}

type state struct {
	isTerminal  bool
	transitions map[string]*state
}

func newNode() node {
	return node{}
}

func newState() *state {
	return &state{transitions: map[string]*state{}}
}

func newNDFA(expressions ...string) node {
	if len(expressions) == 0 {
		return newNode()
	}
	states := make([]*state, 0, len(expressions))

	for i := 0; i < len(expressions); i++ {
		states = append(states, build(expression(expressions[i]).splitByPoint()))
	}

	return node{states: states}
}

func (n *node) next(input string, buf []*state) node {
	buf = buf[:0]
	var isTerminal bool
	for _, s := range n.states {
		nextState := s.next(input)
		if nextState != nil {
			buf = append(buf, nextState)
			if nextState.isTerminal {
				isTerminal = true
			}
		}
	}
	return node{states: buf, isTerminal: isTerminal}
}

func (s *state) next(input string) *state {
	automata := s.transitions[input]
	if automata != nil {
		return automata
	}
	automata = s.transitions[`\`+input]
	if (input == "*" || input == "#") && automata != nil {
		return automata
	}
	return s.transitions["#"]
}

func merge(right, left *state, rightToAutomata map[*state]*state, leftToAutomata map[*state]*state) *state {
	commonKeys := make(map[string]bool, len(left.transitions)+len(right.transitions))
	automata := newState()
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
	a := newState()
	if expressions[0] == "*" {
		return buildRecursive(expressions)
	}
	a.transitions[expressions[0]] = build(expressions[1:])
	return a
}

// buildRecursive builds *state from recursive expression, (example *.a.b, *.a.*.b)
func buildRecursive(expressions []string) *state {
	root := newState()
	//defer func() {
	//	if r := recover(); r != nil {
	//		fmt.Printf("expressions='%+v'\n", strings.Join(expressions, " | "))
	//		fmt.Println("Recovered in buildRecursive", r)
	//		fmt.Printf("root='%+v'\n", root)
	//		panic(r)
	//	}
	//}()
	if getNextExpr(0, expressions) != "#" {
		root.transitions["#"] = root
	}
	a := root
	for i := 1; i < len(expressions); i++ {
		if len(expressions) > i+1 && expressions[i+1] == "*" {
			a.transitions[expressions[i]] = buildRecursive(expressions[i+1:])
			return root
		}
		next := newState()
		safeSet(a.transitions, expressions[i], next)
		nextExpr := getNextExpr(i, expressions)
		if i == len(expressions)-1 {
			next.isTerminal = true
		} else {
			if nextExpr != expressions[1] && nextExpr != "#" {
				safeSet(next.transitions, expressions[1], root.transitions[expressions[1]])
			}
			if nextExpr != "#" {
				safeSet(next.transitions, "#", root)
			}
			//next.transitions[expressions[1]] = root.transitions[expressions[1]]
			//next.transitions["#"] = root
		}
		//a.transitions[expressions[i]] = next
		//safeSet(a.transitions, expressions[i], next)
		a = next
	}
	return root
}

func getNextExpr(k int, expressions []string) string {
	for i := k + 1; i < len(expressions); i++ {
		if expressions[i] != "*" {
			return expressions[i]
		}
	}
	return ""
}

func safeSet(m map[string]*state, k string, v *state) {
	if _, ok := m[k]; ok {
		panic("duplicate key " + k)
	}
	m[k] = v
}

func (s *state) string(been map[*state]bool) string {
	buffer := bytes.Buffer{}
	if been[s] {
		return ""
	}
	been[s] = true
	if s.isTerminal {
		return ""
	}
	buffer.WriteString(fmt.Sprintf("state(%p) ", s))
	for k, v := range s.transitions {
		if v.isTerminal {
			buffer.WriteString(fmt.Sprintf("%s -> terminal ", k))
			continue
		}
		buffer.WriteString(fmt.Sprintf("%s -> %p ", k, v))
	}
	buffer.WriteByte('\n')
	for _, v := range s.transitions {
		buffer.WriteString(v.string(been))
	}
	return buffer.String()
}

func (s *state) String() string {
	been := map[*state]bool{}
	str := s.string(been)
	//0x1400012c9c0
	re, err := regexp.Compile(`0x.{11}`)
	if err != nil {
		panic(err)
	}
	pointers := re.FindAllString(str, -1)
	replace := make([]string, 0, len(pointers)*2)
	for i := range pointers {
		replace = append(replace, pointers[i], strconv.Itoa(i))
	}
	return strings.NewReplacer(replace...).Replace(str)
}

func (n node) String() string {
	buffer := bytes.Buffer{}
	buffer.WriteString("isTerminal")
	if n.isTerminal {
		buffer.WriteString(" true")
	} else {
		buffer.WriteString(" false")
	}
	buffer.WriteByte('\n')
	for i, s := range n.states {
		buffer.WriteString(strconv.Itoa(i))
		buffer.WriteByte(':')
		buffer.WriteByte('\n')
		buffer.WriteString(s.String())
	}
	return buffer.String()
}
