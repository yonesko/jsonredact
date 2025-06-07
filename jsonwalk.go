package jsonredact

import (
	"fmt"
	"runtime"
	"strings"
)

const (
	valueTypeUndefined int = iota
	valueTypeObject    int = iota
	valueTypeArray     int = iota
	valueTypeString    int = iota
	valueTypeNumber    int = iota
)

type memberContext struct {
	//raw
	key string
	//raw
	value     string
	valueType int
	runeIndex int
}

type objectContext struct {
	//raw
	value string
}

type valueContext struct {
	//raw
	value     string
	valueType int
}

var (
	_ listener = noopListener{}
)

type noopListener struct {
}

func (n noopListener) ExitValue(ctx valueContext) {
}

func (n noopListener) EnterValue(ctx valueContext) {
}

func (n noopListener) EnterArray() {
}

func (n noopListener) ExitArray() {
}

func (n noopListener) EnterMemberKey(ctx memberContext) {
}

func (n noopListener) ExitMemberValue(ctx memberContext) {
}

func (n noopListener) EnterMemberValue(ctx memberContext) {
}

func (n noopListener) EnterComma() {
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
	EnterComma()

	EnterMemberValue(ctx memberContext)
	ExitMemberKey(ctx memberContext)

	EnterObject(ctx objectContext)
	ExitObject(ctx objectContext)

	EnterValue(ctx valueContext)
	ExitValue(ctx valueContext)

	EnterArray()
	ExitArray()
}

type traverseCtx struct {
	runeIndex     int
	input         []rune
	err           error
	lastMemberKey string
	l             listener
}

// TODO don't walk on error
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

func (ctx *traverseCtx) elementWalk() int {
	ctx.wsWalk()
	before := ctx.runeIndex
	vt := ctx.valueWalk()
	ctx.l.ExitValue(valueContext{
		value:     string(ctx.input[before:ctx.runeIndex]),
		valueType: vt,
	})
	ctx.wsWalk()
	return vt
}

func (ctx *traverseCtx) objectWalk() {
	if !ctx.assertNextIs('{') {
		return
	}
	before := ctx.runeIndex
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
		ctx.runeIndex += 1 //skip }
	}
	ctx.l.ExitObject(objectContext{value: string(ctx.input[before:ctx.runeIndex])})
}

func (ctx *traverseCtx) membersWalk() {
	ctx.memberWalk()
	for ctx.checkNextIs(',') {
		ctx.l.EnterComma()
		ctx.runeIndex += 1 //skip ,
		ctx.memberWalk()
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
	ctx.l.ExitMemberKey(memberContext{key: key, runeIndex: ctx.runeIndex})
	ctx.runeIndex += 1 //skip :
	ctx.wsWalk()
	before := ctx.runeIndex
	vt := ctx.elementWalk()
	ctx.l.ExitMemberValue(memberContext{
		key:       ctx.lastMemberKey,
		value:     string(ctx.input[before:ctx.runeIndex]),
		valueType: vt,
	})
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
	for ctx.characterWalk() {
		ctx.runeIndex += 1
		ctx.characterWalk()
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

func (ctx *traverseCtx) valueWalk() int {
	if !ctx.assertNotEmpty() {
		return 0
	}
	switch ctx.input[ctx.runeIndex] {
	case '{':
		ctx.l.EnterValue(valueContext{valueType: valueTypeObject})
		ctx.objectWalk()
		return valueTypeObject
	case '[':
		ctx.l.EnterValue(valueContext{valueType: valueTypeArray})
		ctx.arrayWalk()
		return valueTypeArray
	case '"':
		ctx.l.EnterValue(valueContext{valueType: valueTypeString})
		ctx.stringWalk()
		return valueTypeString
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		ctx.l.EnterValue(valueContext{valueType: valueTypeNumber})
		ctx.numberWalk()
		return valueTypeNumber
	default:
		ctx.err = fmt.Errorf("invalid json input, expected value at %d", ctx.runeIndex)
		return 0
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
	ctx.l.EnterArray()
	ctx.runeIndex += 1 //skip [
	ctx.wsWalk()
	switch ctx.input[ctx.runeIndex] {
	case ']':
		//empty array
		ctx.runeIndex += 1 //skip ]
	default:
		//non-empty array
		ctx.elementsWalk()
		if !ctx.assertNextIs(']') {
			return
		}
	}
	ctx.l.ExitArray()
}

func (ctx *traverseCtx) elementsWalk() {
	ctx.elementWalk()
	for ctx.checkNextIs(',') {
		ctx.l.EnterComma()
		ctx.runeIndex += 1 //skip ,
		ctx.elementWalk()
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

type debugListener struct {
	l listener
}

func (d debugListener) EnterValue(ctx valueContext) {
	fmt.Printf("%s(%+v)\n", printCurrentFunctionName(), ctx)
	d.l.EnterValue(ctx)
}

func (d debugListener) EnterArray() {
	fmt.Printf("%s(%+v)\n", printCurrentFunctionName(), nil)
	d.l.EnterArray()
}

func (d debugListener) ExitArray() {
	fmt.Printf("%s(%+v)\n", printCurrentFunctionName(), nil)
	d.l.ExitArray()
}

func (d debugListener) ExitMemberValue(ctx memberContext) {
	fmt.Printf("%s(%+v)\n", printCurrentFunctionName(), ctx)
	d.l.ExitMemberValue(ctx)
}

func (d debugListener) EnterComma() {
	fmt.Printf("%s(%+v)\n", printCurrentFunctionName(), nil)
	d.l.EnterComma()
}

func (d debugListener) EnterMemberValue(ctx memberContext) {
	fmt.Printf("%s(%+v)\n", printCurrentFunctionName(), ctx)
	d.l.EnterMemberValue(ctx)
}

func (d debugListener) ExitMemberKey(ctx memberContext) {
	fmt.Printf("%s(%+v)\n", printCurrentFunctionName(), ctx)
	d.l.ExitMemberKey(ctx)
}

func (d debugListener) EnterObject(ctx objectContext) {
	fmt.Printf("%s(%+v)\n", printCurrentFunctionName(), ctx)
	d.l.EnterObject(ctx)
}

func (d debugListener) ExitObject(ctx objectContext) {
	fmt.Printf("%s(%+v)\n", printCurrentFunctionName(), ctx)
	d.l.ExitObject(ctx)
}

func (d debugListener) ExitValue(ctx valueContext) {
	fmt.Printf("%s(%+v)\n", printCurrentFunctionName(), ctx)
	d.l.ExitValue(ctx)
}

func printCurrentFunctionName() string {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		fmt.Println("Unable to retrieve caller")
		return ""
	}
	fn := runtime.FuncForPC(pc)
	if fn != nil {
		fullName := fn.Name()
		// Extract the part after the last "/"
		shortName := fullName[strings.LastIndex(fullName, "/")+1:]
		// Remove the package name, keep only the function
		shortName = shortName[strings.Index(shortName, ".")+1:]
		return shortName
	}
	return ""
}
