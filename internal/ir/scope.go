// Scope construction and its evidence-recording methods: universe
// sealing, generic instance registries, anonymous-struct identity, and
// external bindings — the mutable corpus-level stores behind the
// requirement propagation in genericclosure.go.
package ir

import (
	"fmt"
	"go/types"
	"sort"
)

// SealUniverse finalizes the closed-world named-type universe: after this
// the dynamic-type set is immutable and any AddConcreteType/AddExternConcrete
// panics. It is called once, at the end of the whole-unit pre-pass, before
// any body builds and any interface union resolves.
func (s Scope) SealUniverse() { *s.universeSealed = true }

// UniverseSealed reports whether the named-type universe is finalized.
func (s Scope) UniverseSealed() bool { return *s.universeSealed }

// NewScope builds a unit scope over the given package paths.
func NewScope(paths ...string) Scope {
	packages := make(map[string]bool, len(paths))
	for _, path := range paths {
		packages[path] = true
	}
	return Scope{
		packages:           packages,
		generics:           map[*types.Func][][]types.Type{},
		externals:          map[*types.Func]bool{},
		typeGenerics:       map[*types.TypeName][][]types.Type{},
		anonStructs:        map[string]map[string]*Struct{},
		externTypes:        map[string]*ExternTypeObligation{},
		externVars:         map[string]*types.Var{},
		concreteTypes:      &[]*types.TypeName{},
		externConcrete:     &[]*types.TypeName{},
		boxedComposites:    map[string]*boxedComposite{},
		valueBoxedNamed:    map[string]bool{},
		ifaceMembers:       map[string][]IfaceMember{},
		instCandidates:     new([]InstCandidate),
		instSlotsCache:     map[string]instSlotsEntry{},
		addressTakenFields: map[string]bool{},
		genericEdges:       &[]genericEdge{},
		paramKeyReqs:       map[string][]bool{},
		paramCaptureReqs:   map[string][]bool{},
		paramPtrReqs:       map[string][]bool{},
		universeSealed:     new(bool),
	}
}

// MarkFieldAddressTaken records that the address of the field with the
// given storage key is taken somewhere in the unit (the pre-pass calls
// this with FieldStorageKeyOfSelection).
func (s Scope) MarkFieldAddressTaken(key string) {
	if key != "" {
		s.addressTakenFields[key] = true
	}
}

// FieldAddressTaken reports whether the field with the given storage key
// has its address taken anywhere in the unit — the condition (with a
// non-identity field type) for the stable-cell field representation.
func (s Scope) FieldAddressTaken(key string) bool {
	return key != "" && s.addressTakenFields[key]
}

// Owns reports whether the package path is part of the unit.
func (s Scope) Owns(path string) bool { return s.packages[path] }

// Paths returns every unit package path (unordered).
func (s Scope) Paths() []string {
	paths := make([]string, 0, len(s.packages))
	for path := range s.packages {
		paths = append(paths, path)
	}
	return paths
}

// AddGenericInstance records one instantiation of a generic function.
func (s Scope) AddGenericInstance(fn *types.Func, typeArgs []types.Type) {
	s.generics[fn] = append(s.generics[fn], typeArgs)
}

// GenericInstances returns every recorded instantiation of fn.
func (s Scope) GenericInstances(fn *types.Func) [][]types.Type { return s.generics[fn] }

// AddGenericTypeInstance records one instantiation of a generic named
// type.
func (s Scope) AddGenericTypeInstance(name *types.TypeName, typeArgs []types.Type) {
	s.typeGenerics[name] = append(s.typeGenerics[name], typeArgs)
}

// GenericTypeInstances returns every recorded instantiation of a
// generic named type.
func (s Scope) GenericTypeInstances(name *types.TypeName) [][]types.Type { return s.typeGenerics[name] }

// GenericTypeObjects returns every generic named type with recorded
// instantiation evidence, sorted by canonical object identity so
// enumeration order is deterministic.
func (s Scope) GenericTypeObjects() []*types.TypeName {
	out := make([]*types.TypeName, 0, len(s.typeGenerics))
	for name := range s.typeGenerics {
		out = append(out, name)
	}
	sort.Slice(out, func(i, j int) bool {
		return objKey(out[i]) < objKey(out[j])
	})
	return out
}

// RegisterAnonStruct records one synthesized anonymous-struct class for
// the package's module (idempotent per shape).
func (s Scope) RegisterAnonStruct(pkg string, decl *Struct) error {
	if s.anonStructs[pkg] == nil {
		s.anonStructs[pkg] = map[string]*Struct{}
	}
	if existing, ok := s.anonStructs[pkg][decl.Name]; ok && existing.Identity != decl.Identity {
		// A digest collision between distinct anonymous shapes must never
		// silently overwrite: the identity is not injective, so fail
		// closed rather than mis-dispatch one shape as the other.
		return fmt.Errorf("anonymous struct identity collision on %s: %q vs %q", decl.Name, existing.Identity, decl.Identity)
	}
	s.anonStructs[pkg][decl.Name] = decl
	return nil
}

// AnonStructs returns the synthesized anonymous-struct classes of one
// package in sorted name order.
func (s Scope) AnonStructs(pkg string) []*Struct {
	names := make([]string, 0, len(s.anonStructs[pkg]))
	for name := range s.anonStructs[pkg] {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]*Struct, 0, len(names))
	for _, name := range names {
		out = append(out, s.anonStructs[pkg][name])
	}
	return out
}

// AddExternalVar records one external package variable the unit reads.
func (s Scope) AddExternalVar(id string, variable *types.Var) { s.externVars[id] = variable }

// ExternalVars returns the referenced external variables in sorted
// identity order.
func (s Scope) ExternalVars() []struct {
	ID       string
	Variable *types.Var
} {
	ids := make([]string, 0, len(s.externVars))
	for id := range s.externVars {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]struct {
		ID       string
		Variable *types.Var
	}, 0, len(ids))
	for _, id := range ids {
		out = append(out, struct {
			ID       string
			Variable *types.Var
		}{ID: id, Variable: s.externVars[id]})
	}
	return out
}

// ExternalFuncs returns every admitted external function contract,
// sorted by package path then name.
func (s Scope) ExternalFuncs() []*types.Func {
	out := make([]*types.Func, 0, len(s.externals))
	for fn := range s.externals {
		out = append(out, fn)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Pkg().Path() != out[j].Pkg().Path() {
			return out[i].Pkg().Path() < out[j].Pkg().Path()
		}
		return out[i].Name() < out[j].Name()
	})
	return out
}
