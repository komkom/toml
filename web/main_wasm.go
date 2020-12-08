package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"syscall/js"

	"github.com/komkom/toml2json"
	"github.com/pkg/errors"
)

var Document = js.Global().Get("document")

const (
	Fmt      = `div#format-button`
	Clear    = `input#clear`
	TOMLArea = `toml`
	JSONArea = `json`
	ErrorMsg = `errormsg`
)

func main() {

	js.Global().Set(`format`, js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		load()
		return nil
	}))

	fmt.Printf("xxxxxx33\n")

	<-make(chan struct{})
}

func load() {

	errMsg := Document.Call("getElementById", ErrorMsg)
	//errMsg.Set(`innerHTML`, `testtest`)
	//style := errMsg.Get(`style`)
	//style.Set(`display`, `none`)

	j, err := transform()
	if err != nil {
		errMsg.Set(`innerHTML`, err.Error())
		return
	}

	errMsg.Set(`innerHTML`, ``)
	Document.Call("getElementById", JSONArea).Set(`innerHTML`, j)
}

func transform() (string, error) {

	var edit string
	val := Document.Call("getElementById", TOMLArea).Get(`value`)
	if val.Truthy() {
		edit = val.String()
	}

	r := strings.NewReader(edit)
	rd := toml2json.New(r)
	data, err := ioutil.ReadAll(rd)

	fmt.Printf("____u %v\n", err)
	if err != nil {
		return ``, err
	}

	fmt.Printf("____u %s\n", data)

	if !json.Valid(data) {
		return ``, fmt.Errorf(`generated josn not valid`)
	}

	str, err := PrettyJSON(data)
	if err != nil {
		return ``, errors.Wrap(err, `transform prettyJSON failed`)
	}

	return str, nil
}

func PrettyJSON(jsn []byte) (string, error) {

	var pretty bytes.Buffer
	err := json.Indent(&pretty, jsn, "", "&nbsp;&nbsp;&nbsp;")
	if err != nil {
		return ``, err
	}

	return string(pretty.Bytes()), nil
}
