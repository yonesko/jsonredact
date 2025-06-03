package jsonredact

import (
	"fmt"
)

const (
	valueTypeUndefined int = iota
	valueTypeObject    int = iota
)

type memberContext struct {
	//raw
	key string
	//raw
	value     string
	valueType int
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

func (n noopListener) EnterMembersComma() {
}

func (n noopListener) ExitMemberKey(ctx memberContext) {
}

func (n noopListener) EnterObject(ctx objectContext) {
}

func (n noopListener) ExitObject(ctx objectContext) {
}

type listener interface {
	//EnterMemberKey(ctx memberContext)
	ExitMemberValue(ctx memberContext)
	EnterMembersComma()

	EnterMemberValue(ctx memberContext)
	ExitMemberKey(ctx memberContext)

	EnterObject(ctx objectContext)
	ExitObject(ctx objectContext)
}

type traverseCtx struct {
	runeIndex     int
	input         []rune
	err           error
	lastMemberKey string
	l             listener
}

func (ctx *traverseCtx) assertNotEmpty() bool {
	if len(ctx.input[ctx.runeIndex:]) == 0 {
		ctx.err = fmt.Errorf("unexpected end of input at %d", ctx.runeIndex)
		return false
	}
	return true
}

func (ctx *traverseCtx) assertNextIs(char int32) bool {
	if !ctx.assertNotEmpty() {
		return false
	}
	if ctx.input[ctx.runeIndex] != char {
		ctx.err = fmt.Errorf("unexpected character %q at %d", ctx.input[ctx.runeIndex], ctx.runeIndex)
		return false
	}

	return true
}

func (ctx *traverseCtx) checkNextIs(char int32) bool {
	if len(ctx.input[ctx.runeIndex:]) == 0 {
		return false
	}
	if ctx.input[ctx.runeIndex] != char {
		return false
	}

	return true
}

func jsonWalk(input string, l listener) error {
	ctx := &traverseCtx{input: []rune(input), l: l}
	ctx.elementWalk()
	return ctx.err
}

func (ctx *traverseCtx) elementWalk() {
	ctx.wsWalk()
	ctx.valueWalk()
	ctx.wsWalk()
}

func (ctx *traverseCtx) objectWalk() {
	if !ctx.assertNextIs('{') {
		return
	}
	ctx.l.EnterObject(objectContext{})
	ctx.runeIndex += 1 //skip {
	ctx.wsWalk()
	switch ctx.input[ctx.runeIndex] {
	case '}':
		//empty object
		ctx.runeIndex += 1 //skip }
	case '"':
		//non-empty object
		ctx.membersWalk()
		if !ctx.assertNextIs('}') {
			return
		}
	}
	ctx.l.ExitObject(objectContext{})
}

func (ctx *traverseCtx) membersWalk() {
	ctx.memberWalk()
	if ctx.checkNextIs(',') {
		ctx.l.EnterMembersComma()
		ctx.runeIndex += 1 //skip ,
		ctx.membersWalk()
	}
}

func (ctx *traverseCtx) memberWalk() {
	ctx.wsWalk()
	ctx.stringWalk()
	ctx.wsWalk()
	if !ctx.assertNextIs(':') {
		return
	}
	key := ctx.lastMemberKey
	ctx.l.ExitMemberKey(memberContext{key: key})
	ctx.runeIndex += 1 //skip :
	before := ctx.runeIndex
	ctx.elementWalk()
	ctx.l.ExitMemberValue(memberContext{key: ctx.lastMemberKey,
		value: string(ctx.input[before:ctx.runeIndex])})
}

func (ctx *traverseCtx) stringWalk() {
	if !ctx.assertNextIs('"') {
		return
	}
	before := ctx.runeIndex
	ctx.runeIndex += 1 //skip "
	ctx.charactersWalk()
	if !ctx.assertNextIs('"') {
		return
	}
	ctx.runeIndex += 1 //skip "
	ctx.lastMemberKey = string(ctx.input[before:ctx.runeIndex])
}

// TODO empty key test case
func (ctx *traverseCtx) charactersWalk() {
	if len(ctx.input[ctx.runeIndex:]) == 0 {
		return
	}
	if ctx.characterWalk() {
		ctx.runeIndex += 1
		ctx.charactersWalk()
	}
}

// TODO escape
func (ctx *traverseCtx) characterWalk() bool {
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

func (ctx *traverseCtx) valueWalk() {
	if !ctx.assertNotEmpty() {
		return
	}
	switch ctx.input[ctx.runeIndex] {
	case '{':
		ctx.objectWalk()
	case '[':
		ctx.arrayWalk()
	case '"':
		ctx.stringWalk()
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		ctx.numberWalk()
	default:
		ctx.err = fmt.Errorf("invalid json input, expected value at %d", ctx.runeIndex)
		return
	}
}

func (ctx *traverseCtx) numberWalk() {
	ctx.integerWalk()
	ctx.fractionWalk()
	ctx.exponentWalk()
}

func (ctx *traverseCtx) exponentWalk() {
	if !ctx.checkNextIs('e') && !ctx.checkNextIs('E') {
		return
	}
	ctx.runeIndex += 1 //skip e
	if ctx.checkNextIs('+') || !ctx.checkNextIs('-') {
		ctx.runeIndex += 1 //skip -
	}
	ctx.digitsWalk()
}

func (ctx *traverseCtx) integerWalk() {
	if ctx.checkNextIs('-') {
		ctx.runeIndex += 1 //skip -
	}
	ctx.digitsWalk()
}

func (ctx *traverseCtx) digitsWalk() {
	for _, r := range ctx.input[ctx.runeIndex:] {
		if r >= '0' && r <= '9' {
			ctx.runeIndex += 1
		} else {
			break
		}
	}
}

func (ctx *traverseCtx) fractionWalk() {
	if !ctx.checkNextIs('.') {
		return
	}
	ctx.runeIndex += 1 //skip .
	ctx.digitsWalk()
}

func (ctx *traverseCtx) arrayWalk() {
	if !ctx.assertNextIs('[') {
		return
	}
	ctx.runeIndex += 1 //skip [
	ctx.wsWalk()
	switch ctx.input[ctx.runeIndex] {
	case ']':
		//empty array
		ctx.runeIndex += 1 //skip ]
	case '"':
		//non-empty array
		ctx.elementsWalk()
		if !ctx.assertNextIs(']') {
			return
		}
	}
}

func (ctx *traverseCtx) elementsWalk() {
	ctx.elementWalk()
	if ctx.checkNextIs(',') {
		ctx.runeIndex += 1 //skip ,
		ctx.elementsWalk()
	}
}

func (ctx *traverseCtx) wsWalk() {
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
