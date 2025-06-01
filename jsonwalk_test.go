package jsonredact

import (
	"fmt"
	"testing"
)

type printingListener struct {
}

func (p printingListener) EnterMember(ctx memberContext) {
	fmt.Println("EnterMember", ctx.key)
}

func (p printingListener) ExitMember(ctx memberContext) {
	//TODO implement me
	panic("implement me")
}

func (p printingListener) EnterObject(ctx objectContext) {
	//fmt.Println("EnterObject")
}

func (p printingListener) ExitObject(ctx objectContext) {
	//fmt.Println("ExitObject")
}

func Test(t *testing.T) {
	jsonWalk(`{"a":{"b":{"c":{"d":{}}}}}`, printingListener{})
}
