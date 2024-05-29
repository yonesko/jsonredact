package jsonredact

import (
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
			name: "no keys - don't change input",
			args: args{json: bigJson, handler: handler},
			want: bigJson,
		},
		{
			name: "point confusion",
			args: args{json: `{ "fav.movie": { "title": "Deer Hunter" }, "fav": { "movie": "BIG" } }`,
				keys:    []string{"fav.movie"},
				handler: handler},
			want: `{ "fav.movie": { "title": "Deer Hunter" }, "fav": { "movie": "REDACTED" } }`,
		},
		{
			name: "point confusion flat",
			args: args{json: `{ "fav.movie": { "title": "Deer Hunter" }, "fav": { "movie": "BIG" } }`,
				keys:    []string{`fav\.movie`},
				handler: handler},
			want: `{ "fav.movie": "REDACTED", "fav": { "movie": "BIG" } }`,
		},
		{
			name: "many fields",
			args: args{json: bigJson, keys: []string{"age", "fav.movie", "friends", "name.last"}, handler: handler},
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
		t.Run(tt.name, func(t *testing.T) {
			assert.JSONEq(t, tt.want, Redact(tt.args.json, tt.args.keys, tt.args.handler))
		})
	}
}

/*
goos: darwin
goarch: arm64
pkg: jsonredact
Benchmark
Benchmark-10    	  493168	      2410 ns/op
*/
func Benchmark(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Redact(bigJson, []string{"age", "fav.movie", "friends", "name.last"}, func(s string) string { return `"REDACTED"` })
	}
}
