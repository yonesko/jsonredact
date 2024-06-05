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
		return `REDACTED`
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
		{
			name: "fields with strange names",
			args: args{json: `{".":1, ".po.int.":2, "\"":3,"*":4, "..":5, "a*":7, "\\\\":8, "#":[{"0":{"#":11}}]}`,
				keys: []string{`\.`, `\.po\.int\.`, `\\"`, `\*`, `\#.0.0.\#`, `a*`}},
			want: `{".":"REDACTED", ".po.int.":"REDACTED", "\"":"REDACTED","*":"REDACTED","..":5,
				"a*":"REDACTED", "\\\\":8, "#":[{"0":{"#":"REDACTED"}}]}`,
		},
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
				keys: []string{`children.#.name`, `children.0.name`, `children.3.0.0.name`}},
			want: `{ "children": [ {"name":"REDACTED"}, "Alex", "Jack", [[{"name":"REDACTED"}]],{},7 ] }`,
		},
		{
			name: "certain field of certain array element",
			args: args{json: `{ "children": [ {"name":"Sara", "null":null}, "Alex", "Jack",null ] }`,
				keys: []string{`children.0.name`, `children.2.name`}},
			want: `{ "children": [ {"name":"REDACTED", "null":null}, "Alex", "Jack",null ] }`,
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
			args: args{json: `{ "a": {"b":{"name":"d","f":5}, "name":"b" }}`,
				keys: []string{`*.name`}},
			want: `{ "a": {"b":{"name":"REDACTED","f":5}, "name":"REDACTED" }}`,
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
		{
			name: "without matched keys",
			args: args{json: bigJson, keys: []string{"1age", "1fav.movie", "1friends", "1name.last"}},
			want: bigJson,
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
Benchmark/with_matched_keys-10         	  663984	      1557 ns/op
Benchmark/without_matched_keys-10      	  877837	      1099 ns/op
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
