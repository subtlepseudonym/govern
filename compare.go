package govern

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
)

// ParsePackage reads a directory and parses the go package defined there. It is an
// error to define multiple packages in a single directory
func ParsePackage(dir string) (*types.Package, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parse directory: %v", err)
	}

	var packages []string
	for key := range pkgs {
		packages = append(packages, key)
	}
	if len(packages) > 1 {
		return nil, fmt.Errorf("multiple packages: %v", packages)
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

// ComparePackages determines whether a package has undergone a major or minor change
//
// The assumption here is that the packages are different, so that if both major and
// minor return false, the difference constitutes a patch change
func ComparePackages(older, newer *types.Package) (major, minor bool) {
	if older.Name() != newer.Name() {
		major = true
	}

	scopeMajor, scopeMinor, _ := compareScopes(older.Scope(), newer.Scope(), false)
	return major || scopeMajor, scopeMinor
}

// ExplainPackageChange identifies what has changed in a package that constitutes more
// than a patch change
func ExplainPackageChange(older, newer *types.Package) error {
	if older.Name() != newer.Name() {
		return MultiError([]error{
			fmt.Errorf("package renamed: %q != %q", older.Name(), newer.Name()),
		})
	}

	major, minor, err := compareScopes(older.Scope(), newer.Scope(), true)
	if major {
		return fmt.Errorf("major: %v", err)
	} else if minor {
		return fmt.Errorf("minor: %v", err)
	} else {
		return fmt.Errorf("patch: %v", err)
	}
}

func compareScopes(older, newer *types.Scope, explain bool) (major, minor bool, errs MultiError) {
	oldObjs := make(map[string]types.Object, len(older.Names()))
	for _, name := range older.Names() {
		oldObjs[name] = older.Lookup(name)
	}

	newObjs := make(map[string]types.Object, len(newer.Names()))
	for _, name := range newer.Names() {
		newObjs[name] = newer.Lookup(name)
	}

	var merr MultiError
	for name, oldObj := range oldObjs {
		delete(oldObjs, name)
		if !oldObj.Exported() {
			continue
		}

		newObj, exists := newObjs[name]
		if !exists || !newObj.Exported() {
			major = true
			merr = append(merr, fmt.Errorf("%q removed", name))
		} else {
			mj, mi, err := compareTypes(oldObj.Type(), newObj.Type(), nil)
			if err != nil {
				merr = append(merr, fmt.Errorf("%q changed: %v", name, err))
			}

			major = mj || major
			minor = mi || minor
		}

		delete(newObjs, name)
	}

	for name, obj := range newObjs {
		if obj.Exported() {
			minor = true
			merr = append(merr, fmt.Errorf("%q added", name))
		}
	}

	return major, minor, merr
}

// typePair is copied directly from go/types, except for a minor name change
// This struct is used for preventing endless recusion in identical()
type typePair struct {
	x, y types.Type
	prev *typePair
}

// identical is copied directly from go/types
func (p *typePair) identical(q *typePair) bool {
	return p.x == q.x && p.y == q.y || p.x == q.y && p.y == q.x
}

func compareTypes(x, y types.Type, prevPair *typePair) (major, minor bool, err error) {
	if x == y {
		return false, false, nil
	}

	switch x := x.(type) {
	case *types.Basic:
		if y, ok := y.(*types.Basic); ok {
			kindEqual := x.Kind() == y.Kind()
			if kindEqual {
				return false, false, nil
			}
			return true, false, fmt.Errorf("%q.Kind() != %q.Kind(): %q != %q", x.Name(), y.Name(), x.Kind(), y.Kind())
		}
		return true, false, fmt.Errorf("%T != *types.Basic", y)

	case *types.Array:
		if y, ok := y.(*types.Array); ok {
			if !(x.Len() < 0 || y.Len() < 0 || x.Len() == y.Len()) {
				return true, false, fmt.Errorf("array len mismatch: %d != %d", x.Len(), y.Len())
			}

			major, minor, err := compareTypes(x.Elem(), y.Elem(), prevPair)
			if err != nil {
				return major, minor, fmt.Errorf("%q.Elem() != %q.Elem(): %w", x, y, err)
			}
			return false, false, nil
		}
		return true, false, fmt.Errorf("%T != *types.Array", y)

	case *types.Slice:
		if y, ok := y.(*types.Slice); ok {
			major, minor, err := compareTypes(x.Elem(), y.Elem(), prevPair)
			if err != nil {
				return major, minor, fmt.Errorf("%q.Elem() != %q.Elem(): %w", x, y, err)
			}
			return major, minor, nil
		}
		return true, false, fmt.Errorf("%T != *types.Slice", y)

	case *types.Struct:
		if y, ok := y.(*types.Struct); ok {
			return compareStructs(x, y, prevPair)
		}
		return true, false, fmt.Errorf("%T != *types.Struct", y)

	case *types.Pointer:
		if y, ok := y.(*types.Pointer); ok {
			major, minor, err := compareTypes(x.Elem(), y.Elem(), prevPair)
			if err != nil {
				return major, minor, fmt.Errorf("%q.Elem() != %q.Elem(): %w", x, y, err)
			}
			return major, minor, nil
		}
		return true, false, fmt.Errorf("%T != *types.Pointer", y)

	case *types.Tuple:
		if y, ok := y.(*types.Tuple); ok {
			if x.Len() == y.Len() {
				if x != nil {
					for i := 0; i < x.Len(); i++ {
						xField := x.At(i)
						yField := x.At(i)

						major, minor, err := compareTypes(xField.Type(), yField.Type(), prevPair)
						if err != nil {
							return major, minor, fmt.Errorf("%q.Type() != %q.Type(): %w", xField.Name(), yField.Name(), err)
						}
					}
				}
				return false, false, nil
			}
			return true, false, fmt.Errorf("%q.Len() != %q.Len()", x, y)
		}
		return true, false, fmt.Errorf("%T != *types.Tuple", y)

	case *types.Signature:
		if y, ok := y.(*types.Signature); ok {
			if x.Variadic() != y.Variadic() {
				return true, false, fmt.Errorf("%q.Variadic() != %q.Variadic(): %t != %t", x, y, x.Variadic(), y.Variadic())
			}

			major, minor, err := compareTypes(x.Params(), y.Params(), prevPair)
			if err != nil {
				return major, minor, fmt.Errorf("%q.Params() != %q.Params(): %w", x, y, err)
			}

			major, minor, err = compareTypes(x.Results(), y.Results(), prevPair)
			if err != nil {
				return major, minor, fmt.Errorf("%q.Results != %q.Results(): %w", x, y, err)
			}

			return false, false, nil
		}
		return true, false, fmt.Errorf("%T != *types.Signature", y)

	case *types.Interface:
		if y, ok := y.(*types.Interface); ok {
			return compareInterfaces(x, y, prevPair)
		}
		return true, false, fmt.Errorf("%T != *types.Interface", y)

	case *types.Map:
		if y, ok := y.(*types.Map); ok {
			major, minor, err := compareTypes(x.Key(), y.Key(), prevPair)
			if err != nil {
				return major, minor, fmt.Errorf("%q.Key() != %q.Key(): %w", x, y, err)
			}

			major, minor, err = compareTypes(x.Elem(), y.Elem(), prevPair)
			if err != nil {
				return major, minor, fmt.Errorf("%q.Elem() != %q.Elem(): %w", x, y, err)
			}

			return false, false, nil
		}
		return true, false, fmt.Errorf("%T != *types.Map", y)

	case *types.Chan:
		if y, ok := y.(*types.Chan); ok {
			if x.Dir() != y.Dir() {
				return true, false, fmt.Errorf("%q.Dir() != %q.Dir(): %d != %d", x, y, x.Dir(), y.Dir())
			}

			major, minor, err := compareTypes(x.Elem(), y.Elem(), prevPair)
			if err != nil {
				return major, minor, fmt.Errorf("%q.Elem() != %q.Elem(): %w", x, y, err)
			}
			return false, false, nil
		}
		return true, false, fmt.Errorf("%T != *types.Chan", y)

	case *types.Named:
		if y, ok := y.(*types.Named); ok {
			return compareTypes(x.Underlying(), y.Underlying(), prevPair)
		}
		return true, false, fmt.Errorf("%T != *types.Named", y)

	case nil:
		return true, false, fmt.Errorf("x is nil")
	default:
		panic("unreachable")
	}

	return true, false, fmt.Errorf("fallthrough error")
}

func compareStructs(x, y *types.Struct, prevPair *typePair) (major, minor bool, err error) {
	newPair := &typePair{
		x:    types.Type(x),
		y:    types.Type(y),
		prev: prevPair,
	}
	for prevPair != nil {
		if prevPair.identical(newPair) {
			return false, false, nil // same pair was compared before
		}
		prevPair = prevPair.prev
	}

	xFields := make(map[string]*types.Var, x.NumFields())
	xTags := make(map[string]string, x.NumFields())
	for i := 0; i < x.NumFields(); i++ {
		name := x.Field(i).Name()
		if !token.IsExported(name) {
			continue
		}

		xFields[name] = x.Field(i)
		xTags[name] = x.Tag(i)
	}

	yFields := make(map[string]*types.Var, y.NumFields())
	yTags := make(map[string]string, y.NumFields())
	for i := 0; i < y.NumFields(); i++ {
		name := y.Field(i).Name()
		if !token.IsExported(name) {
			continue
		}

		yFields[name] = y.Field(i)
		yTags[name] = y.Tag(i)
	}

	// check for old fields in new struct
	for name, xField := range xFields {
		yField, ok := yFields[name]
		if !ok {
			return true, false, fmt.Errorf("field missing: %q", name)
		}

		if xField.Embedded() != yField.Embedded() {
			return true, false, fmt.Errorf("field embedded property mismatch: %q.Embedded() = %t != %q.Embedded() = %t", xField.Name(), xField.Embedded(), yField.Name(), yField.Embedded())
		}
		if xTags[name] != yTags[name] {
			return true, false, fmt.Errorf("field tag mismatch: %q != %q", xTags[name], yTags[name])
		}

		major, minor, err := compareTypes(xField.Type(), yField.Type(), newPair)
		if err != nil {
			return major, minor, fmt.Errorf("field type mismatch: %q.Type() != %q.Type(): %w", xField.Name(), yField.Name(), err)
		}
	}

	// check for new fields in old struct
	minor, major, err = compareStructs(y, x, newPair)
	if err != nil {
		return major, minor, fmt.Errorf("field added: %w", err)
	}
	return false, false, nil
}

func compareInterfaces(x, y *types.Interface, prevPair *typePair) (major, minor bool, err error) {
	if x.NumMethods() != y.NumMethods() {
		return true, false, fmt.Errorf("%q.NumMethods() != %q.NumMethods()", x, y)
	}

	newPair := &typePair{
		x:    types.Type(x),
		y:    types.Type(y),
		prev: prevPair,
	}
	for prevPair != nil {
		if prevPair.identical(newPair) {
			return false, false, nil // same pair was compared before
		}
		prevPair = prevPair.prev
	}

	xMethods := make(map[string]*types.Func, x.NumMethods())
	yMethods := make(map[string]*types.Func, y.NumMethods())
	for i := 0; i < x.NumMethods(); i++ {
		xMethods[x.Method(i).Id()] = x.Method(i)
		yMethods[y.Method(i).Id()] = y.Method(i)
	}

	for id, xMethod := range xMethods {
		yMethod, ok := yMethods[id]
		if !ok {
			return true, false, fmt.Errorf("method missing: %q", id)
		}

		major, minor, err := compareTypes(xMethod.Type(), yMethod.Type(), newPair)
		if err != nil {
			return major, minor, fmt.Errorf("method type mismatch: %q.Type() != %q.Type(): %q != %q", xMethod.Name(), yMethod.Name(), xMethod.Type(), yMethod.Type())
		}
	}

	minor, major, err = compareInterfaces(y, x, newPair)
	if err != nil {
		return major, minor, fmt.Errorf("method added: %w", err)
	}
	return false, false, nil
}
