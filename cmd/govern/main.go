package main

import (
	"fmt"
	"path"

	"github.com/subtlepseudonym/govern"
)

const (
	testDir    = "test"
	multiBase  = "multi.v1"
	multiPatch = "multi.v1.0.1"
	multiMinor = "multi.v1.1"
	multiMajor = "multi.v2"
)

func main() {
	base, err := govern.ParsePackage(path.Join(testDir, multiBase))
	if err != nil {
		panic(err)
	}

	patch, err := govern.ParsePackage(path.Join(testDir, multiPatch))
	if err != nil {
		panic(err)
	}

	minor, err := govern.ParsePackage(path.Join(testDir, multiMinor))
	if err != nil {
		panic(err)
	}

	major, err := govern.ParsePackage(path.Join(testDir, multiMajor))
	if err != nil {
		panic(err)
	}

	majorChange, minorChange, err := govern.ComparePackages(base, patch)
	fmt.Printf("base -> patch: %t %t\n", majorChange, minorChange)
	fmt.Println(err, "\n")

	majorChange, minorChange, err = govern.ComparePackages(base, minor)
	fmt.Printf("base -> minor: %t %t\n", majorChange, minorChange)
	fmt.Println(err, "\n")

	majorChange, minorChange, err = govern.ComparePackages(base, major)
	fmt.Printf("base -> major: %t %t\n", majorChange, minorChange)
	fmt.Println(err, "\n")
}
