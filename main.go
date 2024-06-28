package jsonredact

import (
	"bytes"
	"github.com/tidwall/gjson"
	"regexp"
	"strconv"
	"strings"
)

type Redactor struct {
	re      *regexp.Regexp
	handler func(string) string
}

/*
User '.' as separator of objects and arrays.
Use '#' as wildcard for any key or array index.
Use '*' to apply right expression to all object keys recursively.
User '\' to escape control symbols above.
*/
func NewRedactor(keySelectors []string, handler func(string) string) Redactor {
	return Redactor{handler: handler, re: buildRe(keySelectors)}
}

func buildRe(keySelectors []string) *regexp.Regexp {
	if len(keySelectors) == 0 {
		return nil
	}
	exprs := make([]string, 0, len(keySelectors))
	for _, s := range keySelectors {
		s = regexp.QuoteMeta(s)
		s = strings.Replace(s, `\*\.`, `.*`, -1)
		if !strings.Contains(s, `\*\.`) {
			s = "^" + s
		}
		exprs = append(exprs, s)
	}

	compile, err := regexp.Compile(strings.Join(exprs, "|"))
	if err != nil {
		panic(err)
	}
	return compile
}

func Redact(json string, keySelectors []string, handler func(string) string) string {
	return NewRedactor(keySelectors, handler).Redact(json)
}

func (r Redactor) Redact(json string) string {
	if r.re == nil {
		return json
	}
	buffer := bytes.NewBuffer(make([]byte, 0, len(json)))
	r.redact(json, buffer, make([]byte, 0, 256))
	return buffer.String()
}

func (r Redactor) redact(json string, buf *bytes.Buffer, path []byte) {
	root := gjson.Parse(json)
	if root.IsArray() {
		_ = buf.WriteByte('[')
	} else {
		_ = buf.WriteByte('{')
	}
	var index int
	pathLenBefore := len(path)
	root.ForEach(func(key, value gjson.Result) bool {
		keyStr := key.Str
		if root.IsArray() {
			keyStr = strconv.Itoa(index)
		}
		if index != 0 {
			_ = buf.WriteByte(',')
		}
		if len(path) != 0 {
			path = append(path, '.')
		}
		path = append(path, keyStr...)
		index++
		_, _ = buf.WriteString(key.Raw)
		if !root.IsArray() {
			_ = buf.WriteByte(':')
		}

		//fmt.Println(string(path))
		if r.re.Match(path) {
			_ = buf.WriteByte('"')
			_, _ = buf.WriteString(r.handler(value.Raw))
			_ = buf.WriteByte('"')
			path = path[:pathLenBefore]
			return true
		}
		if !value.IsObject() && !value.IsArray() {
			_, _ = buf.WriteString(value.Raw)
			path = path[:pathLenBefore]
			return true
		}
		r.redact(value.Raw, buf, path)
		path = path[:pathLenBefore]
		return true
	})
	if root.IsArray() {
		_ = buf.WriteByte(']')
	} else {
		_ = buf.WriteByte('}')
	}
}
