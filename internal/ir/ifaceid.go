// Canonical interface and method identity: the path-qualified,
// method-key-extended identities used as the ONE dynamic-type / dispatch
// key. Distinct from go/types spellings, which qualify by package NAME.
package ir

import (
	"crypto/sha256"
	"encoding/hex"
	"go/types"
	"strings"

	"github.com/tsoniclang/gotots/internal/typeid"
)

// canonicalTypeID is the RUNTIME dynamic-type identity of one Go type:
// the token an interface box carries and dispatch switches on. It is
// deliberately NOT rendered within a binder environment — an open type
// parameter has NO single runtime identity (Box[int] boxes []int,
// Box[string] boxes []string), so a type mentioning a type parameter
// FAILS CLOSED here rather than producing a shared, wrong token. (A
// per-instantiation RTTI mechanism would specialize it; until then this
// is fail-closed, and such bodies are unimplemented.)
func (b *builder) canonicalTypeID(t types.Type) (string, error) {
	return typeid.Canonical(t)
}

// canonicalIfaceID extends the path-qualified type string with each
// method's canonical key: the type string alone does not package-qualify
// UNEXPORTED method names, so two structurally identical unexported
// interfaces from different packages would collide (distinct Go types,
// disjoint implementer sets).
func (b *builder) canonicalIfaceID(iface *types.Interface) (string, error) {
	id, err := b.canonicalTypeID(iface)
	if err != nil {
		return "", err
	}
	for i := range iface.NumMethods() {
		key, err := MethodKey(iface.Method(i))
		if err != nil {
			return "", err
		}
		id += "|" + key
	}
	return id, nil
}

// compositeRtti interns an rtti for a composite or external type when
// the value's carrier itself is reviewed.
func (b *builder) compositeRtti(t types.Type, span Span, externID string) (RttiRef, error) {
	resolved, err := b.typeOf(t, span)
	if err != nil {
		return RttiRef{}, err
	}
	composite, err := b.canonicalTypeID(t)
	if err != nil {
		return RttiRef{}, &Unsupported{Kind: KindRuntimeTypeIdentityOf, Code: "GOTOTS_UNSUPPORTED_TYPE",
			Construct: "runtime type identity of " + t.String() + " (an open type parameter has no single runtime type)", Span: span}
	}
	out := RttiRef{Composite: composite, Display: displayOf(t), ExternID: externID}
	if externID == "" {
		switch types.Unalias(t).Underlying().(type) {
		case *types.Slice, *types.Map, *types.Signature:
			out.CompositeEq = "uncomparable"
		case *types.Pointer:
			out.CompositeEq = "identity"
		case *types.Array:
			elem := resolved.Elem
			if elem != nil && (elem.Kind.Integer() || elem.Kind.Float() ||
				elem.Kind == KindString || elem.Kind == KindBool || elem.Kind == KindPointer) {
				out.CompositeEq = "array-prim"
			} else {
				out.CompositeEq = "unknown"
			}
		default:
			out.CompositeEq = "unknown"
		}
	}
	return out, nil
}

// displayOf spells a type the way Go's runtime messages do: package
// names (not paths) qualify named types, and the empty interface prints
// with the runtime's spacing.
func displayOf(t types.Type) string {
	spelled := types.TypeString(t, func(p *types.Package) string { return p.Name() })
	if spelled == "any" {
		return "interface {}"
	}
	return strings.ReplaceAll(spelled, "interface{}", "interface {}")
}

// predeclaredRttiName canonicalizes a predeclared basic kind onto its
// ABI rtti name.
func predeclaredRttiName(kind types.BasicKind) (string, bool) {
	switch kind {
	case types.Bool:
		return "bool", true
	case types.String:
		return "string", true
	case types.Int:
		return "int", true
	case types.Int8:
		return "int8", true
	case types.Int16:
		return "int16", true
	case types.Int32:
		return "int32", true
	case types.Int64:
		return "int64", true
	case types.Uint:
		return "uint", true
	case types.Uint8:
		return "uint8", true
	case types.Uint16:
		return "uint16", true
	case types.Uint32:
		return "uint32", true
	case types.Uint64:
		return "uint64", true
	case types.Uintptr:
		return "uintptr", true
	case types.Float32:
		return "float32", true
	case types.Float64:
		return "float64", true
	}
	return "", false
}

