[![Go Report Card](https://goreportcard.com/badge/github.com/komkom/toml)](https://goreportcard.com/report/github.com/komkom/toml)
[![GoDoc](https://godoc.org/github.com/komkom/toml?status.svg)](https://godoc.org/github.com/komkom/toml)
 
# toml
A Toml parser and JSON encoder.

The parser is compliant with the toml specs 1.0.0-rc3.

[give it a try](https://komkom.github.io/toml/)

Installation:

```
go get github.com/komkom/toml
```

# Unmarshaling a toml doc

Since the parser transforms a toml in stream into a valid json, normal json unmarshaling from the std lib can be used.

```
doc := `
[some]
toml="doc"`

dec := json.NewDecoder(toml.New(bytes.NewBufferString(doc)))

st := struct {
  Some struct {
    Toml string `json:"toml"`
  } `json:"some"`
}{}

err := dec.Decode(&st)
if err != nil {
  panic(err)
}
        
fmt.Printf("toml: %v\n", st.Some.Toml)
```

# Performance Considerations

In the repo there are two benchmarks comparing throughputs of just reading data from memory versus also transforming and parsing the data. The parser slows down data throughput around 15x here.
These benchmarks are by no means thorough and only hint at an estimate.

```
Parser Throughput    7.05 MB/s
Memory Throughput    100.03 MB/s
```

