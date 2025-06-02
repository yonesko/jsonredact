package jsonredact

import (
	"fmt"
)

type memberContext struct {
	//raw
	key string
	//raw
	value string
	//member's index in object
	index int
}

type objectContext struct {
}

type noopListener struct {
}

func (n noopListener) EnterMemberKey(ctx memberContext) {
}

func (n noopListener) ExitMemberValue(ctx memberContext) {
}

func (n noopListener) EnterMemberValue(ctx memberContext) {
}

func (n noopListener) ExitMemberKey(ctx memberContext) {
}

func (n noopListener) EnterObject(ctx objectContext) {
}

func (n noopListener) ExitObject(ctx objectContext) {
}

type listener interface {
	EnterMemberKey(ctx memberContext)
	ExitMemberValue(ctx memberContext)

	EnterMemberValue(ctx memberContext)
	ExitMemberKey(ctx memberContext)

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

func (ctx traverseCtx) assertNotEmpty() bool {
	if len(ctx.input[ctx.runeIndex:]) == 0 {
		ctx.err = fmt.Errorf("unexpected end of input at %d", ctx.runeIndex)
		return false
	}
	return true
}

func (ctx traverseCtx) assertNextIs(char int32) bool {
	if !ctx.assertNotEmpty() {
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
	if !ctx.assertNextIs('{') {
		return
	}
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
	l.EnterMemberKey(memberContext{key: ctx.lastMemberKey})
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
	if !ctx.assertNotEmpty() {
		return
	}
	switch ctx.input[ctx.runeIndex] {
	case '{':
		objectWalk(ctx, l)
	case '[':
		arrayWalk(ctx, l)
	case '"':
		stringWalk(ctx, l)
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		numberWalk(ctx, l)
	default:
		ctx.err = fmt.Errorf("invalid json input, expected value at %d", ctx.runeIndex)
		return
	}
}

func numberWalk(ctx *traverseCtx, l listener) {
	integerWalk(ctx, l)
	fractionWalk(ctx, l)
	exponentWalk(ctx, l)
}

func exponentWalk(ctx *traverseCtx, l listener) {
	if !ctx.checkNextIs('e') && !ctx.checkNextIs('E') {
		return
	}
	ctx.runeIndex += 1 //skip e
	if ctx.checkNextIs('+') || !ctx.checkNextIs('-') {
		ctx.runeIndex += 1 //skip -
	}
	digitsWalk(ctx, l)
}

func integerWalk(ctx *traverseCtx, l listener) {
	if ctx.checkNextIs('-') {
		ctx.runeIndex += 1 //skip -
	}
	digitsWalk(ctx, l)
}

func digitsWalk(ctx *traverseCtx, l listener) {
	for _, r := range ctx.input[ctx.runeIndex:] {
		if r >= '0' && r <= '9' {
			ctx.runeIndex += 1
		} else {
			break
		}
	}
}

func fractionWalk(ctx *traverseCtx, l listener) {
	if !ctx.checkNextIs('.') {
		return
	}
	ctx.runeIndex += 1 //skip .
	digitsWalk(ctx, l)
}

func arrayWalk(ctx *traverseCtx, l listener) {
	if !ctx.assertNextIs('[') {
		return
	}
	ctx.runeIndex += 1 //skip [
	wsWalk(ctx, l)
	switch ctx.input[ctx.runeIndex] {
	case ']':
		//empty array
		ctx.runeIndex += 1 //skip ]
	case '"':
		//non-empty array
		elementsWalk(ctx, l)
		if !ctx.assertNextIs(']') {
			return
		}
	}
}

func elementsWalk(ctx *traverseCtx, l listener) {
	elementWalk(ctx, l)
	if ctx.checkNextIs(',') {
		ctx.runeIndex += 1 //skip ,
		elementsWalk(ctx, l)
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

/*
json
	element

value
	object
	array
	string
	number
	"true"
	"false"
	"null"

object
	'{' ws '}'
	'{' members '}'

members
	member
	member ',' members

member
	ws string ws ':' element

array
	'[' ws ']'
	'[' elements ']'

elements
	element
	element ',' elements

element
	ws value ws

string
	'"' characters '"'

characters
	""
	character characters

character
	'0020' . '10FFFF' - '"' - '\'
	'\' escape

escape
	'"'
	'\'
	'/'
	'b'
	'f'
	'n'
	'r'
	't'
	'u' hex hex hex hex

hex
digit
	'A' . 'F'
	'a' . 'f'

number
	integer fraction exponent

integer
	digit
	onenine digits
	'-' digit
	'-' onenine digits

digits
	digit
	digit digits

digit
	'0'
	onenine

onenine
	'1' . '9'

fraction
	""
	'.' digits

exponent
	""
	'E' sign digits
	'e' sign digits

sign
	""
	'+'
	'-'

ws
	""
	'0020' ws
	'000A' ws
	'000D' ws
	'0009' ws
*/
