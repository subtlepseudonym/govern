package govern

import (
	"fmt"
	"path"
	"reflect"
	"strings"

	"go/ast"
	"go/parser"
	"go/token"
)

func ParseFile(filename string) (*Package, error) {
	fset := token.NewFileSet()
	astFile, err := parser.ParseFile(fset, filename, nil, parser.Mode(0))
	if err != nil {
		return nil, err
	}

	return parseFile(astFile)
}

func parseFile(file *ast.File) (*Package, error) {
	if file == nil {
		return nil, fmt.Errorf("syntax tree file is nil")
	}

	var pkg Package
	if file.Name != nil {
		pkg.Name = file.Name.Name
	}

	// Parse package imports
	if len(file.Imports) > 0 {
		for _, imp := range file.Imports {
			if imp == nil || imp.Path == nil {
				continue
			}

			trimmedDep := strings.Trim(imp.Path.Value, `"`)
			dep := Import{
				Path: trimmedDep,
			}

			if imp.Name != nil {
				dep.ImportedAs = imp.Name.Name
			} else {
				// FIXME: should get correct package name rather than path base
				// this can be (hopefully) grabbed from $GOPATH/bin
				dep.ImportedAs = path.Base(trimmedDep)
			}
			pkg.Dependencies = append(pkg.Dependencies, dep)
		}
	}

	if file.Scope == nil {
		return nil, fmt.Errorf("package scope is nil")
	}

	for _, obj := range file.Scope.Objects {
		switch obj.Kind {
		case ast.Con:
			pkg.Constants = append(pkg.Constants, parseField(obj))
		case ast.Var:
			pkg.Variables = append(pkg.Variables, parseField(obj))
		case ast.Typ:
			s, i := parseType(obj)
			if s != nil {
				pkg.Structs = append(pkg.Structs, *s)
			}
			if i != nil {
				pkg.Interfaces = append(pkg.Interfaces, *i)
			}
		case ast.Fun:
			pkg.Functions = append(pkg.Functions, parseFunction(obj))
		default:
			continue
		}
	}

	return &pkg, nil
}

func parseField(obj *ast.Object) Field {
	f := Field{
		Name:     obj.Name,
		Exported: token.IsExported(obj.Name),
	}

	if val, ok := obj.Decl.(*ast.ValueSpec); ok {
		if len(val.Values) != 0 {
			if lit, ok := val.Values[0].(*ast.BasicLit); ok {
				f.Type = strings.ToLower(lit.Kind.String())
			}
		}
	}

	return f
}

func parseType(obj *ast.Object) (*Struct, *Interface) {
	typ, ok := obj.Decl.(*ast.TypeSpec)
	if !ok {
		return nil, nil
	}

	var s *Struct
	var i *Interface

	switch val := typ.Type.(type) {
	case *ast.StructType:
		s = &Struct{
			Name:     obj.Name,
			Exported: token.IsExported(obj.Name),
		}

		if val.Fields == nil {
			break
		}
		for _, f := range val.Fields.List {
			var field Field
			if len(f.Names) != 0 && f.Names[0] != nil {
				field.Name = f.Names[0].Name
				field.Exported = token.IsExported(f.Names[0].Name)
			}
			if ident, ok := f.Type.(*ast.Ident); ok && ident != nil {
				field.Type = ident.Name
			}

			s.Fields = append(s.Fields, field)
		}
	case *ast.InterfaceType:
		i = &Interface{
			Name:     obj.Name,
			Exported: token.IsExported(obj.Name),
		}
	default:
		fmt.Println(reflect.TypeOf(obj.Decl))
	}

	return s, i
}

func parseFunction(obj *ast.Object) Function {
	fn, ok := obj.Decl.(*ast.FuncDecl)
	if !ok {
		fmt.Println(reflect.TypeOf(obj.Decl))
	}

	var function Function
	if fn.Name != nil {
		function.Name = fn.Name.Name
		function.Exported = token.IsExported(fn.Name.Name)
	}

	if fn.Type == nil || fn.Type.Params == nil {
		return function
	}

	for _, arg := range fn.Type.Params.List {
		var field Field
		if len(arg.Names) != 0 && arg.Names[0] != nil {
			field.Name = arg.Names[0].Name
			field.Exported = token.IsExported(arg.Names[0].Name)
		}
		if ident, ok := arg.Type.(*ast.Ident); ok && ident != nil {
			field.Type = ident.Name
		}

		function.Arguments = append(function.Arguments, field)
	}

	if fn.Type.Results == nil {
		return function
	}

	for _, arg := range fn.Type.Results.List {
		var field Field
		if len(arg.Names) != 0 && arg.Names[0] != nil {
			field.Name = arg.Names[0].Name
			field.Exported = token.IsExported(arg.Names[0].Name)
		}
		if ident, ok := arg.Type.(*ast.Ident); ok && ident != nil {
			field.Type = ident.Name
		}

		function.Results = append(function.Results, field)
	}

	return function
}
