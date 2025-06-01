package jsonredact

import (
	"github.com/antlr4-go/antlr/v4"
	"os"
	"testing"
)

type printingListener struct {
}

func (p printingListener) EnterMember(ctx memberContext) {
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

func Benchmark_walkers(b *testing.B) {
	bytes, err := os.ReadFile("canada.json")
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	b.Run("my", func(b *testing.B) {
		jsonWalk(string(bytes), printingListener{})
	})
	b.Run("antlr", func(b *testing.B) {
		lexer := NewJSONLexer(antlr.NewInputStream(string(bytes)))
		stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
		parser := NewJSONParser(stream)
		tree := parser.Json()
		antlr.ParseTreeWalkerDefault.Walk(&BaseJSONListener{}, tree)
	})
}
