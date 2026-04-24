// Package model defines the canonical, language-agnostic data types shared
// between all API collectors and formatters.
// This is the only module both sides of the pipeline need to import for types.
package model

// ApiEndpoint is the canonical, language-agnostic model for a single API endpoint.
type ApiEndpoint struct {
	Name        string         `json:"name"`
	Folder      string         `json:"folder,omitempty"`
	Description string         `json:"description,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
	Path        string         `json:"path"`
	Method      string         `json:"method,omitempty"` // empty for non-HTTP protocols
	Protocol    string         `json:"protocol"`         // "http", "grpc", "websocket", etc.
	Parameters  []ApiParameter `json:"parameters,omitempty"`
	Headers     []ApiHeader    `json:"headers,omitempty"`
	RequestBody *ApiBody       `json:"requestBody,omitempty"`
	Response    *ApiBody       `json:"response,omitempty"`
	// Metadata holds protocol-specific extensions without polluting the core struct.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// ApiParameter describes a single input parameter of an endpoint.
type ApiParameter struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"` // "text" | "file"
	Required    bool     `json:"required"`
	In          string   `json:"in"` // "query" | "path" | "header" | "cookie" | "body" | "form"
	Default     string   `json:"default,omitempty"`
	Description string   `json:"description,omitempty"`
	Example     string   `json:"example,omitempty"`
	Enum        []string `json:"enum,omitempty"`
}

// ApiHeader describes an HTTP header associated with an endpoint.
type ApiHeader struct {
	Name        string `json:"name"`
	Value       string `json:"value,omitempty"`
	Description string `json:"description,omitempty"`
	Example     string `json:"example,omitempty"`
	Required    bool   `json:"required"`
}

// ObjectModelKind represents the kind of type model.
type ObjectModelKind string

const (
	KindSingle ObjectModelKind = "single"
	KindObject ObjectModelKind = "object"
	KindArray  ObjectModelKind = "array"
	KindMap    ObjectModelKind = "map"
	KindRef    ObjectModelKind = "ref"
)

const (
	JsonTypeString  = "string"
	JsonTypeInt     = "int"
	JsonTypeLong    = "long"
	JsonTypeFloat   = "float"
	JsonTypeDouble  = "double"
	JsonTypeBoolean = "boolean"
	JsonTypeNull    = "null"
)

type ObjectModel struct {
	Kind       ObjectModelKind        `json:"kind"`
	TypeName   string                 `json:"typeName"`
	Fields     map[string]*FieldModel `json:"fields,omitempty"`
	Items      *ObjectModel           `json:"items,omitempty"`
	KeyModel   *ObjectModel           `json:"keyModel,omitempty"`
	ValueModel *ObjectModel           `json:"valueModel,omitempty"`
}

type FieldModel struct {
	Model        *ObjectModel   `json:"model"`
	Comment      string         `json:"comment,omitempty"`
	Required     bool           `json:"required"`
	DefaultValue string         `json:"defaultValue,omitempty"`
	Options      []FieldOption  `json:"options,omitempty"`
	Demo         string         `json:"demo,omitempty"`
	Advanced     map[string]any `json:"advanced,omitempty"`
	Generic      bool           `json:"generic,omitempty"`
}

type FieldOption struct {
	Value any    `json:"value"`
	Desc  string `json:"desc,omitempty"`
}

func SingleModel(typeName string) *ObjectModel {
	return &ObjectModel{Kind: KindSingle, TypeName: typeName}
}

func ObjectModelFrom(fields map[string]*FieldModel) *ObjectModel {
	return &ObjectModel{Kind: KindObject, TypeName: "object", Fields: fields}
}

func ArrayModel(item *ObjectModel) *ObjectModel {
	return &ObjectModel{Kind: KindArray, TypeName: "array", Items: item}
}

func MapModel(key, value *ObjectModel) *ObjectModel {
	return &ObjectModel{Kind: KindMap, TypeName: "map", KeyModel: key, ValueModel: value}
}

func RefModel(typeName string) *ObjectModel {
	return &ObjectModel{Kind: KindRef, TypeName: typeName}
}

