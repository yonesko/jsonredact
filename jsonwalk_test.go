package jsonredact

import (
	"fmt"
	"testing"
)

type printingListener struct {
	noopListener
}

func (p printingListener) EnterMemberKey(ctx memberContext) {
	fmt.Println("EnterMemberKey", ctx.key)
}

func (p printingListener) ExitMemberValue(ctx memberContext) {
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
