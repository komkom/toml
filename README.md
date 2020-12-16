# toml
A Toml parser and Json transformer

This is work in progress.

At the moment the parser should be compatible with the toml specs 1.0.0-rc3

[please give it a try](https://komkom.github.io/toml/)

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
