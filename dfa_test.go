package jsonredact

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_newDFA(t *testing.T) {
	type testCase struct {
		input string
		want  bool
	}
	tests := []struct {
		name        string
		expressions []string
		testCases   []testCase
	}{
		{
			name:        "base/single",
			expressions: []string{"a"},
			testCases: []testCase{
				{input: "ab", want: true}, {input: "a", want: true}, {input: "aa", want: true}, {input: "b"},
				{input: "bb"}, {input: "ba"},
			},
		},
		{
			name:        "base/3",
			expressions: []string{"a.b.c"},
			testCases: []testCase{
				{input: "abc", want: true}, {input: "abcc", want: true}, {input: "b"}, {input: "bb"}, {input: "ab"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newDFA(tt.expressions...)
			for _, c := range tt.testCases {
				require.Equal(t, c.want, accepts(a, c.input), "dfa=%s input=%s", tt.expressions, c.input)
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
