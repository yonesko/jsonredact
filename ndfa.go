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
	handler    func(string) string
}

type state struct {
	isTerminal  bool
	transitions map[string]*state
	handler     func(string) string
}

func newNode() node {
	return node{}
}

func newState() *state {
	return &state{transitions: map[string]*state{}}
}

func newNDFA(handler func(string) string, expressions ...string) node {
	if len(expressions) == 0 {
		return newNode()
	}
	states := make([]*state, 0, len(expressions))

	for i := 0; i < len(expressions); i++ {
		states = append(states, build(handler, expression(expressions[i]).splitByPoint()))
	}

	return node{states: states}
}

func (n node) next(input string, buf []*state) node {
	buf = buf[:0]
	var isTerminal bool
	var handler func(string) string
	for _, s := range n.states {
		nextState, nextState2 := s.next(input)
		if nextState != nil {
			buf = append(buf, nextState)
			if nextState.isTerminal {
				isTerminal = true
			}
			handler = nextState.handler
		}
		if nextState2 != nil {
			buf = append(buf, nextState2)
			if nextState2.isTerminal {
				isTerminal = true
			}
			handler = nextState2.handler
		}
	}
	if n.isTerminal == isTerminal && len(buf) == 1 && len(n.states) == 1 && buf[0] == n.states[0] {
		return n
	}
	return node{handler: handler, states: buf, isTerminal: isTerminal}
}

func (s *state) nextByKey(input string) *state {
	if s.transitions[input] != nil {
		return s.transitions[input]
	}
	if (input == "*" || input == "#") && s.transitions[`\`+input] != nil {
		return s.transitions[`\`+input]
	}
	return nil
}

func (s *state) next(input string) (*state, *state) {
	return s.nextByKey(input), s.transitions["#"]
}

func build(handler func(string) string, expressions []string) *state {
	if len(expressions) == 0 {
		return &state{isTerminal: true, handler: handler}
	}
	a := newState()
	a.handler = handler
	if expressions[0] == "*" {
		a.transitions["#"] = a
		a.transitions[expressions[1]] = build(handler, expressions[2:])
		return a
	}
	a.transitions[expressions[0]] = build(handler, expressions[1:])
	return a
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
