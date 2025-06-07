package jsonredact

import (
	"bytes"
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

type redactingListener struct {
	noopListener
	buf       *lazyBuffer
	handler   func(string) string
	statesBuf []*state
	path      []*redactingListenerState
}

func (r *redactingListener) EnterComma() {
	st := r.path[len(r.path)-1]
	if !st.skipMatching {
		r.buf.WriteByte(',')
	}
}

func (r *redactingListener) ExitMemberKey(ctx memberContext) {
	st := r.path[len(r.path)-1]

	next := st.automata.next(ctx.key[1:len(ctx.key)-1], r.statesBuf)
	st.nextAutomata = &next

	if next.isTerminal && r.buf.buf == nil {
		r.buf.buf = bytes.NewBuffer(make([]byte, 0, len(r.buf.originalJson)))
		r.buf.WriteString(string([]rune(r.buf.originalJson)[:ctx.runeIndex+1]))
		return
	}

	if !st.skipMatching {
		r.buf.WriteString(ctx.key)
		r.buf.WriteByte(':')
	}
}

func (r *redactingListener) EnterObject(ctx objectContext) {
	st := r.path[len(r.path)-1]
	nextAutomata := *st.nextAutomata
	st.nextAutomata = nil
	nextSt := redactingListenerState{
		automata:     nextAutomata,
		skipMatching: nextAutomata.isTerminal || st.skipMatching,
	}
	r.path = append(r.path, &nextSt)

	if !nextSt.skipMatching {
		r.buf.WriteByte('{')
	}
}

func (r *redactingListener) ExitObject(ctx objectContext) {
	st := r.path[len(r.path)-1]
	r.path = r.path[:len(r.path)-1]
	if st.automata.isTerminal {
		r.buf.WriteString(`"` + r.handler(ctx.value) + `"`)
	} else if !st.skipMatching {
		r.buf.WriteByte('}')
	}
}

func (r *redactingListener) ExitMemberValue(ctx memberContext) {
	st := r.path[len(r.path)-1]
	if ctx.valueType == valueTypeArray || ctx.valueType == valueTypeObject || st.skipMatching {
		return
	}
	nextAutomata := *st.nextAutomata
	st.nextAutomata = nil
	if nextAutomata.isTerminal {
		r.buf.WriteString(`"` + r.handler(ctx.value) + `"`)
	} else {
		r.buf.WriteString(ctx.value)
	}
}

func (r *redactingListener) ExitValue(ctx valueContext) {

}

type redactingListenerState struct {
	automata     node
	nextAutomata *node
	skipMatching bool
}

func (r Redactor) redact(json string, automata node, buf *lazyBuffer, offset int) {
	path := make([]*redactingListenerState, 0, 20)
	path = append(path, &redactingListenerState{nextAutomata: &automata})
	l := &redactingListener{
		buf:       buf,
		statesBuf: make([]*state, 0, 16),
		handler:   r.handler,
		path:      path,
	}
	err := jsonWalk(json, l)
	//err := jsonWalk(json, debugListener{l: l})
	if err != nil {
		buf.buf = bytes.NewBuffer([]byte(buf.originalJson))
	}
}
