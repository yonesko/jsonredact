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
	f := r.selectorForest
	//fmt.Println(f.String())
	if len(f) == 0 {
		return json
	}
	//if !isContainsFields(json, f) {
	//	return json
	//}
	s := state{json: json, selectorForest: f, handler: r.handler, buf: bytes.NewBuffer(make([]byte, 0, len(json)))}
	s.redact()
	return s.buf.String()
}

//func isContainsFields(json string, forest selectorForest) bool {
//	containsFields := false
//	gjson.Parse(json).ForEach(func(key, value gjson.Result) bool {
//		if forest.selectForest(key) != nil {
//			containsFields = true
//			return false
//		}
//		return true
//	})
//	return containsFields
//}

// splitSelectorExpression splits selector expression (field1.fie\.ld2) to elements [field1,fie.ld2]
func splitSelectorExpression(s string) []string {
	var elems []string
	builder := strings.Builder{}
	wasEscape := false
	for _, c := range []rune(s) {
		switch c {
		case '#', '*':
			if wasEscape {
				wasEscape = false
				if len(s) == 2 { //just \*
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

func parseSelector(keySelectors []string) selectorForest {
	result := selectorForest{}
	for _, k := range keySelectors {
		f := selectorForest{}
		f.add(k)
		result.mergeWith(f)
	}
	return result
}

type state struct {
	json           string
	selectorForest selectorForest
	handler        func(string) string
	index          int
	buf            *bytes.Buffer
}

func (s *state) redact() {
	root := gjson.Parse(s.json)
	if root.IsArray() {
		_ = s.buf.WriteByte('[')
	} else {
		_ = s.buf.WriteByte('{')
	}
	root.ForEach(func(key, value gjson.Result) bool {
		keyStr := key.String()
		if s.index != 0 {
			_ = s.buf.WriteByte(',')
		}
		s.index++
		_, _ = s.buf.WriteString(key.Raw)
		if !root.IsArray() {
			_ = s.buf.WriteByte(':')
		}

		if (s.selectorForest[keyStr] != nil && len(s.selectorForest[keyStr]) == 0) ||
			(s.selectorForest["#"] != nil && len(s.selectorForest["#"]) == 0) {
			_ = s.buf.WriteByte('"')
			_, _ = s.buf.WriteString(s.handler(value.Raw))
			_ = s.buf.WriteByte('"')
			return true
		}
		if s.selectorForest[keyStr] == nil && s.selectorForest["#"] == nil || (!value.IsObject() && !value.IsArray()) {
			_, _ = s.buf.WriteString(value.Raw)
			return true
		}
		f := selectorForest{}
		f.mergeWith(s.selectorForest[keyStr])
		f.mergeWith(s.selectorForest["#"])
		(&state{json: value.Raw, selectorForest: f, handler: s.handler, buf: s.buf}).redact()
		return true
	})
	if root.IsArray() {
		_ = s.buf.WriteByte(']')
	} else {
		_ = s.buf.WriteByte('}')
	}
}
