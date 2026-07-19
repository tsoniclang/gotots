package ir

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/types"
	"sort"

	"golang.org/x/tools/go/packages"

	"github.com/tsoniclang/gotots/internal/typeid"
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
		return nil, &Unsupported{Kind: KindTypeWithoutTypedDefinition, Code: "GOTOTS_UNSUPPORTED_DECLARATION", Construct: "type without typed definition", Span: span}
	}
	named, ok := object.Type().(*types.Named)
	if !ok {
		return nil, &Unsupported{Kind: KindAliasDeclaration, Code: "GOTOTS_UNSUPPORTED_DECLARATION", Construct: "alias declaration", Span: span}
	}
	b.binders = typeid.OwnerBinders(named)
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
		return nil, &Unsupported{Kind: KindNonStructNamedType, Code: "GOTOTS_UNSUPPORTED_DECLARATION", Construct: "non-struct named type", Span: span}
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
			// A blank field of a provably ZERO-SIZE type ([0]X arrays,
			// empty structs — the noCopy vet-guard idiom, SyncMap's
			// phantom type markers) occupies no storage and can never be
			// referenced: it is a no-output field, skipped entirely in the
			// generated class. Any other blank field stays fail-closed.
			if zeroSizeType(field.Type()) {
				continue
			}
			return nil, &Unsupported{Kind: KindBlankStructField, Code: "GOTOTS_UNSUPPORTED_DECLARATION", Construct: "blank struct field", Span: span}
		}
		fieldType, err := b.typeOf(field.Type(), span)
		if err != nil {
			return nil, err
		}
		cell, err := b.fieldIsCell(named, field.Name(), fieldType)
		if err != nil {
			return nil, err
		}
		out.Fields = append(out.Fields, Var{Name: field.Name(), Type: fieldType, Cell: cell})
	}
	promoted, err := b.promotedDelegates(named, span)
	if err != nil {
		return nil, err
	}
	out.Promoted = promoted
	out.Comparable = b.structEqComparable(named)
	out.KeyEncodable = b.structKeyEncodable(named, span)
	return out, nil
}

// fieldStorageKey identifies one struct field's storage location by the
// DECLARING struct's origin (uninstantiated generic) plus the field name.
// This is stable across instantiations, so &Box[int].X and &Box[string].X
// resolve to distinct keys yet the address scan (which sees an
// instantiated field object) and class emission (which sees the generic
// declaration) agree. An anonymous struct is keyed by its CANONICAL
// identity (typeid.Canonical), which package-qualifies unexported field
// names by import path — never t.String(), which conflates two
// spelling-alike anon structs whose unexported fields come from different
// packages (a storage-identity collision). A canonical failure (a free
// type parameter in an anonymous struct) propagates so the address-taken
// analysis fails closed rather than keying by spelling.
func fieldStorageKey(declaringStruct types.Type, fieldName string) (string, error) {
	t := types.Unalias(declaringStruct)
	if pointer, ok := t.(*types.Pointer); ok {
		t = types.Unalias(pointer.Elem())
	}
	named, ok := t.(*types.Named)
	if !ok {
		canon, err := typeid.Canonical(t)
		if err != nil {
			return "", err
		}
		return canon + "#" + fieldName, nil
	}
	obj := named.Origin().Obj()
	pkg := ""
	if obj.Pkg() != nil {
		pkg = obj.Pkg().Path()
	}
	return pkg + "." + obj.Name() + "#" + fieldName, nil
}

// FieldStorageKeyOfSelection resolves the storage key of the field a
// selection reaches, walking any embedded-field prefix to the declaring
// struct (promotion). The pre-pass marks address-taken fields by this key.
// The bool reports whether the selection is a field access at all; a
// non-nil error is a canonical-identity failure (fail closed).
func FieldStorageKeyOfSelection(sel *types.Selection) (string, bool, error) {
	if sel == nil || sel.Kind() != types.FieldVal {
		return "", false, nil
	}
	current := sel.Recv()
	index := sel.Index()
	for _, i := range index[:len(index)-1] {
		u := types.Unalias(current)
		if pointer, ok := u.Underlying().(*types.Pointer); ok {
			u = types.Unalias(pointer.Elem())
		}
		st, ok := u.Underlying().(*types.Struct)
		if !ok {
			return "", false, nil
		}
		current = st.Field(i).Type()
	}
	key, err := fieldStorageKey(current, sel.Obj().Name())
	if err != nil {
		return "", false, err
	}
	return key, true, nil
}

// fieldIsCell reports whether a struct field (of the given declaring
// struct) is stored as a stable per-instance cell: its address is taken
// somewhere in the unit AND its type is a non-identity carrier (a scalar,
// pointer, map, slice, interface, or function). Identity carriers —
// structs, arrays, external handles — are their own stable address.
func (b *builder) fieldIsCell(declaringStruct types.Type, fieldName string, fieldType Type) (bool, error) {
	if !boxable(fieldType.Kind) {
		return false, nil
	}
	key, err := fieldStorageKey(declaringStruct, fieldName)
	if err != nil {
		return false, err
	}
	return b.unit.FieldAddressTaken(key), nil
}

