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
age
name.first
a.*.name
children.#.name
children.#
children.1
children.1.name
*/
func NewRedactor(keySelectors []string, handler func(string) string) Redactor {
	return Redactor{handler: handler, selectorForest: parseSelector(keySelectors)}
}

func Redact(json string, keySelectors []string, handler func(string) string) string {
	return Redactor{selectorForest: parseSelector(keySelectors), handler: handler}.Redact(json)
}

func (r Redactor) Redact(json string) string {
	if len(r.selectorForest) == 0 {
		return json
	}
	return (&state{json: json, selectorForest: r.selectorForest, handler: r.handler}).redact()
}

type selectorForest map[string]selectorForest

// TODO dot and other control symbols escape
func (f selectorForest) add(str string) {
	var fi = f
	for _, val := range strings.Split(str, ".") {
		_, ok := fi[val]
		if !ok {
			fi[val] = map[string]selectorForest{}
		}
		fi = fi[val]
	}
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
}

func (s *state) selectForest(key gjson.Result) selectorForest {
	f, ok := s.selectorForest[key.String()]
	if ok {
		return f
	}
	f, ok = s.selectorForest["#"]
	return f
}

func (s *state) redact() string {
	buffer := bytes.NewBuffer(make([]byte, 0, len(s.json)))
	root := gjson.Parse(s.json)
	if root.IsArray() {
		_ = buffer.WriteByte('[')
	} else {
		_ = buffer.WriteByte('{')
	}
	root.ForEach(func(key, value gjson.Result) bool {
		if s.index != 0 {
			_ = buffer.WriteByte(',')
		}
		s.index++
		_, _ = buffer.WriteString(key.Raw)
		if !root.IsArray() {
			_ = buffer.WriteByte(':')
		}
		if forest := s.selectForest(key); forest == nil {
			_, _ = buffer.WriteString(value.Raw)
		} else if len(forest) != 0 {
			if value.IsObject() || value.IsArray() {
				redactedValue := (&state{json: value.Raw, selectorForest: forest, handler: s.handler}).redact()
				_, _ = buffer.WriteString(redactedValue)
			} else {
				_, _ = buffer.WriteString(value.Raw)
			}
		} else {
			_, _ = buffer.WriteString(s.handler(value.Raw))
		}
		return true
	})
	if root.IsArray() {
		_ = buffer.WriteByte(']')
	} else {
		_ = buffer.WriteByte('}')
	}
	return buffer.String()
}
