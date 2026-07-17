// External-contract obligations recorded on a unit Scope: admitted
// external functions, referenced external named types and their methods,
// and external variables. The stub modules and dispatch tables are built
// from these.
package ir

import (
	"go/types"
	"sort"
)

// AddExternalFunc records one admitted external function contract.
func (s Scope) AddExternalFunc(fn *types.Func) { s.externals[fn] = true }

// AddExternalType records one external named type the unit carries; the
// stub module exports its value-semantics contract.
func (s Scope) AddExternalType(pkg, name string) *ExternTypeObligation {
	id := pkg + "." + name
	obligation, has := s.externTypes[id]
	if !has {
		obligation = &ExternTypeObligation{Pkg: pkg, Name: name, methods: map[string]*types.Func{}}
		s.externTypes[id] = obligation
	}
	return obligation
}

// AddExternalMethod records one referenced method of an external type,
// keyed by its FULL canonical identity (MethodKey) — never its bare Go
// name. Two distinct methods (e.g. same-spelled unexported methods owned
// by different packages) are therefore distinct obligation members with
// distinct keys; neither overwrites the other, and there is no bare-name
// fallback. A method whose identity cannot be resolved fails closed
// through a poison key that emission rejects.
func (s Scope) AddExternalMethod(pkg, name string, method *types.Func) {
	obligation := s.AddExternalType(pkg, name)
	key, err := MethodKey(method)
	if err != nil {
		key = "\x00unresolved\x00" + method.Name()
	}
	obligation.methods[key] = method
}

// ExternMethodEntry pairs one external method with its canonical key —
// the join key between dispatch, vtable, and stub emission.
type ExternMethodEntry struct {
	Key    string
	Method *types.Func
}

// SortedMethods returns the type's referenced methods in deterministic
// canonical-key order — each a distinct method identity.
func (o *ExternTypeObligation) SortedMethods() []*types.Func {
	out := make([]*types.Func, 0, len(o.methods))
	for _, entry := range o.MethodKeys() {
		out = append(out, entry.Method)
	}
	return out
}

// MethodByKey returns the method with the given canonical key, or nil.
func (o *ExternTypeObligation) MethodByKey(key string) *types.Func {
	return o.methods[key]
}

// MethodKeys returns each referenced method's canonical key alongside its
// object, in deterministic (key-sorted) order.
func (o *ExternTypeObligation) MethodKeys() []ExternMethodEntry {
	keys := make([]string, 0, len(o.methods))
	for key := range o.methods {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]ExternMethodEntry, 0, len(keys))
	for _, key := range keys {
		out = append(out, ExternMethodEntry{Key: key, Method: o.methods[key]})
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
