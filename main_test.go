package jsonredact

import (
	"fmt"
	"github.com/stretchr/testify/assert"
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
		json    string
		keys    []string
		handler func(string) string
	}
	handler := func(s string) string {
		return `"REDACTED"`
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "no selectors - return as is",
			args: args{json: bigJson, handler: handler},
			want: bigJson,
		},
		{
			name: "try to confuse with point",
			args: args{json: `{ "fav.movie": { "title": "Deer Hunter" }, "fav": { "movie": "BIG" } }`,
				keys:    []string{"fav.movie"},
				handler: handler},
			want: `{ "fav.movie": { "title": "Deer Hunter" }, "fav": { "movie": "REDACTED" } }`,
		},
		//		{
		//			name: "do not count escaped point",
		//			args: args{json: `{ "fav.movie": { "title": "Deer Hunter" }, "fav": { "movie": "BIG" } }`,
		//				keys:    []string{`fav\.movie`},
		//				handler: handler},
		//			want: `{ "fav.movie": "REDACTED", "fav": { "movie": "BIG" } }`,
		//		},
		{
			name: "array with index",
			args: args{json: `{ "children": [ "Sara", "Alex", "Jack" ] }`,
				keys:    []string{`children.1`, `children.6`},
				handler: handler},
			want: `{ "children": [ "Sara", "REDACTED", "Jack" ] }`,
		},
		{
			name: "all array elements",
			args: args{json: `{ "children": [ "Sara", "Alex", "Jack" ] }`,
				keys:    []string{`children.#`},
				handler: handler},
			want: `{ "children": [ "REDACTED", "REDACTED", "REDACTED" ] }`,
		},
		{
			name: "certain field of all array elements",
			args: args{json: `{ "children": [ {"name":"Sara"}, "Alex", "Jack", [[{"name":"Greg"}]],{},7 ] }`,
				keys:    []string{`children.#.name`, `children.0.name`},
				handler: handler},
			want: `{ "children": [ {"name":"REDACTED"}, "Alex", "Jack", [[{"name":"Greg"}]],{},7 ] }`,
		},
		{
			name: "certain field of certain array element",
			args: args{json: `{ "children": [ {"name":"Sara"}, "Alex", "Jack" ] }`,
				keys:    []string{`children.0.name`, `children.2.name`},
				handler: handler},
			want: `{ "children": [ {"name":"REDACTED"}, "Alex", "Jack" ] }`,
		},
		//		{
		//			name: "certain field of all fields and subfields",
		//			args: args{json: `{ "a": {"b":{"name":"d"}, "name":"b" }`,
		//				keys:    []string{`*.name`},
		//				handler: handler},
		//			want: `{ "a": {"b":{"name":"REDACTED"}, "name":"REDACTED" }`,
		//		},
		//		{
		//			name: "certain field of all fields and subfields of a certain object",
		//			args: args{json: `{ "a": {"b":{"name":"d", "c":{ "a": {"b":{"name":"d"}, "name":"b" }}, "name":"b" }`,
		//				keys:    []string{`a.*.name`},
		//				handler: handler},
		//			want: `{ "a": {"b":{"name":"REDACTED"}, "name":"REDACTED" }`,
		//		},
		//		{
		//			args: args{json: bigJson, keys: []string{"age", "fav.movie", "friends", "name.last"}, handler: handler},
		//			want: `{
		//  "age": "REDACTED",
		//  "children": [ "Sara", "Alex", "Jack" ],
		//  "fav.movie": { "title": "Deer Hunter" },
		//  "fav": { "movie": "REDACTED" },
		//  "friends": "REDACTED",
		//  "name": { "first": "Tom", "last": "REDACTED" }
		//}`,
		//		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.name), func(t *testing.T) {
			assert.JSONEq(t, tt.want, Redact(tt.args.json, tt.args.keys, tt.args.handler))
		})
	}
}

/*
age
name.first
a.*.name
children.#.name
children.#
children.1
children.1.name
*/

/*
goos: darwin
goarch: arm64
pkg: jsonredact
Benchmark
Benchmark-10    	  698648	      1606 ns/op
*/
func Benchmark(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Redact(bigJson, []string{"age", "fav.movie", "friends", "name.last"}, func(s string) string { return `"REDACTED"` })
	}
}
