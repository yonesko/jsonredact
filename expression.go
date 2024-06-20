package jsonredact

import "strings"

type expression string

func (e expression) splitByPoint() []string {
	var elems []string
	builder := strings.Builder{}
	wasEscape := false
	for _, c := range []rune(e) {
		switch c {
		case '#', '*':
			if wasEscape {
				wasEscape = false
				if len(e) == 2 { //just \*
					_, _ = builder.WriteString(`\`)
				}
			}
			_, _ = builder.WriteRune(c)
		case '\\':
			if wasEscape {
				wasEscape = false
				_, _ = builder.WriteString(`\`)
			} else {
				wasEscape = true
			}
		case '.':
			if wasEscape {
				_, _ = builder.WriteRune(c)
				wasEscape = false
			} else {
				elems = append(elems, builder.String())
				builder.Reset()
			}
		default:
			_, _ = builder.WriteRune(c)
		}
	}
	return append(elems, builder.String())
}
