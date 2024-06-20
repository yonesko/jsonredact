package jsonredact

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
	return a[input]
}

func (a dfa) isInTerminalState() bool {
	return a != nil && len(a) == 0
}

func merge(left, right dfa) dfa {
	automata := dfa{}
	for k := range left {
		automata[k] = merge(right[k], left[k])
	}
	for k := range right {
		automata[k] = merge(right[k], left[k])
	}
	return automata
}

func build(expressions []string) dfa {
	if len(expressions) == 0 {
		return dfa{}
	}
	a := dfa{}
	a[expressions[0]] = build(expressions[1:])
	return a
}
