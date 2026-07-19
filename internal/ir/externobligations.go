// External-contract obligations recorded on a unit Scope: admitted
// external functions, referenced external named types and their methods,
// and external variables. The stub modules and dispatch tables are built
// from these.
package ir

import (
	"fmt"
	"go/types"
	"sort"
	"strings"
)

// ExternMethodObligation is the ONE authoritative record of a referenced
// external method: its canonical identity key, its dispatch slot (== the
// box-vtable property and IfaceMember.Slots selector), and its object. The
// three are computed and validated together at the sole constructor
// (AddExternalMethod), so they cannot drift and no consumer re-derives the
// key or the slot from the bare method name.
type ExternMethodObligation struct {
	Key    string
	Slot   string
	Method *types.Func
}

// AddExternalFunc records one admitted external function contract.
func (s Scope) AddExternalFunc(fn *types.Func) { s.externals[fn] = true }

// AddExternalType records one external named type the unit carries; the
// stub module exports its value-semantics contract.
func (s Scope) AddExternalType(pkg, name string) *ExternTypeObligation {
	id := pkg + "." + name
	obligation, has := s.externTypes[id]
	if !has {
		obligation = &ExternTypeObligation{
			Pkg: pkg, Name: name,
			methods: map[string]ExternMethodObligation{},
		}
		s.externTypes[id] = obligation
	}
	return obligation
}

// AddLiteralShape records one keyed composite-literal constructor
// obligation on the external type (deduplicated by the exact field set)
// and returns its stub symbol.
func (o *ExternTypeObligation) AddLiteralShape(fields []string, fieldTypes []Type) string {
	key := strings.Join(fields, "$")
	if o.literalShapes == nil {
		o.literalShapes = map[string]ExternLiteralShape{}
	}
	if _, has := o.literalShapes[key]; !has {
		o.literalShapes[key] = ExternLiteralShape{Fields: fields, FieldTypes: fieldTypes}
	}
	return o.Name + "$lit$" + key + "$"
}

// LiteralShapes returns the recorded constructor obligations sorted by
// field-set key.
func (o *ExternTypeObligation) LiteralShapes() []ExternLiteralShape {
	keys := make([]string, 0, len(o.literalShapes))
	for key := range o.literalShapes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]ExternLiteralShape, 0, len(keys))
	for _, key := range keys {
		out = append(out, o.literalShapes[key])
	}
	return out
}

// AddExternalMethod is the SOLE validating constructor of an external
// method obligation. It records one referenced method of the external type
// `named` as a single atomic record, keyed by its FULL canonical identity
// (MethodKey) — never its bare Go name. It validates, together and up
// front, everything a downstream consumer would otherwise have to re-check:
//
//   - REPRESENTABILITY: a generic external method (its own or its
//     receiver's type parameters, or any type-parameter mention) has no
//     single exact stub signature or dispatch adapter. It is rejected here,
//     so the obligation set contains ONLY representable methods and no
//     consumer can silently skip one.
//   - IDENTITY and SLOT: the canonical key and the dispatch slot are
//     computed once and stored on the record, so dispatch, the box vtable,
//     and stub emission all read the SAME values.
//
// Any failure fails closed IMMEDIATELY: the error propagates through this
// typed return and NOTHING is recorded — no in-band poison key, no partial
// obligation, no downstream recovery.
func (s Scope) AddExternalMethod(named *types.Named, method *types.Func) error {
	signature, ok := method.Type().(*types.Signature)
	if !ok {
		return fmt.Errorf("ir: external method %s has no signature", method.Name())
	}
	if (signature.TypeParams() != nil && signature.TypeParams().Len() > 0) ||
		(signature.RecvTypeParams() != nil && signature.RecvTypeParams().Len() > 0) ||
		SignatureMentionsTypeParam(signature) {
		return fmt.Errorf("ir: external method %s.%s is generic and not representable as an external obligation",
			named.Obj().Pkg().Path(), method.Name())
	}
	key, err := MethodKey(method)
	if err != nil {
		return err
	}
	slot, err := MethodSlot(named, method)
	if err != nil {
		return err
	}
	s.AddExternalType(named.Obj().Pkg().Path(), named.Obj().Name()).
		methods[key] = ExternMethodObligation{Key: key, Slot: slot, Method: method}
	return nil
}

// MethodByKey returns the atomic obligation record for one canonical key.
func (o *ExternTypeObligation) MethodByKey(key string) (ExternMethodObligation, bool) {
	entry, ok := o.methods[key]
	return entry, ok
}

// Methods returns every recorded method obligation in deterministic
// (canonical-key-sorted) order. Each record carries its key, slot, and
// object together — the single join across dispatch, vtable, and stub
// emission. Every recorded method is representable (AddExternalMethod
// rejects the rest), so a consumer that emits one per record omits none.
func (o *ExternTypeObligation) Methods() []ExternMethodObligation {
	keys := make([]string, 0, len(o.methods))
	for key := range o.methods {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]ExternMethodObligation, 0, len(keys))
	for _, key := range keys {
		out = append(out, o.methods[key])
	}
	return out
}

// ExternalTypes returns every referenced external type obligation in
// sorted identity order.
func (s Scope) ExternalTypes() []*ExternTypeObligation {
	ids := make([]string, 0, len(s.externTypes))
	for id := range s.externTypes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]*ExternTypeObligation, 0, len(ids))
	for _, id := range ids {
		out = append(out, s.externTypes[id])
	}
	return out
}
