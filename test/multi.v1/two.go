// +build ignore

package multi

import (
	"fmt"
	"os"
)

const Int = 0

// Object is for testing struct parsing
type Object struct {
	Wrap Wrapper
	Internal string
	unexport string
}

// Print is defined on Object
func (o Object) Print(s string) {
	fmt.Fprintln(os.Stdout, s)
}

