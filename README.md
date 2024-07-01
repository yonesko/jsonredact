### JSONRedact redacts sensitive fields from JSON

```go
//User '.' as separator of objects and arrays.
//Use '#' as wildcard for any key or array index.
//Use '*' to apply right expression to all object keys recursively.
//User '\' to escape control symbols above.

redactor := NewRedactor([]string{`a.b`, `c`}, func(s string) string {
		return "REDACTED"
	}

redactedJson := redactor.Redact(`{"a":{"b":"secret", "c":"Jack"}, "c":"secret"}`)
//{"a":{"b":"REDACTED", "c":"Jack"}, "c":"REDACTED"}
```

* Production tested
* Doesn't waste your cpu and mem