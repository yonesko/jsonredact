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
h := func (s string) string { return `` }
output := NewRedactor([]string{`a.b.f`}, h).
Redact(`{ "a": {"b":{"name":"d","f":5}, "name":"b" }}`)
fmt.Println(output)
//{ "a": {"b":{"name":"d","f":""},"name":"b"}}
```

### Expressions

### Performance