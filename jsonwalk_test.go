package jsonredact

import (
	"bytes"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
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
	//fmt.Println("ExitMemberValue")
	p.b.WriteString(ctx.key)
	p.b.WriteByte(':')
}

func (p *printingListener) EnterObject(ctx objectContext) {
	//fmt.Println("EnterObject")
	p.b.WriteByte('{')
}

func (p *printingListener) ExitObject(ctx objectContext) {
	//fmt.Println("ExitObject")
	p.b.WriteByte('}')
}

func Test_jsonWalk(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			input: `{"a":{},"b":{},"c":{},"x":{"terminal":{}}}`,
		},
	}
	buffer := bytes.Buffer{}
	//l := debugListener{l: &printingListener{b: buffer}}
	l := &printingListener{b: &buffer}
	for _, tt := range tests {
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
	}
}

type debugListener struct {
	l listener
}

func (d debugListener) ExitMemberValue(ctx memberContext) {
	printCurrentFunctionName()
	d.l.ExitMemberValue(ctx)
}

func (d debugListener) EnterMembersComma() {
	printCurrentFunctionName()
	d.l.EnterMembersComma()
}

func (d debugListener) EnterMemberValue(ctx memberContext) {
	printCurrentFunctionName()
	d.l.EnterMemberValue(ctx)
}

func (d debugListener) ExitMemberKey(ctx memberContext) {
	printCurrentFunctionName()
	d.l.ExitMemberKey(ctx)
}

func (d debugListener) EnterObject(ctx objectContext) {
	printCurrentFunctionName()
	d.l.EnterObject(ctx)
}

func (d debugListener) ExitObject(ctx objectContext) {
	printCurrentFunctionName()
	d.l.ExitObject(ctx)
}

func printCurrentFunctionName() {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		fmt.Println("Unable to retrieve caller")
		return
	}
	fn := runtime.FuncForPC(pc)
	if fn != nil {
		fullName := fn.Name()
		// Extract the part after the last "/"
		shortName := fullName[strings.LastIndex(fullName, "/")+1:]
		// Remove the package name, keep only the function
		shortName = shortName[strings.Index(shortName, ".")+1:]
		fmt.Println(">", shortName)
	}
}
