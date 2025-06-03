package jsonredact

import (
	"bytes"
	"container/list"
	"log"
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
	path      *list.List
}

func (r *redactingListener) EnterMembersComma() {
	r.buf.WriteByte(',')
}

func (r *redactingListener) ExitMemberKey(ctx memberContext) {
	st := r.path.Back().Value.(*redactingListenerState)

	next := st.parentAutomata.next(ctx.key[1:len(ctx.key)-1], r.statesBuf)
	st.currentAutomata = &next

	if len(st.parentAutomata.states) != 0 && !st.parentAutomata.isTerminal {
		r.buf.WriteString(ctx.key)
		r.buf.WriteByte(':')
	}
}

func (r *redactingListener) ExitMemberValue(ctx memberContext) {
	if ctx.valueType != valueTypeObject {
		return
	}
	st := r.path.Back().Value.(*redactingListenerState)

	//fmt.Println("value", ctx.value, "st",
	//	st.parentAutomata.isTerminal, st.currentAutomata.isTerminal,
	//	len(st.parentAutomata.states), len(st.currentAutomata.states),
	//)

	//println(ctx.key)

	if st.currentAutomata.isTerminal {
		r.buf.WriteString(`"` + r.handler(ctx.value) + `"`)
	} else if len(st.parentAutomata.states) != 0 {
		r.buf.WriteString(ctx.value)
		//r.buf.WriteString("[" + ctx.value + "]")

	}

}

func (r *redactingListener) EnterObject(ctx objectContext) {
	st := r.path.Back().Value.(*redactingListenerState)
	//fmt.Println("st",
	//	st.currentAutomata.isTerminal, len(st.currentAutomata.states),
	//)
	r.path.PushBack(&redactingListenerState{parentAutomata: st.currentAutomata})
	if !st.currentAutomata.isTerminal && len(st.currentAutomata.states) != 0 {
		r.buf.WriteByte('{')
	}
}

func (r *redactingListener) ExitObject(ctx objectContext) {
	st := r.path.Back().Value.(*redactingListenerState)
	r.path.Remove(r.path.Back())
	if !st.parentAutomata.isTerminal && len(st.parentAutomata.states) != 0 {
		r.buf.WriteByte('}')
	}
}

type redactingListenerState struct {
	parentAutomata  *node
	currentAutomata *node
}

func (r Redactor) redact(json string, automata node, buf *lazyBuffer, offset int) {
	buf.buf = bytes.NewBuffer(make([]byte, 0, len(buf.originalJson)))
	path := list.New()
	path.PushBack(&redactingListenerState{currentAutomata: &automata})
	l := &redactingListener{
		buf:       buf,
		statesBuf: make([]*state, 0, 16),
		handler:   r.handler,
		path:      path,
	}
	err := jsonWalk(json, debugListener{l: l})
	if err != nil {
		log.Fatal(err)
	}

}

//func (r Redactor) redactOld(json string, parentAutomata node, buf *lazyBuffer, offset int) {
//	root := gjson.Parse(json)
//	if root.IsArray() {
//		_ = buf.WriteByte('[')
//	} else {
//		_ = buf.WriteByte('{')
//	}
//	var index int
//	statesBuf := make([]*state, 0, 16)
//	root.ForEach(func(key, value gjson.Result) bool {
//		keyStr := key.Str
//		if root.IsArray() {
//			keyStr = strconv.Itoa(index)
//		}
//		if index != 0 {
//			_ = buf.WriteByte(',')
//		}
//		index++
//		_, _ = buf.WriteString(key.Raw)
//		if !root.IsArray() {
//			_ = buf.WriteByte(':')
//		}
//		next := parentAutomata.next(keyStr, statesBuf)
//		if next.isTerminal {
//			if buf.buf == nil {
//				buf.buf = bytes.NewBuffer(make([]byte, 0, len(buf.originalJson)))
//				_, _ = buf.WriteString(buf.originalJson[:offset+value.Index])
//			}
//			_ = buf.WriteByte('"')
//			_, _ = buf.WriteString(r.handler(value.Raw))
//			_ = buf.WriteByte('"')
//			return true
//		}
//		if len(next.states) == 0 || (!value.IsObject() && !value.IsArray()) {
//			_, _ = buf.WriteString(value.Raw)
//			return true
//		}
//		r.redact(value.Raw, next, buf, offset+value.Index)
//		return true
//	})
//	if root.IsArray() {
//		_ = buf.WriteByte(']')
//	} else {
//		_ = buf.WriteByte('}')
//	}
//}
