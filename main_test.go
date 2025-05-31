package jsonredact

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	bigJson  = `[{"id":0,"name":"Elijah","city":"Austin","age":78,"friends":[{"name":"Michelle","hobbies":["Watching Sports","Reading","Skiing & Snowboarding"]},{"name":"Robert","hobbies":["Traveling","Video Games"]}]},{"id":1,"name":"Noah","city":"Boston","age":97,"friends":[{"name":"Oliver","hobbies":["Watching Sports","Skiing & Snowboarding","Collecting"]},{"name":"Olivia","hobbies":["Running","Music","Woodworking"]},{"name":"Robert","hobbies":["Woodworking","Calligraphy","Genealogy"]},{"name":"Ava","hobbies":["Walking","Church Activities"]},{"name":"Michael","hobbies":["Music","Church Activities"]},{"name":"Michael","hobbies":["Martial Arts","Painting","Jewelry Making"]}]},{"id":2,"name":"Evy","city":"San Diego","age":48,"friends":[{"name":"Joe","hobbies":["Reading","Volunteer Work"]},{"name":"Joe","hobbies":["Genealogy","Golf"]},{"name":"Oliver","hobbies":["Collecting","Writing","Bicycling"]},{"name":"Liam","hobbies":["Church Activities","Jewelry Making"]},{"name":"Amelia","hobbies":["Calligraphy","Dancing"]}]},{"id":3,"name":"Oliver","city":"St. Louis","age":39,"friends":[{"name":"Mateo","hobbies":["Watching Sports","Gardening"]},{"name":"Nora","hobbies":["Traveling","Team Sports"]},{"name":"Ava","hobbies":["Church Activities","Running"]},{"name":"Amelia","hobbies":["Gardening","Board Games","Watching Sports"]},{"name":"Leo","hobbies":["Martial Arts","Video Games","Reading"]}]},{"id":4,"name":"Michael","city":"St. Louis","age":95,"friends":[{"name":"Mateo","hobbies":["Movie Watching","Collecting"]},{"name":"Chris","hobbies":["Housework","Bicycling","Collecting"]}]},{"id":5,"name":"Michael","city":"Portland","age":19,"friends":[{"name":"Jack","hobbies":["Painting","Television"]},{"name":"Oliver","hobbies":["Walking","Watching Sports","Movie Watching"]},{"name":"Charlotte","hobbies":["Podcasts","Jewelry Making"]},{"name":"Elijah","hobbies":["Eating Out","Painting"]}]},{"id":6,"name":"Lucas","city":"Austin","age":76,"friends":[{"name":"John","hobbies":["Genealogy","Cooking"]},{"name":"John","hobbies":["Socializing","Yoga"]}]},{"id":7,"name":"Michelle","city":"San Antonio","age":25,"friends":[{"name":"Jack","hobbies":["Music","Golf"]},{"name":"Daniel","hobbies":["Socializing","Housework","Walking"]},{"name":"Robert","hobbies":["Collecting","Walking"]},{"name":"Nora","hobbies":["Painting","Church Activities"]},{"name":"Mia","hobbies":["Running","Painting"]}]},{"id":8,"name":"Emily","city":"Austin","age":61,"friends":[{"name":"Nora","hobbies":["Bicycling","Skiing & Snowboarding","Watching Sports"]},{"name":"Ava","hobbies":["Writing","Reading","Collecting"]},{"name":"Amelia","hobbies":["Eating Out","Watching Sports"]},{"name":"Daniel","hobbies":["Skiing & Snowboarding","Martial Arts","Writing"]},{"name":"Zoey","hobbies":["Board Games","Tennis"]}]},{"id":9,"name":"Liam","city":"New Orleans","age":33,"friends":[{"name":"Chloe","hobbies":["Traveling","Bicycling","Shopping"]},{"name":"Evy","hobbies":["Eating Out","Watching Sports"]},{"name":"Grace","hobbies":["Jewelry Making","Yoga","Podcasts"]}]}]`
	deepJson = `{"b":{"a":{"b":{"b":{"a":{"b":{"b":{"a":{"b":{"b":{"a":{"b":{"b":{"a":{"b":{"b":{"a":{"b":{"b":{"a":{"b":{"b":{"a":{"b":{"b":{"a":{"b":935}}}}}}}}}}}}}}}}}}}}}}}}}}}`
	handler  = func(s string) string { return `REDACTED` }
	random   = rand.New(rand.NewSource(time.Now().UnixNano()))
)

