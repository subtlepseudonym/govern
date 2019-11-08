package main

import (
	"encoding/json"
	"fmt"

	"github.com/subtlepseudonym/govern"
)

const testFile = "testcode/full.go"

func main() {
	pkg, err := govern.ParseFile(testFile)
	if err != nil {
		panic(err)
	}

	b, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))
}
