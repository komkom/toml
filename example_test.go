package toml_test

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/komkom/toml"
)

func ExampleReader() {

	doc := `
[some]
toml="doc"
[to.map]
"!"=true`

	dec := json.NewDecoder(toml.New(bytes.NewBufferString(doc)))

	st := struct {
		Some struct {
			Toml string `json:"toml"`
		} `json:"some"`

		To struct {
			Map struct {
				IsMarked bool `json:"!"`
			}
		} `json:"to"`
	}{}

	err := dec.Decode(&st)
	if err != nil {
		panic(err)
	}

	fmt.Printf("toml: %v", st.Some.Toml)
	fmt.Printf(" is_marked: %v", st.To.Map.IsMarked)

	// Output: toml: doc is_marked: true
}
