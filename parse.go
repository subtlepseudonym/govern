package govern

import (
	"fmt"
	"path"
	"strings"

	"go/ast"
	"go/parser"
	"go/token"

	"github.com/subtlepseudonym/govern/multierror"
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

	if file.Scope == nil || file.Scope.Objects == nil {
		return &pkg, fmt.Errorf("package scope is nil")
	}

	var errs multierror.MultiError

	for name, obj := range file.Scope.Objects {
		if obj == nil {
			errs = append(errs, fmt.Errorf("object %q is nil", name))
			continue
		}

		switch obj.Kind {
		case ast.Con:
			field, err := parseField(obj)
			if err != nil {
				errs = append(errs, fmt.Errorf("parse field %q: %w", name, err))
				continue
			}
			pkg.Constants = append(pkg.Constants, *field)
		case ast.Var:
			field, err := parseField(obj)
			if err != nil {
				errs = append(errs, fmt.Errorf("parse field %q: %w", name, err))
				continue
			}
			pkg.Variables = append(pkg.Variables, *field)
		case ast.Typ:
			s, i, err := parseType(obj)
			if err != nil {
				errs = append(errs, fmt.Errorf("parse type %q: %w", name, err))
				continue
			}

			if s != nil {
				pkg.Structs = append(pkg.Structs, *s)
			}
			if i != nil {
				pkg.Interfaces = append(pkg.Interfaces, *i)
			}
		case ast.Fun:
			if obj.Decl == nil {
				errs = append(errs, fmt.Errorf("type object declaration %q is nil", name))
				continue
			}

			funcDecl, ok := obj.Decl.(*ast.FuncDecl)
			if !ok {
				errs = append(errs, fmt.Errorf("type object declaration %q is not FuncDecl", name))
				continue
			}

			function, err := parseFunction(funcDecl)
			if err != nil {
				errs = append(errs, fmt.Errorf("parse function %q: %q", name, err))
				continue
			}
			pkg.Functions = append(pkg.Functions, function)
		default:
			continue
		}
	}

	return &pkg, errs.ErrOrNil()
}

func parseField(obj *ast.Object) (*Field, error) {
	if obj == nil {
		return nil, fmt.Errorf("field object is nil")
	}

	field := &Field{
		Name:     obj.Name,
		Exported: token.IsExported(obj.Name),
	}

	if obj.Decl == nil {
		return field, nil // TODO: is there actually a case where field is typeless?
	}

	if val, ok := obj.Decl.(*ast.ValueSpec); ok {
		if len(val.Values) > 0 {
			if lit, ok := val.Values[0].(*ast.BasicLit); ok && lit != nil {
				field.Type = strings.ToLower(lit.Kind.String())
			}
		}
	}

	return field, nil
}

func parseType(obj *ast.Object) (*Struct, *Interface, error) {
	if obj == nil {
		return nil, nil, fmt.Errorf("type object is nil")
	}

	typ, ok := obj.Decl.(*ast.TypeSpec)
	if !ok {
		return nil, nil, fmt.Errorf("type object is not TypeSpec")
	}
	if typ == nil {
		// TODO: does this actually happen?
		return nil, nil, fmt.Errorf("type object declaration is nil")
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
			if f == nil {
				continue
			}

			var field Field
			if len(f.Names) > 0 && f.Names[0] != nil {
				field.Name = f.Names[0].Name
				field.Exported = token.IsExported(f.Names[0].Name)
			}
			if ident, ok := f.Type.(*ast.Ident); ok && ident != nil {
				field.Type = ident.Name
			}
			// TODO: should keep track of tags too

			s.Fields = append(s.Fields, field)
		}
	case *ast.InterfaceType:
		i = &Interface{
			Name:     obj.Name,
			Exported: token.IsExported(obj.Name),
		}

		if val.Methods == nil {
			break
		}

		for _, method := range val.Methods.List {
			if method == nil {
				continue
			}

			funcDecl := &ast.FuncDecl{
				Name: method.Names[0],
			}
			if fn, ok := method.Type.(*ast.FuncType); ok {
				funcDecl.Type = fn
			}

			function, err := parseFunction(funcDecl)
			if err != nil {
				return nil, i, fmt.Errorf("parse function: %w", err)
			}

			i.Methods = append(i.Methods, function)
		}
	}

	return s, i, nil
}

func parseFunction(funcDecl *ast.FuncDecl) (Function, error) {
	var function Function
	if funcDecl.Name != nil {
		function.Name = funcDecl.Name.Name
		function.Exported = token.IsExported(funcDecl.Name.Name)
	}

	if funcDecl.Type == nil {
		return function, fmt.Errorf("function declaration type is nil")
	}

	if funcDecl.Type.Params != nil {
		for _, arg := range funcDecl.Type.Params.List {
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
	}

	if funcDecl.Type.Results != nil {
		for _, arg := range funcDecl.Type.Results.List {
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
	}

	return function, nil
}
