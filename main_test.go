package jsonredact

import (
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

var (
	bigJson = `{
  "age": 37,
  "children": [ "Sara", "Alex", "Jack" ],
  "fav.movie": { "title": "Deer Hunter" },
  "fav": { "movie": "BIG" },
  "friends": [
    { "age": 44, "first": "Dale", "last": "Murphy" },
    { "age": 68, "first": "Roger", "last": "Craig" },
    { "age": 47, "first": "Jane", "last": "Murphy" }
  ],
  "name": { "first": "Tom", "last": "Anderson" }
}`
)

func TestRedact(t *testing.T) {
	type args struct {
		json string
		keys []string
	}
	handler := func(s string) string {
		return `REDACTED`
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "base/no selectors - return as is",
			args: args{json: bigJson},
			want: bigJson,
		},
		{
			name: "base/plain path of 0 depth",
			args: args{json: `{"a":1,"b":1,"c":1}`, keys: []string{"a", "b"}},
			want: `{"a":"REDACTED","b":"REDACTED","c":1}`,
		},
		{
			name: "base/no match",
			args: args{json: bigJson, keys: []string{"1age", "1fav.movie", "1friends", "1name.last"}},
			want: bigJson,
		},
		{
			name: "base/plain path of 3 depth",
			args: args{json: `{"a":{"b":{"c":1}},"b":1,"c":1}`, keys: []string{"a.b.c", "c"}},
			want: `{"a":{"b":{"c":"REDACTED"}},"b":1,"c":"REDACTED"}`,
		},
		{
			name: "base/two paths with common prefix",
			args: args{json: `{"a":{"b":{"c":1, "d":1}},"b":1,"c":1}`, keys: []string{"a.b.c", "a.b.d"}},
			want: `{"a":{"b":{"c":"REDACTED", "d":"REDACTED"}},"b":1,"c":1}`,
		},
		{
			name: "base/two paths with common prefix and different depth",
			args: args{json: `{"a":{"b":{"c":1, "d":{"f":1} }},"b":1,"c":1}`, keys: []string{"a.b.c", "a.b.d.e"}},
			want: `{"a":{"b":{"c":"REDACTED", "d":{"f":1} }},"b":1,"c":1}`,
		},
		{
			name: "base/do not override general by particular",
			args: args{json: `{"a":{"b":1}}`, keys: []string{"a", "a.b"}},
			want: `{"a":"REDACTED"}`,
		},
		{
			name: "base/do not override general by particular, different order",
			args: args{json: `{"a":{"b":1}}`, keys: []string{"a.b", "a"}},
			want: `{"a":"REDACTED"}`,
		},
		{
			name: "array/whole",
			args: args{json: `{"a":[1,2,{"c":1,"d":{"e":2}}],"b":2}`, keys: []string{"a"}},
			want: `{"a":"REDACTED","b":2}`,
		},
		{
			name: "array/with index",
			args: args{json: `{"a":[1,2,{"c":1,"d":{"e":2}}],"b":2}`, keys: []string{"a.1"}},
			want: `{"a":[1,"REDACTED",{"c":1,"d":{"e":2}}],"b":2}`,
		},
		{
			name: "array/with indexes",
			args: args{json: `{"a":[1,2,{"c":1,"d":{"e":2}}],"b":2}`, keys: []string{"a.1", "a.0"}},
			want: `{"a":["REDACTED","REDACTED",{"c":1,"d":{"e":2}}],"b":2}`,
		},
		{
			name: "array/with index in middle",
			args: args{json: `{"a":[1,2,{"c":1,"d":{"e":2}}],"b":2}`, keys: []string{"a.2.c", "a.2.d.e"}},
			want: `{"a":[1,2,{"c":"REDACTED","d":{"e":"REDACTED"}}],"b":2}`,
		},
		{
			name: "array/nested",
			args: args{json: `{"a":[1,[1,[1,[{"a":1}]]]],"b":2}`, keys: []string{"a.1.1.1.0.a"}},
			want: `{"a":[1,[1,[1,[{"a":"REDACTED"}]]]],"b":2}`,
		},
		{
			name: "array/certain field of certain array element",
			args: args{json: `{ "children": [ {"name":"Sara", "null":null}, "Alex", "Jack",null ] }`,
				keys: []string{`children.0.name`, `children.2.name`}},
			want: `{ "children": [ {"name":"REDACTED", "null":null}, "Alex", "Jack",null ] }`,
		},
		{
			name: "escape/real point",
			args: args{json: `{ "a.b": 1, "a": { "b": 2 } }`, keys: []string{"a.b"}},
			want: `{ "a.b": 1, "a": { "b": "REDACTED" } }`,
		},
		{
			name: "escape/escaped point",
			args: args{json: `{ "a.b": 1, "a": { "b": 2 } }`, keys: []string{`a\.b`}},
			want: `{ "a.b": "REDACTED", "a": { "b": 2 } }`,
		},
		{
			name: "escape/real and escaped point",
			args: args{json: `{ "a.b": 1, "a": { "b": 2 } }`, keys: []string{`a\.b`, `a.b`}},
			want: `{ "a.b": "REDACTED", "a": { "b": "REDACTED" } }`,
		},
		{
			name: "escape/point as name",
			args: args{json: `{ "a": { ".": 2 } }`, keys: []string{`a.\.`}},
			want: `{ "a": { ".": "REDACTED" } }`,
		},
		{
			name: "escape/quote in name",
			args: args{json: `{ "a\"": 1 }`, keys: []string{`a\"`}},
			want: `{ "a\"": "REDACTED" }`,
		},
		{
			name: "escape/star in name",
			args: args{json: `{ "*": 1, "a*b":2 }`, keys: []string{`\*`, `a\*b`}},
			want: `{ "*": "REDACTED", "a*b":"REDACTED" }`,
		},
		{
			name: "escape/escaped star in name",
			args: args{json: `{ "\\*": 1,"\\\\*": 2,"\\*\\*": 3}`, keys: []string{`\\\*`, `\\\\\*`, `\\\*\\\*`}},
			want: `{ "\\*": "REDACTED","\\\\*": "REDACTED","\\*\\*": "REDACTED"}`,
		},
		{
			name: "escape/slash in name",
			args: args{json: `{ "\\":1}`, keys: []string{`\\`}},
			want: `{ "\\":"REDACTED"}`,
		},
		{
			name: "escape/# in name",
			args: args{json: `{ "#":1,"##":2,"a#b":3"}`, keys: []string{`\#`, `a\#b`}},
			want: `{ "#":"REDACTED","##":2,"a#b":"REDACTED"}`,
		},
		{
			name: "wildcard/all array elements",
			args: args{json: `{ "children": [ "Sara", "Alex", "Jack" ] }`, keys: []string{`children.#`}},
			want: `{ "children": [ "REDACTED", "REDACTED", "REDACTED" ] }`,
		},
		{
			name: "wildcard/certain field of all array elements",
			args: args{json: `{ "children": [ {"name":"Sara"}, "Alex", "Jack", [[{"name":"Greg"}]],{},7 ] }`,
				keys: []string{`children.#.name`, `children.0.name`, `children.3.0.0.name`}},
			want: `{ "children": [ {"name":"REDACTED"}, "Alex", "Jack", [[{"name":"REDACTED"}]],{},7 ] }`,
		},
		{
			name: "wildcard/all fields",
			args: args{json: `{ "a": "a", "name":"b" }`, keys: []string{`#`}},
			want: `{ "a": "REDACTED", "name":"REDACTED" }`,
		},
		{
			name: "wildcard/all fields of an object",
			args: args{json: `{ "a": {"a":1}, "name":"b" }`, keys: []string{`a.#`}},
			want: `{ "a": {"a":"REDACTED"}, "name":"b" }`,
		},
		{
			name: "recursive/one field",
			args: args{json: `{"a": 1}`, keys: []string{`*.a`}},
			want: `{"a": "REDACTED"}`,
		},
		{
			name: "recursive/several stars",
			args: args{json: `{"a": 1, "x":{"b":263, "a":{"b":297, "a":{"x":{"a":{"b":491}}}}}}`, keys: []string{`*.a.*.b`}},
			want: `{"a": 1, "x":{"b":263, "a":{"b":"REDACTED", "a":{"x":{"a":{"b":"REDACTED"}}}}}}`,
		},
		{
			name: "recursive/two field",
			args: args{json: `{"a": 1, "h":{"a":95, "b":466, "k":{"y":{"a":198, "t":109}}}}`, keys: []string{`*.a`, `*.b`}},
			want: `{"a": "REDACTED", "h":{"a":"REDACTED", "b":"REDACTED", "k":{"y":{"a":"REDACTED", "t":109}}}}`,
		},
		{
			name: "recursive/intersection in keys",
			args: args{json: `{"a": 1, "h":{"a":{"c":739,"b":467,"a":{"c":739,"b":467}}, "b":466, "k":{"y":{"a":198, "t":109}}}}`,
				keys: []string{`*.a.c`, `*.a.b`}},
			want: `{"a": 1, "h":{"a":{"c":"REDACTED","b":"REDACTED","a":{"c":"REDACTED","b":"REDACTED"}}, "b":466, "k":{"y":{"a":198, "t":109}}}}`,
		},
		//{TODO
		//	name: "recursive/intersection with static key",
		//	args: args{json: `{"a": 1, "h":{"a":{"c":739,"b":467,"a":{"c":739,"b":467}}, "b":466, "k":{"y":{"a":198, "t":109}}}}`,
		//		keys: []string{`*.a.c`, `a`}},
		//	want: `{"a": "REDACTED", "h":{"a":{"c":"REDACTED","b":467,"a":{"c":"REDACTED","b":467}}, "b":466, "k":{"y":{"a":198, "t":109}}}}`,
		//},
		//{
		//	name: "recursive/675432",
		//	args: args{json: `{ "a": 1, "b":{"c":{"n":3, "z":{"a":34,"k":654}}, "t":{"a":23, "z":0,"k":437}}}`,
		//		keys: []string{`*.a`, `b.c.n`, `b.c.*.k`}},
		//	want: `{ "a": "REDACTED", "b":{"c":{"n":"REDACTED", "z":{"a":"REDACTED","k":"REDACTED"}}, "t":{"a":"REDACTED", "z":0,"k":437}}}`,
		//},
		//TODO array index escape
		{
			name: "recursive/in middle",
			args: args{json: `{"a":{"b":{"name":"d","c":{"a":{"b":[[{"name":"d"},[{"name":"d"}]]],"name":"b"}}}},"name":"b"}`,
				keys: []string{`a.*.name`}},
			want: `{"a":{"b":{"name":"REDACTED","c":{"a":{"b":[[{"name":"REDACTED"},[{"name":"REDACTED"}]]],"name":"REDACTED"}}}},"name":"b"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redactor := NewRedactor(tt.args.keys, handler)
			assert.JSONEq(t, tt.want, redactor.Redact(tt.args.json))
			assert.JSONEq(t, tt.want, redactor.Redact(tt.args.json), "reuse redactor with same result")
		})
	}
}

func TestConcurrent(t *testing.T) {
	waitGroup := sync.WaitGroup{}
	redactor := NewRedactor([]string{`*.name`}, func(s string) string {
		return "REDACTED"
	})
	for i := 0; i < 1000; i++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for i := 0; i < 100; i++ {
				_ = redactor.Redact(`{ "a": {"b":{"name":"d","f":5}, "name":"b" }}`)
			}
		}()
	}

	waitGroup.Wait()
}

/*
goos: darwin
goarch: arm64
pkg: jsonredact
Benchmark/with_matched_keys-10         	  426970	      2787 ns/op
Benchmark/without_matched_keys-10      	  547496	      2211 ns/op
*/
func Benchmark(b *testing.B) {
	b.Run("with matched keys", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Redact(bigJson, []string{"age", "fav.movie", "friends", "name.last"}, func(s string) string { return `REDACTED` })
		}
	})
	b.Run("without matched keys", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Redact(bigJson, []string{"age1", "fav1.movie", "1friends", "1name.last"}, func(s string) string { return `REDACTED` })
		}
	})
}
