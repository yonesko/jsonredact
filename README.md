### JSONRedact redacts sensitive fields from JSON

* Production tested
* Doesn't waste your cpu and mem

### Install

```bash
go get github.com/yonesko/jsonredact@latest
```

### Use

Describe fields you want to handle by expressions.

And provide handler for replacing matching values.
You can replace with empty string, with placeholder, cipher it or whatever you want!

```go
package main

import (
	"fmt"
	"github.com/yonesko/jsonredact"
)

function func main() {
	h := func(s string) string { return `` }
	output := jsonredact.NewRedactor([]string{`a.b.f`}, h).
		Redact(`{ "a": {"b":{"name":"d","f":5}, "name":"b" }}`)
	fmt.Println(output)
	//{ "a": {"b":{"name":"d","f":""},"name":"b"}}
}

```

### Expressions

User `.` as separator of objects and arrays.

Use `#` as wildcard for any key or array index.

Use `*` to apply right expression to all object keys found under path of left expression recursively. (makes redactor
walk the whole json)

User `\` to escape control symbols above.

| Expression | Comment                                                                                  |
|------------|------------------------------------------------------------------------------------------|
| `a`        | Match key 'a' in the root of json                                                        |
| `a.b`      | Match key 'b' in the root of object 'a'. <br/>If 'a' is not an object, ignore expression |
| `a\.b`     | Match key 'a.b' in the root of json                                                      |
| `a.#`      | Match any key in the root of object 'a'                                                  |
| `a.#.c`    | Match key 'c' of every children object of a                                              |
| `*.a`      | Match key 'a' of every object in json recursively                                        |
| `a.*.b`    | Match key 'b' of every object in object 'a' recursively                                  |
| `*.a.*.b`  | Match key 'b' of every object in object 'a' found at eny depth recursively               |

### Performance

Redactor operates like a regex - it compiles expressions into automata once (constructor NewRedactor) then runs jsons
against it. So complexity is
O(n), where n is size of the input.

Here benchmark complexity/n, where n is number of keys of flat json.

```bash
goos: darwin
goarch: arm64
cpu: Apple M1
Benchmark/complexity/1-8        15272268                78.14 ns/op            0 B/op          0 allocs/op
Benchmark/complexity/10-8        2192635               545.9 ns/op             0 B/op          0 allocs/op
Benchmark/complexity/100-8        185149              6397 ns/op               0 B/op          0 allocs/op
Benchmark/complexity/1000-8        15116             79228 ns/op               0 B/op          0 allocs/op
Benchmark/complexity/10000-8        1408            844426 ns/op               0 B/op          0 allocs/op
```

Redactor doesn't traverse all the json until it told so using `*` wildcard.