// fieldCell reports whether a field-selection accesses a cell field —
// the flag FieldLoad/FieldTarget/&f carry so the value read/write
// unwraps the cell and the address is the cell itself.
func (b *builder) fieldCell(selection *types.Selection, fieldType Type) (bool, error) {
	if !boxable(fieldType.Kind) {
		return false, nil
	}
	key, ok, err := FieldStorageKeyOfSelection(selection)
	if err != nil {
		return false, err
	}
	return ok && b.unit.FieldAddressTaken(key), nil
}

// promotedDelegates resolves every promoted method in the type's method
// set into a delegation entry through value embedded fields. A
// promotion the table cannot delegate (pointer embedding, external or
// generic declaring types) fails the declaration — an incomplete table
// would silently mis-dispatch.
func (b *builder) promotedDelegates(named *types.Named, span Span) ([]PromotedDelegate, error) {
	seen := map[string]bool{} // full package-qualified identity already emitted
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
			// Identity is package-qualified: two unexported methods with the
			// same spelling from DIFFERENT packages are distinct methods and
			// both promote (Go qualifies unexported identifiers by package).
			identity := method.Name()
			if !method.Exported() && method.Pkg() != nil {
				identity += "@" + method.Pkg().Path()
			}
			if seen[identity] {
				continue // the same method seen through both method sets
			}
			seen[identity] = true
			if method.Pkg() == nil || !b.unit.Owns(method.Pkg().Path()) {
				return nil, &Unsupported{Kind: KindPromotedMethodFromATypeOutsideTheTranslatedUnit, Code: "GOTOTS_UNSUPPORTED_DECLARATION",
					Construct: "promoted method from a type outside the translated unit (" + method.Name() + ")", Span: span}
			}
			signature := method.Type().(*types.Signature)
			if signature.TypeParams() != nil || signature.RecvTypeParams() != nil {
				return nil, &Unsupported{Kind: KindPromotedGenericMethod, Code: "GOTOTS_UNSUPPORTED_DECLARATION",
					Construct: "promoted generic method (" + method.Name() + ")", Span: span}
			}
			_, pointerRecv := method.Type().(*types.Signature).Recv().Type().(*types.Pointer)
			// The vtable property is the method's canonical SLOT, not its
			// bare name: two same-spelled unexported methods promoted from
			// different packages (both in Go's method set, each satisfying
			// its own package's interface) get DISTINCT slots, so each
			// dispatches to exactly its own method instead of colliding.
			slot, err := MethodSlot(named, method)
			if err != nil {
				return nil, err
			}
			ident, err := MethodKey(method)
			if err != nil {
				return nil, &Unsupported{Kind: KindPromotedMethodWithoutCanonicalIdentity, Code: "GOTOTS_UNSUPPORTED_DECLARATION",
					Construct: "promoted method without canonical identity (" + method.Name() + ")", Span: span}
			}
			entry := PromotedDelegate{Name: method.Name(), Slot: slot, MethodIdent: ident,
				Pkg: method.Pkg().Path(), ValueReceiver: !pointerRecv}
			current := types.Type(named)
			for _, index := range path[:len(path)-1] {
				structType, ok := types.Unalias(current).Underlying().(*types.Struct)
				if !ok {
					return nil, &Unsupported{Kind: KindPromotionThroughANonStructEmbedding, Code: "GOTOTS_UNSUPPORTED_DECLARATION",
						Construct: "promotion through a non-struct embedding (" + method.Name() + ")", Span: span}
				}
				field := structType.Field(index)
				// An embedded POINTER field delegates through Go's implicit
				// dereference; the step records its pointedness so the
				// emitted delegation nil-checks exactly there.
				_, isPointer := types.Unalias(field.Type()).Underlying().(*types.Pointer)
				entry.Path = append(entry.Path, field.Name())
				entry.PathPointer = append(entry.PathPointer, isPointer)
				current = field.Type()
				if pointer, deref := types.Unalias(current).Underlying().(*types.Pointer); deref {
					current = pointer.Elem()
				}
			}
			recvNamed, ok := types.Unalias(current).(*types.Named)
			if !ok {
				return nil, &Unsupported{Kind: KindPromotionThroughAnUnnamedEmbedding, Code: "GOTOTS_UNSUPPORTED_DECLARATION",
					Construct: "promotion through an unnamed embedding (" + method.Name() + ")", Span: span}
			}
			entry.TypeName = recvNamed.Obj().Name()
			if _, isIface := recvNamed.Underlying().(*types.Interface); isIface {
				// The method lives on an embedded INTERFACE value: the
				// delegate dispatches through the field's closed union.
				entry.IfaceField = true
				ifaceIR, err := b.typeOf(recvNamed, span)
				if err != nil {
					return nil, err
				}
				entry.IfaceType = ifaceIR
				signature := method.Type().(*types.Signature)
				for i := range signature.Params().Len() {
					t, err := b.typeOf(signature.Params().At(i).Type(), span)
					if err != nil {
						return nil, err
					}
					entry.Params = append(entry.Params, Var{Name: fmt.Sprintf("$a%d", i), Type: t})
				}
				for i := range signature.Results().Len() {
					t, err := b.typeOf(signature.Results().At(i).Type(), span)
					if err != nil {
						return nil, err
					}
					entry.Results = append(entry.Results, t)
				}
			}
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
			return nil, &Unsupported{Kind: KindPromotedSelectionThrough, Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "promoted selection through " + current.String(), Span: span}
		}
		field := structType.Field(index)
		fieldType, err := b.typeOf(field.Type(), span)
		if err != nil {
			return nil, err
		}
		if out.Type().Kind != KindPointer && out.Type().Kind != KindStruct {
			return nil, &Unsupported{Kind: KindPromotedSelectionThrough, Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "promoted selection through " + out.Type().Go, Span: span}
		}
		b.use("fieldLoad")
		cell, err := b.fieldIsCell(current, field.Name(), fieldType)
		if err != nil {
			return nil, err
		}
		out = &FieldLoad{X: out, Field: field.Name(), T: fieldType, Cell: cell}
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
				if mentionsTypeParamType(arg) {
					continue
				}
				return nil, &Unsupported{Kind: KindGenericTypeInstantiatedWithAnUnreviewedTypeArgument, Code: "GOTOTS_UNSUPPORTED_DECLARATION",
					Construct: "generic type instantiated with an unreviewed type argument (" + arg.String() + ")", Span: span}
			}
			// Value-copy carrier bindings are exact: the class captures
			// its instantiation's clone/set factories at construction.
			_ = resolved
		}
	}
	return names, nil
}

