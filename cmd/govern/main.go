package main

import (
	"fmt"
	"go/ast"
	"go/types"
	"go/importer"
	"go/parser"
	"go/token"
	"path"
)

const (
	testDir = "test"
	multiBase = "multi.v1"
	multiPatch = "multi.v1.0.1"
	multiMinor = "multi.v1.1"
	multiMajor = "multi.v2"
)

func main() {
	base, err := parsePackage(path.Join(testDir, multiBase))
	if err != nil {
		panic(err)
	}

	patch, err := parsePackage(path.Join(testDir, multiPatch))
	if err != nil {
		panic(err)
	}

	minor, err := parsePackage(path.Join(testDir, multiMinor))
	if err != nil {
		panic(err)
	}

	major, err := parsePackage(path.Join(testDir, multiMinor))
	if err != nil {
		panic(err)
	}

	majorChange, minorChange := comparePackages(base, patch)
	fmt.Printf("base -> patch: %t %t\n", majorChange, minorChange)

	majorChange, minorChange = comparePackages(base, minor)
	fmt.Printf("base -> minor: %t %t\n", majorChange, minorChange)

	majorChange, minorChange = comparePackages(base, major)
	fmt.Printf("base -> major: %t %t\n", majorChange, minorChange)
}

func parsePackage(dir string) (*types.Package, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parse directory: %v", err)
	}

	var files []*ast.File
	for _, file := range pkgs["multi"].Files {
		files = append(files, file)
	}

	conf := types.Config{
		Importer: importer.Default(),
	}

	pkg, err := conf.Check("multi", fset, files, nil)
	if err != nil {
		return nil, fmt.Errorf("code check: %v", err)
	}

	return pkg, nil
}

func comparePackages(older, newer *types.Package) (major, minor bool) {
	if older.Name() != newer.Name() {
		major = true
		fmt.Println("name mismatch")
		fmt.Println("older.Name()", older.Name())
		fmt.Println("newer.Name()", newer.Name())
	}

	scopeMajor, scopeMinor := compareScopes(older.Scope(), newer.Scope())
	return major || scopeMajor, scopeMinor
}

func compareScopes(older, newer *types.Scope) (major, minor bool) {
	oldObjs := make(map[string]types.Object)
	for _, name := range older.Names() {
		oldObjs[name] = older.Lookup(name)
	}

	newObjs := make(map[string]types.Object)
	for _, name := range newer.Names() {
		newObjs[name] = newer.Lookup(name)
	}

	for name, oldObj := range oldObjs {
		delete(oldObjs, name)
		if !oldObj.Exported() {
			continue
		}

		newObj, exists := newObjs[name]
		if !exists || !crossPackageIdentical(oldObj.Type(), newObj.Type()) || !newObj.Exported() {
			fmt.Println(name)
			major = true
		}

		delete(newObjs, name)
	}

	for _, obj := range newObjs {
		if obj.Exported() {
			minor = true
			break
		}
	}

	return major, minor
}

func crossPackageIdentical(x, y types.Type) bool {
	_, xNamed := x.(*types.Named)
	_, yNamed := y.(*types.Named)

	if xNamed && yNamed {
		return types.Identical(x.Underlying(), y.Underlying())
	}
	return types.Identical(x, y)
}
