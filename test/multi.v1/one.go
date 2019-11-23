// +build ignore

package multi

import (
	"fmt"
)

const (
	removedInPatch = "cool"
	RemovedInMajor = "neat"
	ChangedInMajor = 0
)

var (
	unexportedVarString = "cool"
	ExportedVarString = "neat"
	unexportedVarInt = 1
	ExportedVarInt = 0
)

// Wrapper is a wrapper type
type Wrapper bool

// Inter is for testing interface parsing
type Inter interface {
	Print(string)
	Break(string)
	Loop() Recurse
}

type Recurse interface {
	Loop() Inter
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
}

func unexportedFunc(stringArg string) (int, error) {
	return 0, nil
}

func ExportedFunc(i int) (str string, b bool) {
	return
}
