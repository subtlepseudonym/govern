// +build ignore

package test

import (
	"fmt"

	"github.com/subtlepseudonym/notes"
)

const (
	unexportedConstantString = "cool"
	ExportedConstantString = "neat"
	unexportedConstantInt = 1
	ExportedConstantInt = 0
)

var (
	unexportedVarString = "cool"
	ExportedVarString = "neat"
	unexportedVarInt = 1
	ExportedVarInt = 0
)

// Object is for testing struct parsing
type Object struct {
	Internal string
	unexport string
}

// Print is defined on Object
func (o Object) Print(s string) {
	fmt.Println(s)
}

// Inter is for testing interface parsing
type Inter interface {
	Print(string)
	Break(string)
}

func test() {
	t := Object{
		Internal: "test",
		unexport: "neat",
	}
	t.Print(t.Internal)

	i, err := unexportedFunc(t.Internal)
	if err != nil {
		panic(err)
	}
	fmt.Println(i)

	s, b := ExportedFunc(0)
	fmt.Println(s, b)

	fmt.Println(notes.Note{})
}

func unexportedFunc(stringArg string) (int, error) {
	return 0, nil
}

func ExportedFunc(i int) (str string, b bool) {
	return
}
