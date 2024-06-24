package jsonredact

import (
	"github.com/stretchr/testify/require"
	"math/rand/v2"
	"testing"
)

func Test_newDFA(t *testing.T) {
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
		{
			name:        "base/do not override general by particular",
			expressions: []string{"a", "a.b", "x.y.z.g.d.g", "x.y.z.g.d", "x"},
			accepted:    []string{"a", "ab", "axy", "x"},
			notAccepted: []string{"c", "fx", "b"},
		},
		{
			name:        "wildcard/all fields",
			expressions: []string{"#", "#"},
			accepted:    []string{"a", "ab", "axy", "x"},
		},
		{
			name:        "wildcard/all fields of an object",
			expressions: []string{"a.#"},
			accepted:    []string{"ab", "axy"},
			notAccepted: []string{"a", "x"},
		},
		{
			name:        "wildcard/one field of all objects",
			expressions: []string{"#.a"},
			accepted:    []string{"aa", "fa", "ya", "aaa"},
			notAccepted: []string{"a", "x", "g"},
		},
		{
			name:        "wildcard/do not override general by particular",
			expressions: []string{"a.#", "a"},
			accepted:    []string{"a", "ab", "axy"},
			notAccepted: []string{"c", "fx", "b"},
		},
		{
			name:        "wildcard/intersection",
			expressions: []string{"a.b", "#.b"},
			accepted:    []string{"ab", "jb", "kb", "kbkkrker", "abbb"},
			notAccepted: []string{"b", "a"},
		},
		{
			name:        "wildcard/intersection",
			expressions: []string{"#.#.#", "#.#"},
			accepted:    []string{"ab", "jb", "kb", "kbkkrker", "abbb"},
			notAccepted: []string{"b", "a"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < 10; i++ {
				rand.Shuffle(len(tt.expressions), func(i, j int) {
					tt.expressions[i], tt.expressions[j] = tt.expressions[j], tt.expressions[i]
				})
				a := newDFA(tt.expressions...)
				for _, input := range tt.accepted {
					require.True(t, accepts(a, input), "dfa=%s input=%s", tt.expressions, input)
				}
				for _, input := range tt.notAccepted {
					require.False(t, accepts(a, input), "dfa=%s input=%s", tt.expressions, input)
				}
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