func TestRedact(t *testing.T) {
	type args struct {
		json string
		keys []string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "base/non json - return as is",
			args: args{json: "ynbtrvcew98hguibrfd", keys: []string{"b", "a"}},
			want: "ynbtrvcew98hguibrfd",
		},
		{
			name: "base/no selectors - return as is",
			args: args{json: bigJson},
			want: bigJson,
		},
		{
			name: "base/deepJson",
			args: args{json: deepJson, keys: []string{"b", "a"}},
			want: `{"b":"REDACTED"}`,
		},
		{
			name: "base/plain path of 0 depth",
			args: args{json: `{"a":459,"b":707,"c":116, "x":{"terminal":577}"}`, keys: []string{"a", "b", "x.terminal"}},
			want: `{"a":"REDACTED","b":"REDACTED","c":116, "x":{"terminal":"REDACTED"}}`,
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
			args: args{json: `{"a":[18,2,{"c":1,"d":{"e":2}}],"b":2}`, keys: []string{"a.1"}},
			want: `{"a":[18,"REDACTED",{"c":1,"d":{"e":2}}],"b":2}`,
		},
		{
			name: "array/with indexes",
			args: args{json: `{"a":[1,2,{"c":1,"d":{"e":2}}],"b":2}`, keys: []string{"a.1", "a.0"}},
			want: `{"a":["REDACTED","REDACTED",{"c":1,"d":{"e":2}}],"b":2}`,
		},
		{
			name: "array/root array",
			args: args{json: `[{"a":[1,2,{"c":1,"d":{"e":2}}],"b":2}`, keys: []string{"0.a.1", "0.a.0"}},
			want: `[{"a":["REDACTED","REDACTED",{"c":1,"d":{"e":2}}],"b":2}]`,
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
			args: args{json: `{"a.b":1,"a":{"b": 2 } }`, keys: []string{"a.b"}},
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
			name: "escape/number in name",
			args: args{json: `{ "0":232, "453":171, "4":406, "1":{"2":332, "3":946}, "5.6":122, "5.7":122}`,
				keys: []string{`0`, `453`, `1.2`, `5\.6`}},
			want: `{ "0":"REDACTED", "453":"REDACTED", "4":406, "1":{"2":"REDACTED", "3":946}, "5.6":"REDACTED", "5.7":122}`,
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
			name: "recursive/intersection with prefix",
			args: args{json: `{"a": 1, "h":{"a":{"c":739,"b":467,"a":{"c":739,"b":467}}, "b":466, "k":{"y":{"a":198, "t":109}}}}`,
				keys: []string{`*.a.c`, `*.a.b`}},
			want: `{"a": 1, "h":{"a":{"c":"REDACTED","b":"REDACTED","a":{"c":"REDACTED","b":"REDACTED"}}, "b":466, "k":{"y":{"a":198, "t":109}}}}`,
		},
		{
			name: "recursive/intersection with static key",
			args: args{json: `{"a": 1, "h":{"a":{"c":739,"b":467,"a":{"c":739,"b":467}}, "b":466, "k":{"y":{"a":198, "t":109}}}}`,
				keys: []string{`*.a.c`, `a`}},
			want: `{"a": "REDACTED", "h":{"a":{"c":"REDACTED","b":467,"a":{"c":"REDACTED","b":467}}, "b":466, "k":{"y":{"a":198, "t":109}}}}`,
		},
		{
			name: "recursive/intersection without prefix",
			args: args{json: `{ "a": 1, "b":{"c":{"n":3, "z":{"a":34,"k":654}}, "t":{"a":23, "z":0,"k":437}}}`,
				keys: []string{`*.a`, `b.c.n`, `b.c.*.k`}},
			want: `{ "a": "REDACTED", "b":{"c":{"n":"REDACTED", "z":{"a":"REDACTED","k":"REDACTED"}}, "t":{"a":"REDACTED", "z":0,"k":437}}}`,
		},
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
			fmt.Println(redactor.automata)
			require.JSONEq(t, tt.want, redactor.Redact(tt.args.json))
		})
	}
}

func TestConcurrent(t *testing.T) {
	waitGroup := sync.WaitGroup{}
	redactor := NewRedactor([]string{`*.name`}, handler)
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

func Test(t *testing.T) {
	h := func(s string) string { return `` }
	output := NewRedactor([]string{`a.b.f`}, h).Redact(`{ "a": {"b":{"name":"d","f":5}, "name":"b" }}`)
	fmt.Println(output)
	//{ "a": {"b":{"name":"d","f":""},"name":"b"}}
}

/*
goos: darwin
goarch: arm64
cpu: Apple M1
Benchmark/complexity/1-8        15272268                78.14 ns/op            0 B/op          0 allocs/op
Benchmark/complexity/10-8        2192635               545.9 ns/op             0 B/op          0 allocs/op
Benchmark/complexity/100-8        185149              6397 ns/op               0 B/op          0 allocs/op
Benchmark/complexity/1000-8        15116             79228 ns/op               0 B/op          0 allocs/op
Benchmark/complexity/10000-8        1408            844426 ns/op               0 B/op          0 allocs/op
*/
func Benchmark(b *testing.B) {
	b.ReportAllocs()
	b.Run("complexity", func(b *testing.B) {
		redactor := NewRedactor([]string{"*.nomatch"}, handler)
		for n := range 5 {
			size := int(math.Pow10(n))
			input := generateJSON(size)
			b.Run(strconv.Itoa(size), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					_ = redactor.Redact(input)
				}
			})
		}
	})
}

func generateJSON(keysNum int) string {
	strBuilder := strings.Builder{}
	strBuilder.WriteByte('{')
	for i := range keysNum {
		strBuilder.WriteByte('"')
		strBuilder.WriteString(stringRandom(5))
		strBuilder.WriteByte('"')
		strBuilder.WriteByte(':')
		strBuilder.WriteByte('"')
		strBuilder.WriteString(stringRandom(10))
		strBuilder.WriteByte('"')
		if i != keysNum-1 {
			strBuilder.WriteByte(',')
		}
	}
	strBuilder.WriteByte('}')
	return strBuilder.String()
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func stringRandom(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[random.Intn(len(letterBytes))]
	}
	return string(b)
}
