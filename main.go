package jsonredact

import (
	"bytes"
	"github.com/tidwall/gjson"
	"strings"
)

type Redactor struct {
	selectorForest selectorForest
	handler        func(string) string
}

/*
User '.' as separator of objects and arrays.
Use '#' as wildcard for any key or array index.
Use '*' to apply right expression to all object keys recursively.
User '\' to escape control symbols above.
*/
func NewRedactor(keySelectors []string, handler func(string) string) Redactor {
	return Redactor{handler: handler, selectorForest: parseSelector(keySelectors)}
}

func Redact(json string, keySelectors []string, handler func(string) string) string {
	return NewRedactor(keySelectors, handler).Redact(json)
}

func (r Redactor) Redact(json string) string {
	if len(r.selectorForest) == 0 {
		return json
	}
	if !isContainsFields(json, r.selectorForest) {
		return json
	}
	s := state{json: json, selectorForest: r.selectorForest, handler: r.handler, buf: bytes.NewBuffer(make([]byte, 0, len(json)))}
	s.redact()
	return s.buf.String()
}

func isContainsFields(json string, forest selectorForest) bool {
	containsFields := false
	gjson.Parse(json).ForEach(func(key, value gjson.Result) bool {
		if forest.selectForest(key) != nil {
			containsFields = true
			return false
		}
		return true
	})
	return containsFields
}

type selectorForest map[string]selectorForest

func (forest selectorForest) String() string {
	return forest.string("")
}

func (forest selectorForest) string(prefix string) string {
	buffer := bytes.Buffer{}
	for k, v := range forest {
		if len(v) > 0 {
			buffer.WriteString(v.string(prefix + k + `->`))
		} else {
			buffer.WriteString(prefix + k + "\n")
		}
	}
	return buffer.String()
}

func (forest selectorForest) add(str string) {
	var fi = forest
	for _, val := range parse(str) {
		_, ok := fi[val]
		if !ok {
			fi[val] = map[string]selectorForest{}
		}
		fi = fi[val]
	}
}

// parse selector expression (field1.fie\.ld2) to elements [field1,fie.ld2]
func parse(s string) []string {
	var elems []string
	builder := strings.Builder{}
	wasEscape := false
	for _, c := range []rune(s) {
		switch c {
		case '#', '*':
			if wasEscape {
				wasEscape = false
				_, _ = builder.WriteString(`\`)
			}
			_, _ = builder.WriteRune(c)
		case '\\':
			wasEscape = true
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

func parseSelector(keySelectors []string) selectorForest {
	f := selectorForest{}
	for _, k := range keySelectors {
		f.add(k)
	}
	return f
}

type state struct {
	json           string
	selectorForest selectorForest
	handler        func(string) string
	index          int
	buf            *bytes.Buffer
}

func (forest selectorForest) selectForest(key gjson.Result) selectorForest {
	f, ok := forest[key.String()]
	if ok {
		return f
	}
	f, ok = forest[`\`+key.String()]
	if ok {
		return f
	}
	f, ok = forest["#"]
	if ok {
		return f
	}
	f, ok = forest["*"]
	if ok {
		f["*"] = f
		return f
	}
	return f
}

func (s *state) redact() {
	parent := gjson.Parse(s.json)
	if parent.IsArray() {
		_ = s.buf.WriteByte('[')
	} else {
		_ = s.buf.WriteByte('{')
	}
	parent.ForEach(func(key, value gjson.Result) bool {
		if s.index != 0 {
			_ = s.buf.WriteByte(',')
		}
		s.index++
		_, _ = s.buf.WriteString(key.Raw)
		if !parent.IsArray() {
			_ = s.buf.WriteByte(':')
		}
		if forest := s.selectorForest.selectForest(key); forest == nil {
			_, _ = s.buf.WriteString(value.Raw)
		} else if len(forest) != 0 {
			if value.IsObject() || value.IsArray() {
				(&state{json: value.Raw, selectorForest: forest, handler: s.handler, buf: s.buf}).redact()
			} else {
				_, _ = s.buf.WriteString(value.Raw)
			}
		} else {
			str := s.handler(value.Raw)
			_ = s.buf.WriteByte('"')
			_, _ = s.buf.WriteString(str)
			_ = s.buf.WriteByte('"')
		}
		return true
	})
	if parent.IsArray() {
		_ = s.buf.WriteByte(']')
	} else {
		_ = s.buf.WriteByte('}')
	}
}
