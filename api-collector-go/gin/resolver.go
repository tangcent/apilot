// Package gin provides Go struct parsing and type resolution for Gin endpoints.
package gin

import (
	"go/ast"
	"go/token"
	"reflect"
	"strings"

	model "github.com/tangcent/apilot/api-model"
)

// StructDef represents a parsed Go struct definition.
type StructDef struct {
	Name          string
	Fields        []StructField
	EmbeddedTypes []string
}

// StructField represents a single field in a Go struct.
type StructField struct {
	Name       string
	Type       string
	JsonTag    string
	BindingTag string
	Comment    string
}

// extractStructs walks the AST and extracts all struct type definitions.
func extractStructs(f *ast.File) map[string]StructDef {
	structs := make(map[string]StructDef)

	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			structDef := StructDef{
				Name:   typeSpec.Name.Name,
				Fields: []StructField{},
			}

			if structType.Fields != nil {
				for _, field := range structType.Fields.List {
					// Embedded field (no name)
					if len(field.Names) == 0 {
						typeName := extractTypeNameFromExpr(field.Type)
						if typeName != "" {
							structDef.EmbeddedTypes = append(structDef.EmbeddedTypes, typeName)
						}
						continue
					}

					// Regular field
					for _, name := range field.Names {
						sf := StructField{
							Name: name.Name,
							Type: extractTypeNameFromExpr(field.Type),
						}

						if field.Tag != nil {
							tag := strings.Trim(field.Tag.Value, "`")
							structTag := reflect.StructTag(tag)

							if jsonTag, ok := structTag.Lookup("json"); ok {
								parts := strings.SplitN(jsonTag, ",", 2)
								if parts[0] != "-" {
									sf.JsonTag = parts[0]
								}
							}

							if bindingTag, ok := structTag.Lookup("binding"); ok {
								sf.BindingTag = bindingTag
							}
						}

						if field.Comment != nil {
							sf.Comment = strings.TrimSpace(field.Comment.Text())
						}

						structDef.Fields = append(structDef.Fields, sf)
					}
				}
			}

			structs[structDef.Name] = structDef
		}
	}

	return structs
}

// extractTypeNameFromExpr returns a string representation of a Go type expression.
func extractTypeNameFromExpr(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return "*" + extractTypeNameFromExpr(e.X)
	case *ast.ArrayType:
		return "[]" + extractTypeNameFromExpr(e.Elt)
	case *ast.MapType:
		return "map[" + extractTypeNameFromExpr(e.Key) + "]" + extractTypeNameFromExpr(e.Value)
	case *ast.SelectorExpr:
		if x, ok := e.X.(*ast.Ident); ok {
			return x.Name + "." + e.Sel.Name
		}
		return e.Sel.Name
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.StructType:
		return "struct{}"
	}
	return ""
}

// buildVarTypeMap traces variable declarations in a function body to build
// a mapping from variable name to its declared type name.
func buildVarTypeMap(fn *ast.FuncDecl) map[string]string {
	if fn.Body == nil {
		return nil
	}

	varMap := make(map[string]string)

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch stmt := n.(type) {
		case *ast.DeclStmt:
			genDecl, ok := stmt.Decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.VAR {
				return true
			}
			for _, spec := range genDecl.Specs {
				valueSpec, ok := spec.(*ast.ValueSpec)
				if !ok || valueSpec.Type == nil {
					continue
				}
				typeName := extractTypeNameFromExpr(valueSpec.Type)
				for _, name := range valueSpec.Names {
					varMap[name.Name] = typeName
				}
			}

		case *ast.AssignStmt:
			if stmt.Tok != token.DEFINE {
				return true
			}
			for i, lhs := range stmt.Lhs {
				ident, ok := lhs.(*ast.Ident)
				if !ok || i >= len(stmt.Rhs) {
					continue
				}
				typeName := inferTypeFromExpr(stmt.Rhs[i])
				if typeName != "" {
					varMap[ident.Name] = typeName
				}
			}
		}
		return true
	})

	return varMap
}

