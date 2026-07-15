package ir

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/packages"
)

// BuildStruct converts one named struct type declaration into IR: an
// ordered typed field list generated as a class. unit is the set of
// co-translated package paths field types may reference. Embedding, tags
// with semantic weight, and non-reviewed field types fail closed.
func BuildStruct(p *packages.Package, sourceDir string, unit Scope, spec *ast.TypeSpec, id string) (*Struct, error) {
	b := &builder{
		fset:       p.Fset,
		info:       p.TypesInfo,
		pkgPath:    p.PkgPath,
		sourceDir:  sourceDir,
		unit:       unit,
		operations: map[string]bool{},
	}
	span := b.span(spec.Pos())
	object, ok := b.info.Defs[spec.Name].(*types.TypeName)
	if !ok {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION", Construct: "type without typed definition", Span: span}
	}
	named, ok := object.Type().(*types.Named)
	if !ok {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION", Construct: "alias declaration", Span: span}
	}
	if named.TypeParams() != nil && named.TypeParams().Len() > 0 {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION", Construct: "generic type", Span: span}
	}
	structType, ok := named.Underlying().(*types.Struct)
	if !ok {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION", Construct: "non-struct named type", Span: span}
	}

	out := &Struct{
		ID:       id,
		Name:     spec.Name.Name,
		Exported: spec.Name.IsExported(),
		Span:     span,
	}
	for i := range structType.NumFields() {
		field := structType.Field(i)
		if field.Embedded() {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION", Construct: "embedded field (promotion semantics)", Span: span}
		}
		if field.Name() == "_" {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION", Construct: "blank struct field", Span: span}
		}
		fieldType, err := b.typeOf(field.Type(), span)
		if err != nil {
			return nil, err
		}
		if fieldType.Kind == KindStruct {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION", Construct: "struct-valued field (value-copy semantics)", Span: span}
		}
		out.Fields = append(out.Fields, Var{Name: field.Name(), Type: fieldType})
	}
	return out, nil
}
