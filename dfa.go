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
	for k, v := range left {
		if right[k] == nil {
			automata[k] = v
		}
	}
	for k, v := range right {
		if left[k] == nil {
			automata[k] = v
		}
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
