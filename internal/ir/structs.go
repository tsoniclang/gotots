package ir

import (
	"go/ast"
	"go/types"
	"sort"

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
		sites:      &[]UnsupportedSite{},
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
	var typeParams []string
	if named.TypeParams() != nil && named.TypeParams().Len() > 0 {
		names, err := b.admitGenericType(named, span)
		if err != nil {
			return nil, err
		}
		typeParams = names
		b.genericTypeObj = named
	}
	structType, ok := named.Underlying().(*types.Struct)
	if !ok {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION", Construct: "non-struct named type", Span: span}
	}

	out := &Struct{
		ID:         id,
		TypeParams: typeParams,
		Name:       spec.Name.Name,
		Exported:   spec.Name.IsExported(),
		Span:       span,
	}
	for i := range structType.NumFields() {
		field := structType.Field(i)
		// An embedded field is an ordinary field named by its type's base
		// name; promotion resolves through selection index paths at every
		// use site.
		if field.Name() == "_" {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION", Construct: "blank struct field", Span: span}
		}
		fieldType, err := b.typeOf(field.Type(), span)
		if err != nil {
			return nil, err
		}
		out.Fields = append(out.Fields, Var{Name: field.Name(), Type: fieldType})
	}
	promoted, err := b.promotedDelegates(named, span)
	if err != nil {
		return nil, err
	}
	out.Promoted = promoted
	return out, nil
}

// promotedDelegates resolves every promoted method in the type's method
// set into a delegation entry through value embedded fields. A
// promotion the table cannot delegate (pointer embedding, external or
// generic declaring types) fails the declaration — an incomplete table
// would silently mis-dispatch.
func (b *builder) promotedDelegates(named *types.Named, span Span) ([]PromotedDelegate, error) {
	seen := map[string]bool{}
	var out []PromotedDelegate
	for _, methodSet := range []*types.MethodSet{
		types.NewMethodSet(types.NewPointer(named)),
		types.NewMethodSet(named),
	} {
		for i := range methodSet.Len() {
			selection := methodSet.At(i)
			path := selection.Index()
			if len(path) <= 1 {
				continue // a direct method
			}
			method := selection.Obj().(*types.Func)
			if seen[method.Name()] {
				continue
			}
			seen[method.Name()] = true
			if method.Pkg() == nil || !b.unit.Owns(method.Pkg().Path()) {
				return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION",
					Construct: "promoted method from a type outside the translated unit (" + method.Name() + ")", Span: span}
			}
			signature := method.Type().(*types.Signature)
			if signature.TypeParams() != nil || signature.RecvTypeParams() != nil {
				return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION",
					Construct: "promoted generic method (" + method.Name() + ")", Span: span}
			}
			entry := PromotedDelegate{Name: method.Name(), Pkg: method.Pkg().Path()}
			current := types.Type(named)
			for _, index := range path[:len(path)-1] {
				structType, ok := types.Unalias(current).Underlying().(*types.Struct)
				if !ok {
					return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION",
						Construct: "promotion through a non-struct embedding (" + method.Name() + ")", Span: span}
				}
				field := structType.Field(index)
				if _, isPointer := types.Unalias(field.Type()).Underlying().(*types.Pointer); isPointer {
					return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION",
						Construct: "promotion through an embedded pointer (" + method.Name() + ")", Span: span}
				}
				entry.Path = append(entry.Path, field.Name())
				current = field.Type()
			}
			recvNamed, ok := types.Unalias(current).(*types.Named)
			if !ok {
				return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION",
					Construct: "promotion through an unnamed embedding (" + method.Name() + ")", Span: span}
			}
			entry.TypeName = recvNamed.Obj().Name()
			out = append(out, entry)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// chainFieldPath lowers a promoted selection: successive field loads
// through the embedded fields named by the selection's index path.
func (b *builder) chainFieldPath(base Expr, baseType types.Type, path []int, span Span) (Expr, error) {
	current := baseType
	out := base
	for _, index := range path {
		if pointer, isPointer := types.Unalias(current).Underlying().(*types.Pointer); isPointer {
			current = pointer.Elem()
		}
		structType, ok := types.Unalias(current).Underlying().(*types.Struct)
		if !ok {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "promoted selection through " + current.String(), Span: span}
		}
		field := structType.Field(index)
		fieldType, err := b.typeOf(field.Type(), span)
		if err != nil {
			return nil, err
		}
		if out.Type().Kind != KindPointer && out.Type().Kind != KindStruct {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "promoted selection through " + out.Type().Go, Span: span}
		}
		b.use("fieldLoad")
		out = &FieldLoad{X: out, Field: field.Name(), T: fieldType}
		current = field.Type()
	}
	return out, nil
}

// chainPromotedReceiver resolves a promoted method's actual receiver by
// walking the embedded-field prefix of the selection's index path.
func (b *builder) chainPromotedReceiver(recv Expr, recvType types.Type, selection *types.Selection, span Span) (Expr, error) {
	path := selection.Index()
	if len(path) <= 1 {
		return recv, nil
	}
	return b.chainFieldPath(recv, recvType, path[:len(path)-1], span)
}

// admitGenericType verifies a generic named type under the unit's
// closed-world instantiation evidence: every recorded type argument
// resolves in the reviewed set and no argument is a value-copy carrier
// (a single class body cannot express per-instantiation copies).
func (b *builder) admitGenericType(named *types.Named, span Span) ([]string, error) {
	typeParams := named.TypeParams()
	names := make([]string, 0, typeParams.Len())
	for i := range typeParams.Len() {
		names = append(names, typeParams.At(i).Obj().Name())
	}
	for _, instance := range b.unit.GenericTypeInstances(named.Obj()) {
		for _, arg := range instance {
			resolved, err := b.typeOf(arg, span)
			if err != nil {
				return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION",
					Construct: "generic type instantiated with an unreviewed type argument (" + arg.String() + ")", Span: span}
			}
			if resolved.Kind == KindStruct || resolved.Kind == KindArray {
				return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION",
					Construct: "generic type instantiated with a value-copy carrier (copy semantics vary per instantiation)", Span: span}
			}
		}
	}
	return names, nil
}
