package jsonredact

import (
	"bytes"
	"github.com/tidwall/gjson"
	"strconv"
)

type Redactor struct {
	automata node
	handler  func(string) string
}

/*
User '.' as separator of objects and arrays.
Use '#' as wildcard for any key or array index.
Use '*' to apply right expression to all object keys recursively.
User '\' to escape control symbols above.
*/
func NewRedactor(keySelectors []string, handler func(string) string) Redactor {
	return Redactor{handler: handler, automata: newDFA(keySelectors...)}
}

func Redact(json string, keySelectors []string, handler func(string) string) string {
	return NewRedactor(keySelectors, handler).Redact(json)
}

func (r Redactor) Redact(json string) string {
	if len(r.automata.states) == 0 {
		return json
	}
	buffer := bytes.NewBuffer(make([]byte, 0, len(json)))
	r.redact(json, r.automata, buffer)
	return buffer.String()
}

func (r Redactor) redact(json string, automata node, buf *bytes.Buffer) {
	root := gjson.Parse(json)
	if root.IsArray() {
		_ = buf.WriteByte('[')
	} else {
		_ = buf.WriteByte('{')
	}
	var index int
	root.ForEach(func(key, value gjson.Result) bool {
		keyStr := key.Str
		if root.IsArray() {
			keyStr = strconv.Itoa(index)
		}
		if index != 0 {
			_ = buf.WriteByte(',')
		}
		index++
		_, _ = buf.WriteString(key.Raw)
		if !root.IsArray() {
			_ = buf.WriteByte(':')
		}

		next := automata.next(keyStr)
		if next.isTerminal {
			_ = buf.WriteByte('"')
			_, _ = buf.WriteString(r.handler(value.Raw))
			_ = buf.WriteByte('"')
			return true
		}
		if len(next.states) == 0 || (!value.IsObject() && !value.IsArray()) {
			_, _ = buf.WriteString(value.Raw)
			return true
		}
		r.redact(value.Raw, next, buf)
		return true
	})
	if root.IsArray() {
		_ = buf.WriteByte(']')
	} else {
		_ = buf.WriteByte('}')
	}
}