// boxIfaceValue converts a concrete expression to an interface value at
// a binding site. Interface-to-interface bindings pass the box through;
// nil stays the nil interface; struct values are copied into the box.
func (b *builder) boxIfaceValue(built Expr, source types.Type, expected Type, span Span) (Expr, error) {
	if built.Type().Kind == KindIface && built.Type().TypeParamName != "" {
		// A type-parameter VALUE is not an interface box: boxing T into a
		// concrete union needs the binding's runtime type identity, which
		// varies per instantiation. Fail closed (the body placeholders)
		// rather than pass the raw value where a box is expected.
		return nil, &Unsupported{Kind: KindInterfaceValueOfType, Code: "GOTOTS_UNSUPPORTED_EXPRESSION",
			Construct: "interface value of type " + source.String() + " (type-parameter boxing varies per instantiation)", Span: span}
	}
	if built.Type().Kind == KindIface {
		return built, nil
	}
	if _, isNil := built.(*NilConst); isNil {
		return built, nil
	}
	rtti, err := b.rttiFor(source, span)
	if err != nil {
		return nil, err
	}
	if rtti.Composite == "" && rtti.TypeName != "" && !rtti.Pointer {
		// VALUE-form boxing of a named type: reachability evidence for
		// union $key encoders' unreachability claims.
		b.unit.AddValueBoxedNamed(rtti.Pkg + "." + rtti.TypeName)
	}
	if rtti.Composite != "" && rtti.ExternID == "" && !mentionsTypeParamType(source) {
		// The boxed-composite enumeration: this exact composite becomes
		// an exact union member (its payload type, no erasure). A
		// composite that mentions a type parameter is per-instantiation
		// and never a concrete boxed member.
		plan, err := b.compositeEqPlan(source, span)
		if err != nil {
			return nil, err
		}
		b.unit.AddBoxedComposite(rtti.Composite, built.Type(), plan)
	}
	b.use("ifaceBox")
	return &IfaceBox{X: b.bindStructValue(built), Rtti: rtti, T: expected}, nil
}

// MethodKey is the canonical dynamic-dispatch identity of one method:
// its name, its package identity when unexported (Go's method sets
// match unexported methods only within their declaring package), and a
// digest of its path-qualified signature — so name-only collisions can
// never dispatch or satisfy a method-set test.
func MethodKey(method *types.Func) (string, error) {
	// The FULL 256-bit signature digest: a truncated prefix is not
	// collision-free, and a dispatch/method-set key must never conflate
	// two distinct signatures. The key is an internal identity, so length
	// does not matter.
	signature, err := typeid.MethodCanonical(method)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256([]byte(signature))
	key := method.Name()
	if !method.Exported() && method.Pkg() != nil {
		key += "@" + method.Pkg().Path()
	}
	return key + "|" + hex.EncodeToString(digest[:]), nil
}

// MethodSlot is the vtable property name and dispatch selector for one
// method WITHIN a concrete type's dispatch surface. It is the bare method
// name when that name is unique in the type's method set (the overwhelming
// case, so generated dispatch stays readable and unchanged). When a type's
// method set carries two DISTINCT methods of the same bare name — a type
// promoting same-spelled unexported methods from two different packages,
// which Go admits in the method set (each satisfies its own package's
// interface) even though the selector expression x.m is ambiguous — the
// slot is disambiguated by the method's canonical identity, so the two
// slots are distinct and each interface dispatches to exactly its own
// method. Both the vtable construction and the dispatch site derive the
// selector through THIS function, so they always agree.
// methodReceiverNamed returns a method signature's receiver named type
// (pointer unwrapped), or nil if the receiver is not a named type.
func methodReceiverNamed(sig *types.Signature) *types.Named {
	recv := sig.Recv()
	if recv == nil {
		return nil
	}
	t := types.Unalias(recv.Type())
	if p, ok := t.(*types.Pointer); ok {
		t = types.Unalias(p.Elem())
	}
	named, _ := t.(*types.Named)
	return named
}

func MethodSlot(named *types.Named, method *types.Func) (string, error) {
	name := method.Name()
	set := types.NewMethodSet(types.NewPointer(named))
	collisions := 0
	for i := range set.Len() {
		if set.At(i).Obj().Name() == name {
			collisions++
		}
	}
	if collisions <= 1 {
		return name, nil
	}
	key, err := MethodKey(method)
	if err != nil {
		return "", err
	}
	// The FULL digest of the canonical identity — a truncated prefix is not
	// an injective identity, and the slot is an internal TS property name so
	// its length is free. Distinct canonical methods therefore get distinct
	// slots with no birthday collision.
	digest := sha256.Sum256([]byte(key))
	return name + "$s" + hex.EncodeToString(digest[:]), nil
}
