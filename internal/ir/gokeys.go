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
		if recvParams := signature.RecvTypeParams(); recvParams != nil {
			// A method on a generic type: its in-scope parameters are the
			// RECEIVER's (Go permits no method-own type parameters), and
			// the identity evidence is the receiver type's closed
			// instantiation set — exactly the set every concrete call
			// resolves against.
			if param.Index() >= recvParams.Len() ||
				recvParams.At(param.Index()).Obj().Name() != param.Obj().Name() {
				return false
			}
			recv := signature.Recv().Type()
			if pointer, isPointer := types.Unalias(recv).(*types.Pointer); isPointer {
				recv = pointer.Elem()
			}
			named, isNamed := types.Unalias(recv).(*types.Named)
			if !isNamed {
				return false
			}
			instances = b.unit.GenericTypeInstances(named.Obj())
			break
		}
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

// HasEncFamilyInstances reports whether the closed evidence instantiates
// this generic struct with a struct binding at any HARD map-keyed
// position — the "$ek" emission variant exists exactly then.
func HasEncFamilyInstances(unit Scope, named *types.Named) bool {
	obj := named.Origin().Obj()
	hard := []int{}
	if named.TypeParams() != nil {
		for i := range named.TypeParams().Len() {
			if unit.ParamRequiresSVZKey(obj, i) {
				hard = append(hard, i)
			}
		}
	}
	if len(hard) == 0 {
		return false
	}
	for _, vector := range unit.GenericTypeInstances(obj) {
		for _, index := range hard {
			if index >= len(vector) || mentionsTypeParamType(vector[index]) {
				continue
			}
			if _, isStruct := types.Unalias(vector[index]).Underlying().(*types.Struct); isStruct {
				return true
			}
		}
	}
	return false
}

// HasEncFamilyFuncInstances reports whether the closed evidence
// instantiates this generic FUNCTION with a struct binding at any HARD
// map-keyed position — its "$ek" emission variant exists exactly then.
func HasEncFamilyFuncInstances(unit Scope, fn *types.Func) bool {
	signature, isSig := fn.Type().(*types.Signature)
	if !isSig || signature.TypeParams() == nil {
		return false
	}
	hard := []int{}
	for i := range signature.TypeParams().Len() {
		if unit.ParamRequiresSVZKey(fn, i) {
			hard = append(hard, i)
		}
	}
	if len(hard) == 0 {
		return false
	}
	for _, vector := range unit.GenericInstances(fn) {
		for _, index := range hard {
			if index >= len(vector) || mentionsTypeParamType(vector[index]) {
				continue
			}
			if _, isStruct := types.Unalias(vector[index]).Underlying().(*types.Struct); isStruct {
				return true
			}
		}
	}
	return false
}

// instanceFamilyEnc reports whether a generic-struct INSTANCE binds any
// HARD map-keyed parameter to a struct — selecting the encoded-family
// ("$ek") class/method variant for this instance.
func (b *builder) instanceFamilyEnc(named *types.Named) bool {
	origin := named.Origin()
	args := named.TypeArgs()
	if args == nil {
		return false
	}
	for i := range args.Len() {
		if !b.unit.ParamRequiresSVZKey(origin.Obj(), i) {
			continue
		}
		arg := args.At(i)
		if mentionsTypeParamType(arg) {
			continue
		}
		if _, isStruct := types.Unalias(arg).Underlying().(*types.Struct); isStruct {
			return true
		}
	}
	return false
}