// inferTypeFromExpr infers the type name from an expression on the RHS of :=.
func inferTypeFromExpr(expr ast.Expr) string {
	if unary, ok := expr.(*ast.UnaryExpr); ok && unary.Op == token.AND {
		inner := inferTypeFromExpr(unary.X)
		if inner != "" {
			return inner
		}
	}

	if comp, ok := expr.(*ast.CompositeLit); ok {
		return extractTypeNameFromExpr(comp.Type)
	}

	return ""
}

// TypeResolver resolves Go type names to ObjectModel schemas.
type TypeResolver struct {
	structRegistry map[string]StructDef
	resolving      map[string]bool
}

// NewTypeResolver creates a new TypeResolver with the given struct definitions.
func NewTypeResolver(structs map[string]StructDef) *TypeResolver {
	return &TypeResolver{
		structRegistry: structs,
		resolving:      make(map[string]bool),
	}
}

var goPrimitives = map[string]string{
	"string":  model.JsonTypeString,
	"bool":    model.JsonTypeBoolean,
	"int":     model.JsonTypeInt,
	"int8":    model.JsonTypeInt,
	"int16":   model.JsonTypeInt,
	"int32":   model.JsonTypeInt,
	"int64":   model.JsonTypeLong,
	"uint":    model.JsonTypeInt,
	"uint8":   model.JsonTypeInt,
	"uint16":  model.JsonTypeInt,
	"uint32":  model.JsonTypeInt,
	"uint64":  model.JsonTypeLong,
	"float32": model.JsonTypeFloat,
	"float64": model.JsonTypeDouble,
	"byte":    model.JsonTypeInt,
	"rune":    model.JsonTypeInt,
}

// Resolve resolves a Go type name to an ObjectModel.
func (r *TypeResolver) Resolve(typeName string) *model.ObjectModel {
	if typeName == "" || typeName == "interface{}" || typeName == "any" {
		return model.NullModel()
	}

	if typeName == "gin.H" {
		return model.MapModel(model.SingleModel(model.JsonTypeString), model.NullModel())
	}

	if typeName == "struct{}" {
		return model.EmptyObject()
	}

	if jsonType, ok := goPrimitives[typeName]; ok {
		return model.SingleModel(jsonType)
	}

	if strings.HasPrefix(typeName, "*") {
		return r.Resolve(typeName[1:])
	}

	if strings.HasPrefix(typeName, "[]") {
		elemType := typeName[2:]
		return model.ArrayModel(r.Resolve(elemType))
	}

	if strings.HasPrefix(typeName, "map[") {
		keyType, valueType := parseMapType(typeName)
		return model.MapModel(r.Resolve(keyType), r.Resolve(valueType))
	}

	structDef, found := r.structRegistry[typeName]
	if !found {
		return model.SingleModel(typeName)
	}

	if r.resolving[typeName] {
		return model.RefModel(typeName)
	}

	r.resolving[typeName] = true
	defer func() { delete(r.resolving, typeName) }()

	fields := make(map[string]*model.FieldModel)

	for _, f := range structDef.Fields {
		fieldName := f.Name
		if f.JsonTag != "" {
			fieldName = f.JsonTag
		} else {
			fieldName = strings.ToLower(f.Name[:1]) + f.Name[1:]
		}

		required := strings.Contains(f.BindingTag, "required")

		fields[fieldName] = &model.FieldModel{
			Model:    r.Resolve(f.Type),
			Required: required,
		}
	}

	for _, embedded := range structDef.EmbeddedTypes {
		embeddedModel := r.Resolve(strings.TrimPrefix(embedded, "*"))
		if embeddedModel != nil && embeddedModel.IsObject() {
			for k, v := range embeddedModel.Fields {
				if _, exists := fields[k]; !exists {
					fields[k] = v
				}
			}
		}
	}

	return &model.ObjectModel{
		Kind:     model.KindObject,
		TypeName: typeName,
		Fields:   fields,
	}
}

// parseMapType extracts key and value types from "map[K]V".
func parseMapType(typeName string) (string, string) {
	inner := typeName[4:] // strip "map["
	depth := 1
	for i, c := range inner {
		switch c {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return inner[:i], inner[i+1:]
			}
		}
	}
	return "string", "interface{}"
}
