package main

import (
	"encoding/json"
	"reflect"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

type Package struct {
	Name string
	Constants []Field
	Variables []Field
	Structs []Struct
	Interfaces []Interface
	Functions []Function
}

func (p *Package) GetName() string {
	return p.Name
}

type Field struct {
	Name string
	Type string
	Exported bool
}

func (f Field) GetName() string {
	return f.Name
}

func (f Field) IsExported() bool {
	return f.Exported
}

type Struct struct {
	Name string
	Fields []Field
	Exported bool
}

func (s Struct) GetName() string {
	return s.Name
}

func (s Struct) IsExported() bool {
	return s.Exported
}

type Interface struct {
	Name string
	Methods []Function
	Exported bool
}

func (i Interface) GetName() string {
	return i.Name
}

func (i Interface) IsExported() bool {
	return i.Exported
}

type Function struct {
	Name string
	Arguments []Field
	Results []Field
	Exported bool
}

func (f Function) GetName() string {
	return f.Name
}

func (f Function) IsExported() bool {
	return f.Exported
}

type Named interface {
	GetName() string
}

func GetName(obj Named) string {
	return obj.GetName()
}

type Exported interface {
	IsExported() bool
}

func main() {
	fset := token.NewFileSet()
	src := `package test

import (
	"fmt"
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

func main() {
	t := Object{
		Internal: "test",
		unexport: "neat",
	}
	t.Print(t.Internal)

	i, err := testFunc(t.Internal)
	if err != nil {
		panic(err)
	}
	fmt.Println(i)
}

func testFunc(stringArg string) (int, error) {
	return 0, nil
}
`

	f, err := parser.ParseFile(fset, "", src, parser.Mode(0))
	if err != nil {
		panic(err)
	}

	pkg, err := parseFile(f)
	if err != nil {
		panic(err)
	}

	b, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))
}

func parseFile(obj *ast.File) (*Package, error) {
	if obj == nil {
		return nil, fmt.Errorf("file pointer is nil")
	}

	if obj.Name == nil {
		return nil, fmt.Errorf("file package name is nil")
	}

	if obj.Scope == nil {
		return nil, fmt.Errorf("package scope is nil")
	}

	pkg := &Package{
		Name: obj.Name.Name,
	}

	for _, obj := range obj.Scope.Objects {
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

	return pkg, nil
}

func parseField(obj *ast.Object) Field {
	f := Field{
		Name: obj.Name,
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
			Name: obj.Name,
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
			Name: obj.Name,
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
