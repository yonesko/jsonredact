package jsonredact

import (
	"bytes"
	"container/list"
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
Use '*' to apply right expression to all object keys recursively. (makes redactor walk the whole json)
User '\' to escape control symbols above.
*/
func NewRedactor(expressions []string, handler func(string) string) Redactor {
	return Redactor{handler: handler, automata: newNDFA(expressions...)}
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

const (
	matchingStateRedact int = iota
	matchingStateSkip   int = iota
	matchingStateVisit  int = iota
)

type redactingListener struct {
	noopListener
	automata          node
	buf               *lazyBuffer
	matchingState     int
	nodes             list.List
	handler           func(string) string
	statesBuf         []*state
	depth             int
	lastMatchingDepth int
}

func (r *redactingListener) EnterMemberKey(ctx memberContext) {
	if r.matchingState != matchingStateVisit {
		return
	}
	if ctx.index != 0 {
		r.buf.WriteByte(',')
	}
}
func (r *redactingListener) ExitMemberKey(ctx memberContext) {
	if r.matchingState != matchingStateVisit {
		return
	}
	r.buf.WriteString(ctx.key)
	r.buf.WriteByte(':')
	//match
	next := r.automata.next(ctx.key[1:len(ctx.key)-1], r.statesBuf)
	if next.isTerminal {
		r.matchingState = matchingStateRedact
		r.lastMatchingDepth = r.depth
		return
	}
	if len(next.states) == 0 {
		r.matchingState = matchingStateSkip
		r.lastMatchingDepth = r.depth
		return
	}
	r.matchingState = matchingStateVisit
}

func (r *redactingListener) ExitMemberValue(ctx memberContext) {
	if r.matchingState != matchingStateVisit && r.depth != r.lastMatchingDepth {
		return
	}
	var value string
	switch r.matchingState {
	case matchingStateSkip:
		value = ctx.value
	case matchingStateRedact:
		value = r.handler(ctx.value)
	case matchingStateVisit:
		return
	}
	r.buf.WriteString(value)
	r.matchingState = matchingStateVisit
	r.lastMatchingDepth = -1
}

func (r *redactingListener) EnterObject(ctx objectContext) {
	r.depth++
	if r.matchingState != matchingStateVisit {
		return
	}
	r.buf.WriteByte('{')
}

func (r *redactingListener) ExitObject(ctx objectContext) {
	r.depth--
	if r.matchingState != matchingStateVisit {
		return
	}
	r.buf.WriteByte('{')
}

func (r Redactor) redact(json string, automata node, buf *lazyBuffer, offset int) {
	jsonWalk(json, &redactingListener{automata: automata, buf: buf,
		statesBuf: make([]*state, 0, 16)})
}

func (r Redactor) redactOld(json string, automata node, buf *lazyBuffer, offset int) {
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
			_, _ = buf.WriteString(r.handler(value.Raw))
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
