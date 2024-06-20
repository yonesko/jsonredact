package jsonredact

type dfa map[string]dfa

func newDFA(expressions ...string) dfa {
	return build(expression(expressions[0]).splitByPoint())
}

func (a dfa) next(input string) dfa {
	return a[input]
}

func (a dfa) isInTerminalState() bool {
	return a != nil && len(a) == 0
}

func build(expressions []string) dfa {
	if len(expressions) == 0 {
		return dfa{}
	}
	a := dfa{}
	a[expressions[0]] = build(expressions[1:])
	return a
}
