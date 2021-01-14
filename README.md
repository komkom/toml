# toml
A Toml parser and JSON transformer.

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

In the repo there are two benchmarks comparing throughputs of just reading data from memory versus also transforming and parsing the data. The parser slows down data throughput roughly 15x here.
The overhead introduced is neglectable for most use cases.

```
BenchmarkParserThroughput-12    	   67611	     17595 ns/op	   7.05 MB/s
BenchmarkMemoryThroughput-12    	 1000000	      1270 ns/op	 100.03 MB/s
```

