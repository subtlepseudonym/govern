package govern

type Package struct {
	Name         string      `json:"name"`
	Dependencies []Import    `json:"dependencies"`
	Constants    []Field     `json:"constants"`
	Variables    []Field     `json:"variables"`
	Structs      []Struct    `json:"structs"`
	Interfaces   []Interface `json:"interfaces"`
	Functions    []Function  `json:"functions"`
}

func (p *Package) GetName() string {
	return p.Name
}

type Import struct {
	Path       string `json:"path"`
	ImportedAs string `json:"importedAs"`
}

type Field struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Exported bool   `json:"exported"`
}

func (f Field) GetName() string {
	return f.Name
}

func (f Field) IsExported() bool {
	return f.Exported
}

type Struct struct {
	Name     string  `json:"name"`
	Fields   []Field `json:"fields"`
	Exported bool    `json:"exported"`
}

func (s Struct) GetName() string {
	return s.Name
}

func (s Struct) IsExported() bool {
	return s.Exported
}

type Interface struct {
	Name     string     `json:"name"`
	Methods  []Function `json"methods"`
	Exported bool
}

func (i Interface) GetName() string {
	return i.Name
}

func (i Interface) IsExported() bool {
	return i.Exported
}

type Function struct {
	Name      string  `json:"name"`
	Arguments []Field `json:"arguments"`
	Results   []Field `json:"results"`
	Exported  bool    `json:"exported"`
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