// EqComparableField reports whether one field type participates in the
// generated exact equality (goEq$): direct === carriers (floats keep
// NaN and signed-zero semantics under ===), nested comparable structs,
// arrays of those, and interfaces (whose equality may panic, exactly
// Go). External fields and uncomparable kinds stay out.
func (b *builder) EqComparableField(t Type, goType types.Type) bool {
	switch t.Kind {
	case KindIface:
		// A bare type-parameter field compares through the eq$P operation
		// the class captured at construction — exact per instantiation
		// (goEqUnsupported for Go-illegal bindings). A concrete interface
		// field compares through its union's exact equality.
		return true
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
	if verdict, memoized := b.eqComparableMemo[guard]; memoized {
		return verdict
	}
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
	verdict := true
	for i := range structType.NumFields() {
		field := structType.Field(i)
		fieldIR, err := b.typeOf(field.Type(), Span{})
		if err != nil || !b.EqComparableField(fieldIR, field.Type()) {
			verdict = false
			break
		}
	}
	if b.eqComparableMemo == nil {
		b.eqComparableMemo = map[string]bool{}
	}
	b.eqComparableMemo[guard] = verdict
	return verdict
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
	if verdict, memoized := b.keyEncodableMemo[guard]; memoized {
		return verdict
	}
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
	verdict := b.structKeyEncodableWalk(keyType, structType, span)
	if b.keyEncodableMemo == nil {
		b.keyEncodableMemo = map[string]bool{}
	}
	b.keyEncodableMemo[guard] = verdict
	return verdict
}

// structKeyEncodableWalk is structKeyEncodable's uncached walk.
func (b *builder) structKeyEncodableWalk(keyType types.Type, structType *types.Struct, span Span) bool {
	// An INSTANTIATED generic struct key must bind every bare-parameter
	// field to the scalar family the generic class's goKey$ encodes
	// exactly (goKeyScalar: strings, booleans, integers — floats stay out
	// for NaN/signed-zero key semantics). The origin declaration says
	// which fields are bare parameters; the instance says what they bound.
	if named, isNamed := types.Unalias(keyType).(*types.Named); isNamed && named.TypeArgs() != nil && named.TypeArgs().Len() > 0 {
		originStruct, isStruct := named.Origin().Underlying().(*types.Struct)
		if !isStruct || originStruct.NumFields() != structType.NumFields() {
			return false
		}
		for i := range structType.NumFields() {
			if _, isParam := types.Unalias(originStruct.Field(i).Type()).(*types.TypeParam); isParam {
				if !scalarKeyBinding(structType.Field(i).Type()) {
					return false
				}
				continue
			}
			if !b.keyEncodableFieldType(structType.Field(i).Type(), span) {
				return false
			}
		}
		return true
	}
	for i := range structType.NumFields() {
		if !b.keyEncodableFieldType(structType.Field(i).Type(), span) {
			return false
		}
	}
	return true
}

// scalarKeyBinding reports whether one instantiation binding of a
// bare-parameter key field is inside goKeyScalar's exact family:
// strings, booleans, and integers (floats' NaN and signed-zero key
// semantics need goKeyFloat, which the shared generic encoding cannot
// select per binding).
func scalarKeyBinding(goType types.Type) bool {
	basic, ok := types.Unalias(goType).Underlying().(*types.Basic)
	if !ok {
		return false
	}
	info := basic.Info()
	return info&types.IsString != 0 || info&types.IsBoolean != 0 || info&types.IsInteger != 0
}

// keyEncodableFieldType reports whether one struct field's Go type
// participates in the generated canonical key encoding. Scalars, byte
// strings, pointers (identity), and unit encode directly; a nested named
// struct encodes through its own goKey$ (recursion); a fixed array
// encodes element-wise. Floats (NaN, signed zeros), interfaces, external
// types, and maps/slices stay out — their Go equality is not injective
// onto a deterministic string.
func (b *builder) keyEncodableFieldType(goType types.Type, span Span) bool {
	if _, isParam := types.Unalias(goType).(*types.TypeParam); isParam {
		// A bare type-parameter field encodes via the captured key$P —
		// checked HERE (before typeOf) so the origin walk also resolves
		// from outside the generic declaration's scope.
		return true
	}
	if _, isPtr := types.Unalias(goType).Underlying().(*types.Pointer); isPtr {
		// Every pointer field keys by object identity (goKeyId; the GoPtr
		// carrier's both branches are object identities) — structural,
		// so origin walks resolve param-mentioning pointers too.
		return true
	}
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
		if fieldIR.TypeParamName != "" {
			// A bare type-parameter field encodes via goKeyScalar in the
			// generic class's goKey$; the INSTANTIATION-side admission
			// (structKeyEncodable's origin walk) restricts bindings to the
			// scalar family that encoding is exact over.
			return true
		}
		// An interface FIELD encodes through the union's generated $key
		// (Go's dynamic-key equality) when every member is encodable.
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
			// An uncomparable extern dynamic key panics in Go — the panic
			// IS the exact encoding. A COMPARABLE opaque handle would hash
			// contents the handle does not expose: its component is a loud
			// runtime fail-closed stop (goKeyExternOpaque), never a wrong
			// identity encoding — the admission covers the members that
			// can actually inhabit keys exactly.
		case member.Struct:
			// A key-encodable struct composes through its goKey$; an
			// UNCOMPARABLE one carries Go's exact unhashable panic; a
			// comparable struct WITHOUT a generated encoding gets the loud
			// runtime fail-closed stop (goKeyUnreachable, finalize-verified) — never a wrong
			// identity encoding.
		default:
			// A named value carrier over a scalar: encodable by carrier.
		}
	}
	return true
}
