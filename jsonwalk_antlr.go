package jsonredact

import (
	"container/list"
	"fmt"
	"github.com/antlr4-go/antlr/v4"
	"strings"
)

func redactWithParser(json string, automata node, buf *lazyBuffer) {
	lexer := NewJSONLexer(antlr.NewInputStream(json))
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := NewJSONParser(stream)
	tree := parser.Json()
	states := list.New()
	states.PushBack(traversingState{
		skip: false,
		node: automata,
	})
	antlr.ParseTreeWalkerDefault.Walk(&antlrListener{
		states:    states,
		buf:       buf,
		statesBuf: make([]*state, 0, 16),
		handler: func(s string) string {
			return "REDACTED"
		},
	}, tree)
}

type antlrListener struct {
	statesBuf []*state
	*BaseJSONListener
	buf       *lazyBuffer
	states    *list.List
	handler   func(string) string
	skipComma bool
}

type traversingState struct {
	skip bool
	node node
}

func (s *antlrListener) VisitTerminal(node antlr.TerminalNode) {}

// VisitErrorNode is called when an error node is visited.
func (s *antlrListener) VisitErrorNode(node antlr.ErrorNode) {}

// EnterEveryRule is called when any rule is entered.
func (s *antlrListener) EnterEveryRule(ctx antlr.ParserRuleContext) {}

// ExitEveryRule is called when any rule is exited.
func (s *antlrListener) ExitEveryRule(ctx antlr.ParserRuleContext) {}

// EnterJson is called when production json is entered.
func (s *antlrListener) EnterJson(ctx *JsonContext) {}

// ExitJson is called when production json is exited.
func (s *antlrListener) ExitJson(ctx *JsonContext) {}

// EnterObj is called when production obj is entered.
func (s *antlrListener) EnterObj(ctx *ObjContext) {
	s.skipComma = true
	parentState := s.states.Back().Value.(traversingState)
	if parentState.skip {
		return
	}
	_ = s.buf.WriteByte('{')
}

// ExitObj is called when production obj is exited.
func (s *antlrListener) ExitObj(ctx *ObjContext) {
	parentState := s.states.Back().Value.(traversingState)
	if parentState.skip {
		return
	}
	_ = s.buf.WriteByte('}')
}

func (s *antlrListener) EnterPair(ctx *PairContext) {
	keyText := ctx.STRING().GetText()
	key := keyText[1 : len(keyText)-1] //remove quotes ""
	value := ctx.Value().GetText()
	valueIndex := ctx.Value().GetStart().GetStart()
	parentState := s.states.Back().Value.(traversingState)
	if parentState.skip {
		s.states.PushBack(parentState)
		return
	}
	currentState := traversingState{
		node: parentState.node.next(key, s.statesBuf),
	}

	if currentState.node.isTerminal {
		s.buf.init()
		_, _ = s.buf.WriteString(s.buf.originalJson[:valueIndex-len(keyText)-1])
		value = `"` + s.handler(value) + `"`
		currentState.skip = true
	} else if len(currentState.node.states) == 0 {
		currentState.skip = true
	}

	_ = s.buf.WriteByte(',')
	_, _ = s.buf.WriteString(keyText)
	_ = s.buf.WriteByte(':')
	_, _ = s.buf.WriteString(value)

	s.states.PushBack(currentState)

	//fmt.Println(
	//	"key=", key,
	//	"value=", value,
	//	"index=", valueIndex,
	//	"path=", listToString(s.path),
	//	"buf.buf=", s.buf.buf.String(),
	//)
}

func (s *antlrListener) ExitPair(ctx *PairContext) {
	s.states.Remove(s.states.Back())

	parentState := s.states.Back().Value.(traversingState)
	if parentState.skip {
		return
	}
}

// EnterArr is called when production arr is entered.
func (s *antlrListener) EnterArr(ctx *ArrContext) {}

// ExitArr is called when production arr is exited.
func (s *antlrListener) ExitArr(ctx *ArrContext) {}

// EnterValue is called when production value is entered.
func (s *antlrListener) EnterValue(ctx *ValueContext) {}

// ExitValue is called when production value is exited.
func (s *antlrListener) ExitValue(ctx *ValueContext) {}

func listToString(l *list.List) string {
	var sb strings.Builder
	for e := l.Front(); e != nil; e = e.Next() {
		sb.WriteString(fmt.Sprintf("%v", e.Value))
		if e.Next() != nil {
			sb.WriteString(".")
		}
	}
	return sb.String()
}