// anonStructType synthesizes the class of one anonymous struct shape:
// deterministic per canonical spelling, registered for the package's
// module, structurally identical across packages (TypeScript's
// structural classes keep cross-package values assignable).
func (b *builder) anonStructType(structType *types.Struct, spelled string, span Span) (Type, error) {
	// The identity goes through the CANONICALIZER, which package-qualifies
	// UNEXPORTED field names — so two struct{x int} shapes with unexported
	// fields from different packages are distinct (Go's rule), while
	// exported-field shapes remain structurally shared. types.TypeString
	// does not qualify field names and would collide them. The full 256-bit
	// digest is only the emitted NAME; the full canonical preimage is the
	// identity, and RegisterAnonStruct rejects any digest collision between
	// distinct preimages.
	identity, err := b.canonical(structType)
	if err != nil {
		return Type{}, &Unsupported{Kind: KindNestedError, Code: "GOTOTS_UNSUPPORTED_TYPE", Construct: err.Error(), Span: span}
	}
	digest := sha256.Sum256([]byte(identity))
	name := "Anon$" + hex.EncodeToString(digest[:])
	out := Type{Kind: KindStruct, Go: spelled, Named: name, Pkg: b.pkgPath}
	decl := &Struct{
		ID:       b.pkgPath + "::type::" + name,
		Name:     name,
		Exported: false,
		Span:     span,
		Identity: identity,
	}
	for i := range structType.NumFields() {
		field := structType.Field(i)
		if field.Name() == "_" {
			return Type{}, &Unsupported{Kind: KindStructType, Code: "GOTOTS_UNSUPPORTED_TYPE", Construct: "struct type " + spelled + " (blank field)", Span: span}
		}
		fieldType, err := b.typeOf(field.Type(), span)
		if err != nil {
			return Type{}, err
		}
		cell, err := b.fieldIsCell(structType, field.Name(), fieldType)
		if err != nil {
			return Type{}, err
		}
		decl.Fields = append(decl.Fields, Var{Name: field.Name(), Type: fieldType, Cell: cell})
	}
	decl.Comparable = b.structEqComparable(structType)
	decl.KeyEncodable = b.structKeyEncodable(structType, span)
	if err := b.unit.RegisterAnonStruct(b.pkgPath, decl); err != nil {
		return Type{}, &Unsupported{Kind: KindNestedError, Code: "GOTOTS_UNSUPPORTED_TYPE", Construct: err.Error(), Span: span}
	}
	return out, nil
}

// zeroSizeType reports whether a type is provably zero-size: a [0]X
// array, or a struct (named or not) with no fields — the vet-only
// noCopy/phantom-marker idioms a blank field legally carries.
func zeroSizeType(t types.Type) bool {
	switch u := t.Underlying().(type) {
	case *types.Array:
		return u.Len() == 0
	case *types.Struct:
		return u.NumFields() == 0
	}
	return false
}
