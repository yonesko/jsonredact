package jsonredact

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"math/rand/v2"
	"regexp"
	"strings"
	"sync"
	"testing"
)

func Test_newNDFA(t *testing.T) {
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
		{
			name:        "recursive/one key",
			expressions: []string{"*.a"},
			accepted:    []string{"aaaa", "htbgvfa", "ba", "ca"},
			notAccepted: []string{"zwexrcvtb", "ygvb", "l"},
		},
		{
			name:        "recursive/one two sized key",
			expressions: []string{"*.a.b"},
			accepted:    []string{"ab", "xxxab", "xxaxxxbbaaabbb", "aaaab"},
			notAccepted: []string{"x", "aa", "bb", "b", "ba"},
		},
		{
			name:        "recursive/long one key",
			expressions: []string{"*.a.b.c.d"},
			accepted:    []string{"aaaaabcd", "abcd", "ohtygabcd", "abcdabcd", "aaabcd"},
			notAccepted: []string{"abc", "bcd", "cd", "aabbccdd", "abccd"},
		},
		{
			name:        "recursive/several stars in key",
			expressions: []string{"*.a.*.b"},
			accepted:    []string{"xxaxxxxb", "aaaaabcd", "abcd", "ohtygab", "axxxb"},
			notAccepted: []string{"ac", "bcd", "bb"},
		},
		{
			name:        "recursive/several keys",
			expressions: []string{"*.a", "*.b"},
			accepted:    []string{"xxxxxa", "xxxxxb", "a", "b", "ab", "ba"},
			notAccepted: []string{"xxx", "ghfd", "tttt"},
		},
		{
			name:        "recursive/several long keys",
			expressions: []string{"*.a.b", "*.a.c"},
			accepted:    []string{"xxab", "xxabbb", "xxabxx", "xxacxx", "ab", "ac", "abc"},
			notAccepted: []string{"xxx", "axc", "axb"},
		},
		{
			name:        "recursive/intersection",
			expressions: []string{"*.a.b", "*.a.b"},
			accepted:    []string{"ab", "xxab", "abcc", "xxabxx"},
			notAccepted: []string{"xxx", "axb", "ba"},
		},
		{
			name:        "recursive/intersection",
			expressions: []string{"*.a.b", "a.b"},
			accepted:    []string{"ab", "xxab", "abcc", "xxabxx"},
			notAccepted: []string{"xxx", "axb", "ba"},
		},
		{
			name:        "recursive/intersection",
			expressions: []string{"*.a.b", "a"},
			accepted:    []string{"ab", "xxab", "abcc", "xxabxx", "a", "aaa", "axxx"},
			notAccepted: []string{"xxx", "ba", "b", "c"},
		},
		{
			name:        "recursive/intersection",
			expressions: []string{"*.a", "x"},
			accepted:    []string{"x", "xxxxa", "yyyya", "a", "aaa", "axa"},
			notAccepted: []string{"b", "c", "tt"},
		},
		{
			name:        "recursive/intersection",
			expressions: []string{"*.a", "b.c"},
			accepted:    []string{"bc", "xxxxa", "yyyya", "a", "aaa", "axa", "bbbba", "bcccc"},
			notAccepted: []string{"b", "c", "tt", "bbbbc"},
		},
		{
			name:        "recursive/intersection",
			expressions: []string{"*.a", "#.a", "a.#", "a.*.a"},
			accepted:    []string{"aaaa", "htbgvfa", "ba", "ca"},
			notAccepted: []string{"zwexrcvtb", "ygvb", "l"},
		},
		{
			name:        "recursive/intersection",
			expressions: []string{"*.a.b", "a.#"},
			accepted:    []string{"ab", "xab", "xabx", "ax", "ac", "acab"},
			notAccepted: []string{"x", "a", "b", "ba", "bx"},
		},
		{
			name:        "recursive/with wildcard",
			expressions: []string{"*.c.d.#"},
			accepted:    []string{"dcdc"},
			notAccepted: []string{"cd", "c", "d", "cc"},
		},
		{
			name:        "recursive/with wildcard",
			expressions: []string{"*.#"},
			accepted:    []string{generateInput(), generateInput(), generateInput()},
			notAccepted: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			once := sync.Once{}
			for i := 0; i < 10; i++ {
				rand.Shuffle(len(tt.expressions), func(i, j int) {
					tt.expressions[i], tt.expressions[j] = tt.expressions[j], tt.expressions[i]
				})
				a := newNDFA(tt.expressions...)
				once.Do(func() {
					fmt.Println(tt.expressions)
					fmt.Println(&a)
				})
				for _, input := range tt.accepted {
					require.True(t, accepts(a, input), "ndfa=%s input=%s", tt.expressions, input)
				}
				for _, input := range tt.notAccepted {
					require.False(t, accepts(a, input), "ndfa=%s input=%s", tt.expressions, input)
				}
			}
		})
	}
}

func accepts(a node, input string) bool {
	for _, v := range input {
		a = a.next(string(v), nil)
		if len(a.states) == 0 {
			return false
		}
		if a.isTerminal {
			return true
		}
	}
	return false
}

func buildRegex(expressions []string) *regexp.Regexp {
	expr := make([]string, len(expressions))
	for i := 0; i < len(expressions); i++ {
		expr[i] = toRegex(expressions[i])
	}

	compile, err := regexp.Compile(strings.Join(expr, "|"))
	if err != nil {
		panic(err)
	}

	return compile
}

func toRegex(expression string) string {
	if expression[0] != '*' {
		expression = "^" + expression
	}
	expression = strings.ReplaceAll(expression, ".", "")
	expression = strings.ReplaceAll(expression, "#", ".")
	expression = strings.ReplaceAll(expression, "*", ".*")
	return expression
}

func TestRandom(t *testing.T) {
	t.Parallel()
	expressions := generateExpressions()
	regex := buildRegex(expressions)
	ndfa := newNDFA(expressions...)
	for i := 0; i < 100e3; i++ {
		input := generateInput()
		expected := regex.MatchString(input)
		actual := accepts(ndfa, input)
		if expected != actual {
			//fmt.Println(&ndfa)
			fmt.Printf("input='%+v'\n", input)
			fmt.Printf("expressions='%+v'\n", strings.Join(expressions, " | "))
			fmt.Printf("actual='%+v'\n", actual)
			fmt.Printf("expected='%+v'\n", expected)
			t.FailNow()
		}
	}
}

func generateExpressions() []string {
	var expressions []string
	for i := 0; i < rand.IntN(10)+1; i++ {
		expressions = append(expressions, generateExpression())
	}
	return expressions
}

func generateExpression() string {
	var letters = []rune("abcd#*")

	expr := ""
	for i := 0; i < rand.IntN(10)+1; i++ {
		v := string(letters[rand.IntN(len(letters))])
		if i != 0 {
			expr += "."
		}
		expr += v
		if v == "*" {
			expr += "." + string(letters[rand.IntN(len(letters[:len(letters)-1]))])
		}
	}
	return expr
}

func generateInput() string {
	var letters = []rune("abcd")

	input := ""
	for i := 0; i < rand.IntN(10)+1; i++ {
		input += string(letters[rand.IntN(len(letters))])
	}
	return input
}
