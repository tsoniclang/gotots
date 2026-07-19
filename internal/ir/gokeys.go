// Map-key and equality admissibility: which Go types may serve as map
// keys or comparable values in the emitted representation, including the
// coinductive guards that make self-referential structures decidable.
package ir

import (
	"go/types"
)

// mapKeySupported admits key kinds whose Go equality coincides with JS
// Map's SameValueZero over the canonical carriers: strings, booleans,
// canonical-range integers/bigints, and pointers (Go pointer equality is
// exactly JS object identity, with undefined for nil). Floats are
// excluded (NaN semantics differ) and composite keys need generated
// hashing.
func mapKeySupported(kind Kind) bool {
	return kind == KindString || kind == KindBool || kind == KindPointer || kind.Integer() || kind == KindUnit
}

// KeyEncodableField reports whether one field type participates in the
// generated canonical key encoding (goKey$): kinds whose Go equality
// maps injectively onto a deterministic string component. Floats (NaN,
// signed zeros), interfaces, and nested named structs stay out.
func KeyEncodableField(t Type) bool {
	switch t.Kind {
	case KindString, KindBool, KindPointer, KindUnit:
		return true
	case KindArray:
		return KeyEncodableField(*t.Elem)
	}
	// Floats encode through the exact float-key rule: -0 folds onto +0
	// and every NaN encode is a fresh token, so a NaN-field key is
	// unretrievable — exactly Go.
	return t.Kind.Integer() || t.Kind.Float()
}

// typeParamKeySupported admits a type-parameter map key when every
// recorded instantiation of the enclosing generic function binds it to
// a carrier whose Go equality is JS SameValueZero — the closed-world
// evidence makes the direct Map carrier exact for all of them.
func (b *builder) typeParamKeySupported(keyType types.Type, span Span) bool {
	param, ok := types.Unalias(keyType).(*types.TypeParam)
	if !ok {
		return false
	}
	var instances [][]types.Type
	switch {
	case b.genericObj != nil:
		signature := b.genericObj.Type().(*types.Signature)
		if signature.TypeParams() == nil || param.Index() >= signature.TypeParams().Len() ||
			signature.TypeParams().At(param.Index()) != param {
			return false
		}
		instances = b.unit.GenericInstances(b.genericObj)
	case b.genericTypeObj != nil:
		typeParams := b.genericTypeObj.TypeParams()
		if typeParams == nil || param.Index() >= typeParams.Len() ||
			typeParams.At(param.Index()).Obj().Name() != param.Obj().Name() {
			return false
		}
		instances = b.unit.GenericTypeInstances(b.genericTypeObj.Obj())
	default:
		return false
	}
	if len(instances) == 0 {
		return false
	}
	// The declaration admits with the direct GoMap carrier: the prepass
	// recorded the SVZ-key REQUIREMENT for this parameter, and every
	// concrete instantiation site is guarded against it (a non-SVZ binding
	// fails closed AT ITS SITE), so the carrier stays exact for every
	// instantiation that exists in emitted code. The evidence is consulted
	// only to confirm the requirement machinery covers this declaration.
	_ = instances
	return true
}

// EqComparableField reports whether one field type participates in the
// generated exact equality (goEq$): direct === carriers (floats keep
// NaN and signed-zero semantics under ===), nested comparable structs,
// arrays of those, and interfaces (whose equality may panic, exactly
// Go). External fields and uncomparable kinds stay out.
func (b *builder) EqComparableField(t Type, goType types.Type) bool {
	switch t.Kind {
	case KindIface:
		// A type-parameter field's equality is instantiation-dependent
		// (=== for one instantiation, interface equality for another):
		// the shared goEq$ body cannot be exact, so the struct stays out.
		return t.TypeParamName == ""
	case KindString, KindBool, KindPointer, KindUnit, KindFloat32, KindFloat64:
		return true
	case KindArray:
		elemGo := goType
		if arr, ok := types.Unalias(goType).Underlying().(*types.Array); ok {
			elemGo = arr.Elem()
		}
		return b.EqComparableField(*t.Elem, elemGo)
	case KindStruct:
		return b.structEqComparable(goType)
	}
	return t.Kind.Integer()
}

// structEqComparable reports whether a struct's generated class carries
// goEq$ — exact field-wise equality over every field.
func (b *builder) structEqComparable(goType types.Type) bool {
	structType, ok := types.Unalias(goType).Underlying().(*types.Struct)
	if !ok {
		return false
	}
	guard := "eq:" + goType.String()
	if b.keyEncodableInProgress[guard] {
		// COINDUCTIVE cycle (a struct reachable through its own fields):
		// assuming comparable is self-consistent — pointer fields compare
		// by identity, so value recursion terminates.
		return true
	}
	if b.keyEncodableInProgress == nil {
		b.keyEncodableInProgress = map[string]bool{}
	}
	b.keyEncodableInProgress[guard] = true
	defer delete(b.keyEncodableInProgress, guard)
	for i := range structType.NumFields() {
		field := structType.Field(i)
		fieldIR, err := b.typeOf(field.Type(), Span{})
		if err != nil || !b.EqComparableField(fieldIR, field.Type()) {
			return false
		}
	}
	return true
}

