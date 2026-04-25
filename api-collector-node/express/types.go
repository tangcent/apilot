package express

type TSInterface struct {
	Name           string
	Fields         []TSField
	TypeParameters []string
	Comment        string
}

type TSField struct {
	Name        string
	Type        string
	Comment     string
	Required    bool
	Annotations []string
}

type TSTypeAlias struct {
	Name           string
	TypeDef        string
	TypeParameters []string
	Comment        string
}

type TSEnum struct {
	Name    string
	Members []TSEnumMember
	Comment string
}

type TSEnumMember struct {
	Name    string
	Value   string
	Comment string
}

type TSTypeRegistry struct {
	Interfaces  map[string]*TSInterface
	TypeAliases map[string]*TSTypeAlias
	Enums       map[string]*TSEnum
}

func NewTSTypeRegistry() *TSTypeRegistry {
	return &TSTypeRegistry{
		Interfaces:  make(map[string]*TSInterface),
		TypeAliases: make(map[string]*TSTypeAlias),
		Enums:       make(map[string]*TSEnum),
	}
}

func (r *TSTypeRegistry) Merge(other *TSTypeRegistry) {
	for k, v := range other.Interfaces {
		r.Interfaces[k] = v
	}
	for k, v := range other.TypeAliases {
		r.TypeAliases[k] = v
	}
	for k, v := range other.Enums {
		r.Enums[k] = v
	}
}

type ExpressHandlerInfo struct {
	ReqBodyType string
	ResBodyType string
	QueryType   string
	ParamsType  string
	ReqBodyIn   string
	ResBodyIn   string
}
