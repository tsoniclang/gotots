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

// canonicalTypeID is the path-qualified identity of one Go type: the
// ONLY string used as generated dynamic-type identity. types.Type.String
// qualifies by package NAME and can collide across packages sharing a
// name; this cannot. It renders within the builder's binder environment
// so a type-parameter-bearing carrier (e.g. *K inside a generic method)
// resolves instead of failing closed.
func (b *builder) canonicalTypeID(t types.Type) (string, error) {
	return b.canonical(t)
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
		return RttiRef{}, err
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
	if rtti.Composite != "" && rtti.ExternID == "" && !mentionsTypeParamType(source) {
		// The boxed-composite enumeration: this exact composite becomes
		// an exact union member (its payload type, no erasure). A
		// composite that mentions a type parameter is per-instantiation
		// and never a concrete boxed member.
		b.unit.AddBoxedComposite(rtti.Composite, built.Type())
	}
	// An external named type BOXED here joins the dynamic-type universe:
	// this catches inferred external dynamic types (value := external.New())
	// precisely, without walking unrelated external internals.
	b.registerBoxedExtern(source)
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
