package jsonredact

import (
	"bytes"
	"github.com/tidwall/gjson"
	"strconv"
)

type Redactor struct {
	pairs    []replacerPair
	automata node
}

/*
User '.' as separator of objects and arrays.
Use '#' as wildcard for any key or array index.
Use '*' to apply right expression to all object keys recursively. (makes redactor walk the whole json)
User '\' to escape control symbols above.
*/
func NewRedactor(expressions []string, handler func(string) string) Redactor {
	pair := replacerPair{handler: handler, expressions: expressions}
	return Redactor{automata: newNDFA(pair), pairs: []replacerPair{pair}}
}

func NewEmptyRedactor() Redactor {
	return Redactor{}
}

func (r Redactor) And(expressions []string, handler func(string) string) Redactor {
	pairs := append(r.pairs, replacerPair{handler: handler, expressions: expressions})
	return Redactor{automata: newNDFA(pairs...), pairs: pairs}
}

func (r Redactor) Redact(json string) string {
	if len(r.automata.states) == 0 {
		return json
	}
	buffer := &lazyBuffer{originalJson: json}
	r.redact(json, r.automata, buffer, 0)
	return buffer.String()
}

type lazyBuffer struct {
	buf          *bytes.Buffer
	originalJson string
}

func (b *lazyBuffer) WriteByte(c byte) error {
	if b.buf == nil {
		return nil
	}
	return b.buf.WriteByte(c)
}

func (b *lazyBuffer) WriteString(s string) (int, error) {
	if b.buf == nil {
		return 0, nil
	}
	return b.buf.WriteString(s)
}

func (b *lazyBuffer) String() string {
	if b.buf == nil {
		return b.originalJson
	}
	return b.buf.String()
}

func (r Redactor) redact(json string, automata node, buf *lazyBuffer, offset int) {
	root := gjson.Parse(json)
	if root.IsArray() {
		_ = buf.WriteByte('[')
	} else {
		_ = buf.WriteByte('{')
	}
	var index int
	statesBuf := make([]*state, 0, 16)
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
		next := automata.next(keyStr, statesBuf)
		if next.isTerminal {
			if buf.buf == nil {
				buf.buf = bytes.NewBuffer(make([]byte, 0, len(buf.originalJson)))
				_, _ = buf.WriteString(buf.originalJson[:offset+value.Index])
			}
			_ = buf.WriteByte('"')
			_, _ = buf.WriteString(next.handler(value.Raw))
			_ = buf.WriteByte('"')
			return true
		}
		if len(next.states) == 0 || (!value.IsObject() && !value.IsArray()) {
			_, _ = buf.WriteString(value.Raw)
			return true
		}
		r.redact(value.Raw, next, buf, offset+value.Index)
		return true
	})
	if root.IsArray() {
		_ = buf.WriteByte(']')
	} else {
		_ = buf.WriteByte('}')
	}
}
