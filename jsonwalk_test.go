package jsonredact

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
)

type printingListener struct {
	noopListener
	b *bytes.Buffer
}

func (p *printingListener) EnterMembersComma() {
	p.b.WriteByte(',')
}

func (p *printingListener) ExitMemberKey(ctx memberContext) {
	p.b.WriteString(ctx.key)
	p.b.WriteByte(':')
}

func (p *printingListener) EnterObject(ctx objectContext) {
	p.b.WriteByte('{')
}

func (p *printingListener) ExitObject(ctx objectContext) {
	p.b.WriteByte('}')
}

func (p *printingListener) ExitMemberValue(ctx memberContext) {
	fmt.Printf("'%s'\n", ctx.value)
	if ctx.valueType != valueTypeObject {
		p.b.WriteString(ctx.value)
	}
}

func Test_jsonWalk(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{input: `{"a":{},"b":{},"c":{},"x":{"terminal":{}}}`},
		{input: `{"a":{"b":12345}}`},
	}
	buffer := bytes.Buffer{}
	//l := debugListener{l: &printingListener{b: buffer}}
	l := &printingListener{b: &buffer}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			buffer.Reset()
			err := jsonWalk(tt.input, l)
			if err != nil {
				t.Fatalf("error parsing json: %v", err)
			}
			if !json.Valid([]byte(tt.input)) {
				t.Fatal("input json is invalid")
			}
			if !json.Valid([]byte(buffer.String())) {
				t.Fatal("got json is invalid:\n", buffer.String())
			}
			if buffer.String() != tt.input {
				t.Fatalf("input is not equal to expected output\ninput:\n%s\noutput:\n%s\n", tt.input, buffer.String())
			}
		})
	}
}
