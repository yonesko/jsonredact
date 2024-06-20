package jsonredact

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_newDFA(t *testing.T) {
	type testCase struct {
		input string
	}
	tests := []struct {
		name        string
		expressions []string
		accepted    []string
		notAccepted []string
	}{
		{
			name:        "base/single one sized",
			expressions: []string{"a"},
			accepted:    []string{"ab", "a", "aa"},
			notAccepted: []string{"b", "bb", "ba"},
		},
		{
			name:        "base/single of 3",
			expressions: []string{"a.b.c"},
			accepted:    []string{"abc", "abcccc", "abcc"},
			notAccepted: []string{"b", "bb", "ab"},
		},
		{
			name:        "base/several one sized",
			expressions: []string{"a", "b"},
			accepted:    []string{"a", "b", "aaaa", "bbbb", "ab", "ba"},
			notAccepted: []string{"x", "c"},
		},
		{
			name:        "base/several different sizes",
			expressions: []string{"a", "b.a", "c.g.d"},
			accepted:    []string{"a", "ba", "cgd", "cgddd"},
			notAccepted: []string{"b", "cg", "gddd"},
		},
		{
			name:        "base/several different sizes with intersection",
			expressions: []string{"a.c", "a.b", "a.x.y"},
			accepted:    []string{"ac", "ab", "axy"},
			notAccepted: []string{"c", "ax", "b"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newDFA(tt.expressions...)
			for _, input := range tt.accepted {
				require.True(t, accepts(a, input), "dfa=%s input=%s", tt.expressions, input)
			}
			for _, input := range tt.notAccepted {
				require.False(t, accepts(a, input), "dfa=%s input=%s", tt.expressions, input)
			}
		})
	}
}

func accepts(a dfa, input string) bool {
	for _, v := range input {
		a = a.next(string(v))
		if a.isInTerminalState() {
			return true
		}
	}
	return false
}