func NullModel() *ObjectModel {
	return SingleModel(JsonTypeNull)
}

func EmptyObject() *ObjectModel {
	return &ObjectModel{Kind: KindObject, TypeName: "object", Fields: map[string]*FieldModel{}}
}

func (m *ObjectModel) IsSingle() bool { return m != nil && m.Kind == KindSingle }
func (m *ObjectModel) IsObject() bool { return m != nil && m.Kind == KindObject }
func (m *ObjectModel) IsArray() bool  { return m != nil && m.Kind == KindArray }
func (m *ObjectModel) IsMap() bool    { return m != nil && m.Kind == KindMap }
func (m *ObjectModel) IsRef() bool    { return m != nil && m.Kind == KindRef }

type FieldOpt func(*FieldModel)

func WithComment(c string) FieldOpt {
	return func(f *FieldModel) { f.Comment = c }
}

func WithRequired(r bool) FieldOpt {
	return func(f *FieldModel) { f.Required = r }
}

func WithDefault(d string) FieldOpt {
	return func(f *FieldModel) { f.DefaultValue = d }
}

func WithDemo(d string) FieldOpt {
	return func(f *FieldModel) { f.Demo = d }
}

func WithGeneric(g bool) FieldOpt {
	return func(f *FieldModel) { f.Generic = g }
}

type ObjectModelBuilder struct {
	fields map[string]*FieldModel
}

func NewObjectModelBuilder() *ObjectModelBuilder {
	return &ObjectModelBuilder{fields: make(map[string]*FieldModel)}
}

func (b *ObjectModelBuilder) Field(name string, model *ObjectModel, opts ...FieldOpt) *ObjectModelBuilder {
	fm := &FieldModel{Model: model}
	for _, opt := range opts {
		opt(fm)
	}
	b.fields[name] = fm
	return b
}

func (b *ObjectModelBuilder) StringField(name string, opts ...FieldOpt) *ObjectModelBuilder {
	return b.Field(name, SingleModel(JsonTypeString), opts...)
}

func (b *ObjectModelBuilder) IntField(name string, opts ...FieldOpt) *ObjectModelBuilder {
	return b.Field(name, SingleModel(JsonTypeInt), opts...)
}

func (b *ObjectModelBuilder) LongField(name string, opts ...FieldOpt) *ObjectModelBuilder {
	return b.Field(name, SingleModel(JsonTypeLong), opts...)
}

func (b *ObjectModelBuilder) FloatField(name string, opts ...FieldOpt) *ObjectModelBuilder {
	return b.Field(name, SingleModel(JsonTypeFloat), opts...)
}

func (b *ObjectModelBuilder) DoubleField(name string, opts ...FieldOpt) *ObjectModelBuilder {
	return b.Field(name, SingleModel(JsonTypeDouble), opts...)
}

func (b *ObjectModelBuilder) BoolField(name string, opts ...FieldOpt) *ObjectModelBuilder {
	return b.Field(name, SingleModel(JsonTypeBoolean), opts...)
}

func (b *ObjectModelBuilder) ArrayField(name string, itemType *ObjectModel, opts ...FieldOpt) *ObjectModelBuilder {
	return b.Field(name, ArrayModel(itemType), opts...)
}

func (b *ObjectModelBuilder) ObjectField(name string, obj *ObjectModel, opts ...FieldOpt) *ObjectModelBuilder {
	return b.Field(name, obj, opts...)
}

func (b *ObjectModelBuilder) MapField(name string, key, value *ObjectModel, opts ...FieldOpt) *ObjectModelBuilder {
	return b.Field(name, MapModel(key, value), opts...)
}

func (b *ObjectModelBuilder) Build() *ObjectModel {
	fields := make(map[string]*FieldModel, len(b.fields))
	for k, v := range b.fields {
		fields[k] = v
	}
	return &ObjectModel{Kind: KindObject, TypeName: "object", Fields: fields}
}

// ApiBody describes the request or response body of an endpoint.
type ApiBody struct {
	MediaType string       `json:"mediaType,omitempty"`
	Body      *ObjectModel `json:"body,omitempty"`
	Example   any          `json:"example,omitempty"`
}
