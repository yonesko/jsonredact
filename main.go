package jsonredact

import (
	"bytes"
	"github.com/tidwall/gjson"
)

type Redactor struct {
	automata dfa
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
	s := state{json: json, automata: r.automata, handler: r.handler, buf: bytes.NewBuffer(make([]byte, 0, len(json)))}
	s.redact()
	return s.buf.String()
}

type state struct {
	json     string
	automata dfa
	handler  func(string) string
	index    int
	buf      *bytes.Buffer
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

		automata := s.automata.next(keyStr)
		if automata.isInTerminalState() {
			_ = s.buf.WriteByte('"')
			_, _ = s.buf.WriteString(s.handler(value.Raw))
			_ = s.buf.WriteByte('"')
			return true
		}
		if automata == nil || (!value.IsObject() && !value.IsArray()) {
			_, _ = s.buf.WriteString(value.Raw)
			return true
		}
		(&state{json: value.Raw, automata: automata, handler: s.handler, buf: s.buf}).redact()
		return true
	})
	if root.IsArray() {
		_ = s.buf.WriteByte(']')
	} else {
		_ = s.buf.WriteByte('}')
	}
}
