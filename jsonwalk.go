package jsonredact

import (
	"fmt"
)

type memberContext struct {
	key   string
	value string
}

type objectContext struct {
}

type listener interface {
	EnterMember(ctx memberContext)
	ExitMember(ctx memberContext)

	EnterObject(ctx objectContext)
	ExitObject(ctx objectContext)
}

const (
	stateElement int = iota
	stateObject  int = iota
	stateKey     int = iota
	statePair    int = iota
)

type traverseCtx struct {
	runeIndex     int
	input         []rune
	err           error
	lastMemberKey string
}

func (ctx traverseCtx) assertNextIs(char int32) bool {
	if len(ctx.input[ctx.runeIndex:]) == 0 {
		ctx.err = fmt.Errorf("unexpected end of input at %d", ctx.runeIndex)
		return false
	}
	if ctx.input[ctx.runeIndex] != char {
		ctx.err = fmt.Errorf("unexpected character %q at %d", ctx.input[ctx.runeIndex], ctx.runeIndex)
		return false
	}

	return true
}
func (ctx traverseCtx) checkNextIs(char int32) bool {
	if len(ctx.input[ctx.runeIndex:]) == 0 {
		return false
	}
	if ctx.input[ctx.runeIndex] != char {
		return false
	}

	return true
}

func jsonWalk(input string, l listener) {
	elementWalk(&traverseCtx{input: []rune(input)}, l)
}

func elementWalk(ctx *traverseCtx, l listener) {
	wsWalk(ctx, l)
	valueWalk(ctx, l)
	wsWalk(ctx, l)
}

func objectWalk(ctx *traverseCtx, l listener) {
	l.EnterObject(objectContext{})
	ctx.runeIndex += 1 //skip {
	wsWalk(ctx, l)
	switch ctx.input[ctx.runeIndex] {
	case '}':
		//empty object
		ctx.runeIndex += 1 //skip }
	case '"':
		//non-empty object
		membersWalk(ctx, l)
		if !ctx.assertNextIs('}') {
			return
		}
	}
	l.ExitObject(objectContext{})
}

func membersWalk(ctx *traverseCtx, l listener) {
	memberWalk(ctx, l)
	if ctx.checkNextIs(',') {
		ctx.runeIndex += 1 //skip ,
		membersWalk(ctx, l)
	}
}

func memberWalk(ctx *traverseCtx, l listener) {
	stringWalk(ctx, l)
	l.EnterMember(memberContext{key: ctx.lastMemberKey})
	wsWalk(ctx, l)
	if !ctx.assertNextIs(':') {
		return
	}
	ctx.runeIndex += 1 //skip :
	elementWalk(ctx, l)
}

func stringWalk(ctx *traverseCtx, l listener) {
	if !ctx.assertNextIs('"') {
		return
	}
	ctx.runeIndex += 1 //skip "
	before := ctx.runeIndex
	charactersWalk(ctx, l)
	if !ctx.assertNextIs('"') {
		return
	}
	ctx.lastMemberKey = string(ctx.input[before:ctx.runeIndex])
	ctx.runeIndex += 1 //skip "
}

// TODO empty key test case
func charactersWalk(ctx *traverseCtx, l listener) {
	if len(ctx.input[ctx.runeIndex:]) == 0 {
		return
	}
	if characterWalk(ctx, l) {
		ctx.runeIndex += 1
		charactersWalk(ctx, l)
	}
}

// TODO escape
func characterWalk(ctx *traverseCtx, l listener) bool {
	r := ctx.input[ctx.runeIndex]
	switch {
	case r == '"':
		return false
	case r == '\\':
		return false
	case r >= 0x20 && r <= 0x10FFF:
		return true
	}
	return false
}

func valueWalk(ctx *traverseCtx, l listener) {
	if len(ctx.input[ctx.runeIndex:]) == 0 {
		ctx.err = fmt.Errorf("unexpected end of input at %d", ctx.runeIndex)
		return
	}
	switch ctx.input[ctx.runeIndex] {
	case '{':
		objectWalk(ctx, l)
	default:
		ctx.err = fmt.Errorf("invalid json input, expected { at %d", ctx.runeIndex)
		return
	}
}

func wsWalk(ctx *traverseCtx, l listener) {
	input := ctx.input[ctx.runeIndex:]
	for i := range input {
		if !ws(input[i]) {
			ctx.runeIndex += i
			return
		}
	}
}

func ws(r rune) bool {
	return r == '\u0020' || r == '\u000A' || r == '\u000D' || r == '\u0009'
}