// structKeyEncodable reports whether a named struct key's fields are all
// encodable, so its class carries goKey$ and the keyed-map carrier can
// hold it. A nested comparable struct field composes through the field
// type's own goKey$, so the recursion descends into struct and
// array-of-struct fields.
func (b *builder) structKeyEncodable(keyType types.Type, span Span) bool {
	structType, ok := types.Unalias(keyType).Underlying().(*types.Struct)
	if !ok {
		return false
	}
	guard := keyType.String()
	if b.keyEncodableInProgress[guard] {
		// A cycle through member/field types: the definition is
		// COINDUCTIVE (assuming encodable is self-consistent — value
		// recursion terminates because pointer members encode by
		// identity), so the in-progress type counts as encodable.
		return true
	}
	if b.keyEncodableInProgress == nil {
		b.keyEncodableInProgress = map[string]bool{}
	}
	b.keyEncodableInProgress[guard] = true
	defer delete(b.keyEncodableInProgress, guard)
	for i := range structType.NumFields() {
		if !b.keyEncodableFieldType(structType.Field(i).Type(), span) {
			return false
		}
	}
	return true
}

// keyEncodableFieldType reports whether one struct field's Go type
// participates in the generated canonical key encoding. Scalars, byte
// strings, pointers (identity), and unit encode directly; a nested named
// struct encodes through its own goKey$ (recursion); a fixed array
// encodes element-wise. Floats (NaN, signed zeros), interfaces, external
// types, and maps/slices stay out — their Go equality is not injective
// onto a deterministic string.
func (b *builder) keyEncodableFieldType(goType types.Type, span Span) bool {
	fieldIR, err := b.typeOf(goType, span)
	if err != nil {
		return false
	}
	switch fieldIR.Kind {
	case KindStruct:
		return b.structKeyEncodable(goType, span)
	case KindArray:
		if arr, ok := types.Unalias(goType).Underlying().(*types.Array); ok {
			return b.keyEncodableFieldType(arr.Elem(), span)
		}
		return false
	case KindIface:
		// An interface FIELD encodes through the union's generated $key
		// (Go's dynamic-key equality) when every member is encodable.
		if fieldIR.TypeParamName != "" {
			return false
		}
		return b.ifaceKeyMembersEncodable(goType, span)
	}
	return KeyEncodableField(fieldIR)
}

// ifaceKeyMembersEncodable reports whether every member of an interface
// type used as a MAP KEY is exactly encodable: pointer identity, a
// scalar carrier (basic-underlying external types included), or a
// key-encodable struct value. An external HANDLE value member (opaque,
// compared by value in Go) has no exact encoding and fails closed. The
// EMPTY interface additionally admits: predeclared members are scalars
// and boxed composites either encode (identity/array-prim) or carry Go's
// exact unhashable panic.
func (b *builder) ifaceKeyMembersEncodable(goType types.Type, span Span) bool {
	iface, ok := goType.Underlying().(*types.Interface)
	if !ok {
		return false
	}
	guard := "iface:" + goType.String()
	if b.keyEncodableInProgress[guard] {
		return true // coinductive: see structKeyEncodable
	}
	if b.keyEncodableInProgress == nil {
		b.keyEncodableInProgress = map[string]bool{}
	}
	b.keyEncodableInProgress[guard] = true
	defer delete(b.keyEncodableInProgress, guard)
	members, err := b.ifaceMembers(iface, span)
	if err != nil {
		return false
	}
	if iface.NumMethods() == 0 {
		// The empty interface: named members checked below; predeclared
		// and composite members are always encodable-or-panicking.
	} else if len(members) == 0 {
		return false
	}
	for _, member := range members {
		switch {
		case member.Pointer:
		case member.Extern && member.ExternCarrier != "":
		case member.Extern:
			return false // opaque handle value: no exact encoding
		case member.Struct:
			if !member.KeyEncodable {
				// An UNCOMPARABLE dynamic key panics in Go ("hash of
				// unhashable type") — the panic IS the exact encoding; a
				// comparable member without an encoding fails closed.
				if member.Eq == nil || member.Eq.Kind != EqUncomparable {
					return false
				}
			}
		default:
			// A named value carrier over a scalar: encodable by carrier.
		}
	}
	return true
}

// paramPtrIdentity reports whether every CONCRETE binding of a type
// parameter (in the enclosing generic declaration's closed evidence) is
// an identity carrier — a struct, fixed array, or external value — so a
// pointer to the parameter is the instance itself, exactly as in every
// concrete instantiation. Mixed or scalar evidence keeps the cell form.
func (b *builder) paramPtrIdentity(keyType types.Type, span Span) bool {
	param, ok := types.Unalias(keyType).(*types.TypeParam)
	if !ok {
		return false
	}
	var instances [][]types.Type
	switch {
	case b.genericObj != nil:
		signature := b.genericObj.Type().(*types.Signature)
		if signature.TypeParams() == nil || param.Index() >= signature.TypeParams().Len() ||
			signature.TypeParams().At(param.Index()) != param {
			return false
		}
		instances = b.unit.GenericInstances(b.genericObj)
	case b.genericTypeObj != nil:
		typeParams := b.genericTypeObj.TypeParams()
		if typeParams == nil || param.Index() >= typeParams.Len() ||
			typeParams.At(param.Index()).Obj().Name() != param.Obj().Name() {
			return false
		}
		instances = b.unit.GenericTypeInstances(b.genericTypeObj.Obj())
	default:
		return false
	}
	concrete := 0
	for _, instance := range instances {
		if param.Index() >= len(instance) {
			return false
		}
		arg := instance[param.Index()]
		if mentionsTypeParamType(arg) {
			continue
		}
		concrete++
		bound, err := b.typeOf(arg, span)
		if err != nil {
			return false
		}
		if bound.Kind != KindStruct && bound.Kind != KindArray && bound.Kind != KindExternal {
			return false
		}
	}
	return concrete > 0
}
