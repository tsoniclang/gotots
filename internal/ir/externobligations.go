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
		obligation = &ExternTypeObligation{Pkg: pkg, Name: name, Methods: map[string]*types.Func{}}
		s.externTypes[id] = obligation
	}
	return obligation
}

// AddExternalMethod records one referenced method of an external type.
// The stub symbol and dispatch table are keyed by the method's Go name
// (Go forbids one type from exposing two accessible methods of the same
// name), but a same-name record of a DISTINCT method identity — which
// would silently overwrite the first — is flagged so emission fails
// closed rather than dropping a contract member.
func (s Scope) AddExternalMethod(pkg, name string, method *types.Func) {
	obligation := s.AddExternalType(pkg, name)
	if prior, ok := obligation.Methods[method.Name()]; ok && MethodKey(prior) != MethodKey(method) {
		obligation.NameCollisions = append(obligation.NameCollisions, method.Name())
	}
	obligation.Methods[method.Name()] = method
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
