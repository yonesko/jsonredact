package jsonredact

import (
	"bytes"
	"github.com/tidwall/gjson"
	"strings"
)

type Redactor struct {
	keys    map[string]bool
	handler func(string) string
}

type state struct {
	json    string
	path    []string
	keys    map[string]bool
	handler func(string) string
	index   int
}

func Redact(json string, keys []string, handler func(string) string) string {
	keysMap := make(map[string]bool)
	for _, k := range keys {
		keysMap[k] = true
	}
	return Redactor{keys: keysMap, handler: handler}.Redact(json)
}

func (r Redactor) Redact(json string) string {
	return (&state{json: json, keys: r.keys, handler: r.handler}).redact()
}

func (s *state) redact() string {
	buffer := bytes.NewBuffer(make([]byte, 0, len(s.json)))
	parsed := gjson.Parse(s.json)
	if parsed.IsArray() {
		_ = buffer.WriteByte('[')
	} else {
		_ = buffer.WriteByte('{')
	}
	parsed.ForEach(func(key, value gjson.Result) bool {
		if s.index != 0 {
			_ = buffer.WriteByte(',')
		}
		s.index++
		path := append(s.path, strings.ReplaceAll(key.String(), `.`, `\.`))
		pathStr := strings.Join(path, ".")
		_, _ = buffer.WriteString(key.Raw)
		if !parsed.IsArray() {
			_ = buffer.WriteByte(':')
		}
		if s.keys[pathStr] {
			_, _ = buffer.WriteString(s.handler(value.Raw))
		} else if value.Type == gjson.JSON {
			_, _ = buffer.WriteString((&state{json: value.Raw, path: path, keys: s.keys, handler: s.handler}).redact())
		} else {
			_, _ = buffer.WriteString(value.Raw)
		}
		return true
	})
	if parsed.IsArray() {
		_ = buffer.WriteByte(']')
	} else {
		_ = buffer.WriteByte('}')
	}
	return buffer.String()
}
