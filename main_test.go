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
		json string
		keys []string
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
			args: args{json: bigJson},
			want: bigJson,
		},
		{
			name: "try to confuse with point",
			args: args{json: `{ "fav.movie": { "title": "Deer Hunter" }, "fav": { "movie": "BIG" } }`,
				keys: []string{"fav.movie"}},
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
				keys: []string{`children.1`, `children.6`}},
			want: `{ "children": [ "Sara", "REDACTED", "Jack" ] }`,
		},
		{
			name: "all array elements",
			args: args{json: `{ "children": [ "Sara", "Alex", "Jack" ] }`,
				keys: []string{`children.#`}},
			want: `{ "children": [ "REDACTED", "REDACTED", "REDACTED" ] }`,
		},
		{
			name: "certain field of all array elements",
			args: args{json: `{ "children": [ {"name":"Sara"}, "Alex", "Jack", [[{"name":"Greg"}]],{},7 ] }`,
				keys: []string{`children.#.name`, `children.0.name`}},
			want: `{ "children": [ {"name":"REDACTED"}, "Alex", "Jack", [[{"name":"Greg"}]],{},7 ] }`,
		},
		{
			name: "certain field of certain array element",
			args: args{json: `{ "children": [ {"name":"Sara"}, "Alex", "Jack" ] }`,
				keys: []string{`children.0.name`, `children.2.name`}},
			want: `{ "children": [ {"name":"REDACTED"}, "Alex", "Jack" ] }`,
		},
		{
			name: "all fields",
			args: args{json: `{ "a": "a", "name":"b" }`,
				keys: []string{`#`}},
			want: `{ "a": "REDACTED", "name":"REDACTED" }`,
		},
		{
			name: "all fields of an object",
			args: args{json: `{ "a": {"a":1}, "name":"b" }`,
				keys: []string{`a.#`}},
			want: `{ "a": {"a":"REDACTED"}, "name":"b" }`,
		},
		{
			name: "certain field of all fields and subfields",
			args: args{json: `{ "a": {"b":{"name":"d"}, "name":"b" }}`,
				keys: []string{`*.name`}},
			want: `{ "a": {"b":{"name":"REDACTED"}, "name":"REDACTED" }}`,
		},
		{
			name: "certain field of all fields and subfields of a certain object",
			args: args{json: `{"a":{"b":{"name":"d","c":{"a":{"b":[[{"name":"d"},[{"name":"d"}]]],"name":"b"}}}},"name":"b"}`,
				keys: []string{`a.*.name`}},
			want: `{"a":{"b":{"name":"REDACTED","c":{"a":{"b":[[{"name":"REDACTED"},[{"name":"REDACTED"}]]],"name":"REDACTED"}}}},"name":"b"}`,
		},
		{
			name: "many different fields",
			args: args{json: bigJson, keys: []string{"age", "fav.movie", "friends", "name.last"}},
			want: `{
		 "age": "REDACTED",
		 "children": [ "Sara", "Alex", "Jack" ],
		 "fav.movie": { "title": "Deer Hunter" },
		 "fav": { "movie": "REDACTED" },
		 "friends": "REDACTED",
		 "name": { "first": "Tom", "last": "REDACTED" }
		}`,
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.name), func(t *testing.T) {
			assert.JSONEq(t, tt.want, Redact(tt.args.json, tt.args.keys, handler))
		})
	}
}

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
