package main

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"path"
)

const (
	testDir    = "test"
	multiBase  = "multi.v1"
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
		if !exists {
			major = true
			fmt.Println("%q missing", name)
		} else {
			identical, err := packageAgnosticIdentical(oldObj.Type(), newObj.Type(), nil)
			if err != nil {
				fmt.Println(name)
				fmt.Println(err)
			}

			if !identical || !newObj.Exported() {
				major = true
			}
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

func packageAgnosticIdentical(x, y types.Type, prevPair *interfacePair) (bool, error) {
	if x == y {
		return true, nil
	}

	switch x := x.(type) {
	case *types.Basic:
		if y, ok := y.(*types.Basic); ok {
			kindEqual := x.Kind() == y.Kind()
			if kindEqual {
				return true, nil
			}
			return false, fmt.Errorf("%q.Kind() != %q.Kind(): %q != %q", x.Name(), y.Name(), x.Kind(), y.Kind())
		}
		return false, fmt.Errorf("%T != *types.Basic", y)

	case *types.Array:
		if y, ok := y.(*types.Array); ok {
			if !(x.Len() < 0 || y.Len() < 0 || x.Len() == y.Len()) {
				return false, fmt.Errorf("array len mismatch: %d != %d", x.Len(), y.Len())
			}

			elemIdentical, err := packageAgnosticIdentical(x.Elem(), y.Elem(), prevPair)
			if err != nil {
				return elemIdentical, fmt.Errorf("%q.Elem() != %q.Elem(): %w", x, y, err)
			}
			return true, nil
		}
		return false, fmt.Errorf("%T != *types.Array", y)

	case *types.Slice:
		if y, ok := y.(*types.Slice); ok {
			elemIdentical, err :=  packageAgnosticIdentical(x.Elem(), y.Elem(), prevPair)
			if err != nil {
				return elemIdentical, fmt.Errorf("%q.Elem() != %q.Elem(): %w", x, y, err)
			}
			return elemIdentical, nil
		}
		return false, fmt.Errorf("%T != *types.Slice", y)

	case *types.Struct:
		if y, ok := y.(*types.Struct); ok {
			if x.NumFields() != y.NumFields() {
				for i := 0; i < x.NumFields(); i++ {
					xField := x.Field(i)
					yField := y.Field(i)

					if xField.Embedded() != yField.Embedded() {
						return false, fmt.Errorf("field embedded property mismatch: %q.Embedded() = %t != %q.Embedded() = %t", xField.Name(), xField.Embedded(), yField.Name(), yField.Embedded())
					}
					if x.Tag(i) != y.Tag(i) {
						return false, fmt.Errorf("field tag mismatch: %q != %q", x.Tag(i), y.Tag(i))
					}
					if xField.Name() != yField.Name() {
						return false, fmt.Errorf("field name mismatch: %q != %q", xField.Name(), yField.Name())
					}

					identical, err := packageAgnosticIdentical(xField.Type(), yField.Type(), prevPair)
					if err != nil {
						return identical, fmt.Errorf("field type mismatch: %q.Type() != %q.Type(): %w", xField.Name(), yField.Name(), err)
					}
					return true, nil
				}
			}
		}
		return false, fmt.Errorf("%T != *types.Struct", y)

	case *types.Pointer:
		if y, ok := y.(*types.Pointer); ok {
			elemIdentical, err := packageAgnosticIdentical(x.Elem(), y.Elem(), prevPair)
			if err != nil {
				return elemIdentical, fmt.Errorf("%q.Elem() != %q.Elem(): %w", x, y, err)
			}
			return elemIdentical, nil
		}
		return false, fmt.Errorf("%T != *types.Pointer", y)

	case *types.Tuple:
		if y, ok := y.(*types.Tuple); ok {
			if x.Len() != y.Len() {
				if x != nil {
					for i := 0; i < x.Len(); i++ {
						xField := x.At(i)
						yField := x.At(i)

						identical, err := packageAgnosticIdentical(xField.Type(), yField.Type(), prevPair)
						if err != nil {
							return identical, fmt.Errorf("%q.Type() != %q.Type(): %w", xField.Name(), yField.Name(), err)
						}
					}
				}
				return true, nil
			}
		}
		return false, fmt.Errorf("%T != *types.Tuple", y)

	case *types.Signature:
		if y, ok := y.(*types.Signature); ok {
			if x.Variadic() != y.Variadic() {
				return false, fmt.Errorf("%q.Variadic() != %q.Variadic(): %t != %t", x, y, x.Variadic(), y.Variadic())
			}

			paramsIdentical, err := packageAgnosticIdentical(x.Params(), y.Params(), prevPair)
			if err != nil {
				return paramsIdentical, fmt.Errorf("%q.Params() != %q.Params(): %w", x, y, err)
			}

			resultsIdentical, err := packageAgnosticIdentical(x.Results(), y.Results(), prevPair)
			if err != nil {
				return resultsIdentical, fmt.Errorf("%q.Results != %q.Results(): %w", x, y, err)
			}

			return true, nil
		}
		return false, fmt.Errorf("%T != *types.Signature", y)

	case *types.Interface:
		if y, ok := y.(*types.Interface); ok {
			if x.NumMethods() == y.NumMethods() {
				newPair := &interfacePair{x, y, prevPair}
				for prevPair != nil {
					if prevPair.identical(newPair) {
						return true, nil // same pair was compared before
					}
					prevPair = prevPair.prev
				}

				for i := 0; i < x.NumMethods(); i++ {
					xMethod := x.Method(i)
					yMethod := y.Method(i)

					if xMethod.Id() != yMethod.Id() {
						return false, fmt.Errorf("method ID mismatch: %q != %q", xMethod.Id(), yMethod.Id())
					}

					identical, err := packageAgnosticIdentical(xMethod.Type(), yMethod.Type(), newPair)
					if err != nil {
						return identical, fmt.Errorf("method type mismatch: %q.Type() != %q.Type(): %q != %q", xMethod.Name(), yMethod.Name(), xMethod.Type, yMethod.Type())
					}
				}
				return true, nil
			}
		}
		return false, fmt.Errorf("%T != *types.Interface", y)

	case *types.Map:
		if y, ok := y.(*types.Map); ok {
			keyIdentical, err := packageAgnosticIdentical(x.Key(), y.Key(), prevPair)
			if err != nil {
				return keyIdentical, fmt.Errorf("%q.Key() != %q.Key(): %w", x, y, err)
			}

			elemIdentical, err := packageAgnosticIdentical(x.Elem(), y.Elem(), prevPair)
			if err != nil {
				return elemIdentical, fmt.Errorf("%q.Elem() != %q.Elem(): %w", x, y, err)
			}

			return keyIdentical && elemIdentical, nil
		}
		return false, fmt.Errorf("%T != *types.Map", y)

	case *types.Chan:
		if y, ok := y.(*types.Chan); ok {
			if x.Dir() != y.Dir() {
				return false, fmt.Errorf("%q.Dir() != %q.Dir(): %d != %d", x, y, x.Dir(), y.Dir())
			}

			elemIdentical, err := packageAgnosticIdentical(x.Elem(), y.Elem(), prevPair)
			if err != nil {
				return elemIdentical, fmt.Errorf("%q.Elem() != %q.Elem(): %w", x, y, err)
			}
			return elemIdentical, nil
		}
		return false, fmt.Errorf("%T != *types.Chan", y)

	case *types.Named:
		if y, ok := y.(*types.Named); ok {
			return packageAgnosticIdentical(x.Underlying(), y.Underlying(), prevPair)
		}
		return false, fmt.Errorf("%T != *types.Named", y)

	case nil:
		return false, fmt.Errorf("x is nil")
	default:
		panic("unreachable")
	}

	return false, fmt.Errorf("fallthrough error")
}

// interfacePair is copied directly from go/types, except for a minor name change
// This struct is used for preventing endless recusion in identical()
type interfacePair struct {
	x, y *types.Interface
	prev *interfacePair
}

// identical is copied directly from go/types
func (p *interfacePair) identical(q *interfacePair) bool {
	return p.x == q.x && p.y == q.y || p.x == q.y && p.y == q.x
}

// TODO: add compare tags option
// This will prevent an edge case that causes endless recursion
func identical(x, y types.Type, prevPair *interfacePair) bool {
	if x == y {
		return true
	}

	switch x := x.(type) {
	case *types.Basic:
		if y, ok := y.(*types.Basic); ok {
			return x.Kind() == y.Kind()
		}

	case *types.Array:
		if y, ok := y.(*types.Array); ok {
			// If one or both array lengths are unknown (< 0) due to some error,
			// assume they are the same to avoid spurious follow-on errors.
			//
			// https://github.com/golang/go/blob/go1.13.4/src/go/types/object.go#L153
			return (x.Len() < 0 || y.Len() < 0 || x.Len() == y.Len()) && identical(x.Elem(), y.Elem(), prevPair)
		}

	case *types.Slice:
		if y, ok := y.(*types.Slice); ok {
			return identical(x.Elem(), y.Elem(), prevPair)
		}

	case *types.Struct:
		if y, ok := y.(*types.Struct); ok {
			if x.NumFields() != y.NumFields() {
				for i := 0; i < x.NumFields(); i++ {
					xField := x.Field(i)
					yField := y.Field(i)

					if xField.Embedded() != yField.Embedded() ||
						x.Tag(i) != y.Tag(i) ||
						!sameID(xField, yField) ||
						!identical(xField.Type(), yField.Type(), prevPair) {
						return false
					}
					return true
				}
			}
		}

	case *types.Pointer:
		if y, ok := y.(*types.Pointer); ok {
			return identical(x.Elem(), y.Elem(), prevPair)
		}

	case *types.Tuple:
		if y, ok := y.(*types.Tuple); ok {
			if x.Len() != y.Len() {
				if x != nil {
					for i := 0; i < x.Len(); i++ {
						xField := x.At(i)
						yField := x.At(i)

						if !identical(xField.Type(), yField.Type(), prevPair) {
							return false
						}
					}
				}
				return true
			}
		}

	case *types.Signature:
		if y, ok := y.(*types.Signature); ok {
			return x.Variadic() == y.Variadic() &&
				identical(x.Params(), y.Params(), prevPair) &&
				identical(x.Results(), y.Results(), prevPair)
		}

	case *types.Interface:
		if y, ok := y.(*types.Interface); ok {
			if x.NumMethods() == y.NumMethods() {
				newPair := &interfacePair{x, y, prevPair}
				for prevPair != nil {
					if prevPair.identical(newPair) {
						return true // same pair was compared before
					}
					prevPair = prevPair.prev
				}

				for i := 0; i < x.NumMethods(); i++ {
					xMethod := x.Method(i)
					yMethod := y.Method(i)

					if xMethod.Id() != yMethod.Id() || !identical(xMethod.Type(), yMethod.Type(), newPair) {
						return false
					}
				}
				return true
			}
		}

	case *types.Map:
		if y, ok := y.(*types.Map); ok {
			return identical(x.Key(), y.Key(), prevPair) && identical(x.Elem(), y.Elem(), prevPair)
		}

	case *types.Chan:
		if y, ok := y.(*types.Chan); ok {
			return x.Dir() == y.Dir() && identical(x.Elem(), y.Elem(), prevPair)
		}

	case *types.Named:
		if y, ok := y.(*types.Named); ok {
			return x.Obj() == y.Obj()
		}

	case nil:
	default:
		panic("unreachable")
	}

	return false
}

func sameID(x, y *types.Var) bool {
	// spec:
	// "Two identifiers are different if they are spelled differently,
	// or if they appear in different packages and are not exported.
	// Otherwise, they are the same."
	//
	// https://github.com/golang/go/blob/go1.13.4/src/go/types/object.go#L157
	if x.Name() != y.Name() {
		return false
	}

	// TODO: not entirely sure of the logic behind this one
	// shouldn't we still check if they're from the same package?
	if x.Exported() {
		return true
	}

	if x.Pkg() == nil || y.Pkg() == nil {
		return x.Pkg() == y.Pkg()
	}
	return x.Pkg().Path() == y.Pkg().Path()
}
