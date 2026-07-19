// Canonical binding identity: every local, parameter, receiver, named
// result, range variable, and type-switch binding is the SAME Go object
// across its declaration and all uses (go/types guarantees info.Defs and
// info.Uses return one *types.Var, captures included). Two source
// variables that share a spelling in nested scopes are DISTINCT objects
// and must never collapse to one TypeScript identifier. This file assigns
// each object a deterministic BindingID (a source-order ordinal, never a
// pointer value) and allocates a unique readable emission name per
// identity — the outer i stays "i", an inner shadow becomes "i$1" — so
// every reference and every variable-indexed plan keys by identity, not
// spelling. Source spelling survives only as display/debug metadata.
package ir

import (
	"go/ast"
	"go/token"
	"go/types"
	"strconv"

	"github.com/tsoniclang/gotots/internal/tsident"
)

// BindingID is a per-top-level-function ordinal identifying one binding.
// It is assigned in a single deterministic source-order walk (stable
// across runs); -1 marks "no binding" (package-level vars, blanks).
type BindingID int

const noBinding BindingID = -1

// bindingInfo is the provenance of one binding. spelling is the source
// name (display/debug only); pos is the declaration position; synthetic
// marks a compiler-minted binding with no source *types.Var.
type bindingInfo struct {
	spelling  string
	pos       token.Pos
	synthetic bool
}

// bindings is one top-level function's identity table and name allocator.
// ids unifies a declaration with all its uses; info is indexed by
// BindingID (== assignment order); names holds the allocated emission
// name per BindingID once allocate has run.
type bindings struct {
	ids    map[*types.Var]BindingID
	info   []bindingInfo
	names  []string
	used   map[string]bool
	shadow map[string]int
}

func newBindings() *bindings {
	return &bindings{ids: map[*types.Var]BindingID{}, used: map[string]bool{}, shadow: map[string]int{}}
}

// newVar assigns (or returns) the id of a source *types.Var.
func (b *bindings) newVar(v *types.Var, spelling string) BindingID {
	if id, ok := b.ids[v]; ok {
		return id
	}
	id := BindingID(len(b.info))
	b.ids[v] = id
	b.info = append(b.info, bindingInfo{spelling: spelling, pos: v.Pos()})
	return id
}

// newSynthetic mints a fresh id for a compiler-generated binding that has
// no source *types.Var (an unnamed parameter, a defer-capture temporary,
// an adapter parameter). The spelling is already a "$"-bearing name no Go
// source can spell, so it never shadows.
func (b *bindings) newSynthetic(spelling string, pos token.Pos) BindingID {
	id := BindingID(len(b.info))
	b.info = append(b.info, bindingInfo{spelling: spelling, pos: pos, synthetic: true})
	return id
}

// idOf returns the id of an already-registered source var.
func (b *bindings) idOf(v *types.Var) (BindingID, bool) {
	id, ok := b.ids[v]
	return id, ok
}

// name returns the allocated emission name of a binding.
func (b *bindings) name(id BindingID) string {
	if id < 0 || int(id) >= len(b.names) {
		return ""
	}
	return b.names[id]
}

// allocate assigns each BindingID a unique readable emission name, in
// ascending id order so the first holder of a spelling keeps the bare
// name and later shadows take numeric suffixes. reservedSeeds are names
// already claimed by the surrounding emission (generic zero$X/eq$X
// factory parameters), which share the value namespace. Reserved words
// fold through tsBinding first, then shadow-probe resolves any clash;
// generated temporaries and cell/label namespaces are disjoint by grammar
// (they end in "$" or live in a separate namespace), so nothing else
// needs seeding.
func (b *bindings) allocate(reservedSeeds []string) {
	for _, s := range reservedSeeds {
		b.used[s] = true
	}
	b.names = make([]string, len(b.info))
	for id := range b.info {
		bi := b.info[id]
		base := bi.spelling
		if !bi.synthetic {
			base = tsBinding(base)
		}
		cand := base
		for b.used[cand] {
			b.shadow[base]++
			cand = base + "$" + strconv.Itoa(b.shadow[base])
		}
		b.used[cand] = true
		b.used[cand+"$b"] = true // reserve the boxed-cell form
		b.names[id] = cand
	}
	// Machine invariant: alpha renaming is injective on names — every
	// identity holds exactly one name and no two identities share one.
	seen := make(map[string]BindingID, len(b.names))
	for id, name := range b.names {
		if prior, dup := seen[name]; dup {
			panic(bindingInvariant("alpha name " + name + " assigned to two identities " +
				strconv.Itoa(int(prior)) + " and " + strconv.Itoa(id)))
		}
		seen[name] = BindingID(id)
	}
}

// tsBinding folds a reserved/hazardous source spelling to its escaped form
// through the single authoritative identifier policy.
func tsBinding(name string) string { return tsident.Escape(name) }

// assignBindings runs the deterministic pre-pass for one top-level scope
// (a function declaration, or a package-level variable initializer that may
// itself contain function literals): a single source-order walk of the
// whole node (nested function literals included, so a captured outer
// variable already has its id before any closure reads it) assigns each
// source *types.Var a BindingID, then the allocator gives each id a unique
// emission name. Type parameters seed the value namespace (their
// zero$X/eq$X factory parameters are real emitted parameters).
func (b *builder) assignBindings(root ast.Node, typeParams []string) {
	b.bind = newBindings()
	ast.Inspect(root, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.TypeSwitchStmt:
			b.assignTypeSwitchBinding(node)
		case *ast.Ident:
			if node.Name == "_" {
				return true
			}
			if variable, ok := b.info.Defs[node].(*types.Var); ok && isLocalBinding(variable) {
				b.bind.newVar(variable, node.Name)
			}
		}
		return true
	})
	var seeds []string
	for _, tp := range typeParams {
		seeds = append(seeds, "zero$"+tp, "eq$"+tp, "clone$"+tp, "set$"+tp)
	}
	b.bind.allocate(seeds)
	// Fail closed if any local binding the type checker records is missing
	// from the table (an omitted binding kind, never a silent fallback).
	b.validateBindings(root)
}

// assignTypeSwitchBinding registers a type switch's clause bindings. In
// `switch x := y.(type)` there is no single object x: each case clause has
// its own implicit variable (info.Implicits), a genuinely distinct Go
// object in a disjoint scope. Each gets its OWN BindingID and unique name,
// so the alpha renaming stays injective (one Go variable per identity) and
// function-wide analyses never merge separate clause variables. The guard
// identifier, if it has its own definition, is registered too.
func (b *builder) assignTypeSwitchBinding(node *ast.TypeSwitchStmt) {
	assign, ok := node.Assign.(*ast.AssignStmt)
	if !ok {
		return
	}
	bindIdent, ok := assign.Lhs[0].(*ast.Ident)
	if !ok || bindIdent.Name == "_" {
		return
	}
	if guardVar, ok := b.info.Defs[bindIdent].(*types.Var); ok {
		b.bind.newVar(guardVar, bindIdent.Name)
	}
	for _, clauseStmt := range node.Body.List {
		clause, ok := clauseStmt.(*ast.CaseClause)
		if !ok {
			continue
		}
		if variable, ok := b.info.Implicits[clause].(*types.Var); ok {
			b.bind.newVar(variable, bindIdent.Name)
		}
	}
}

// typeSwitchClauseImplicit resolves one case clause's implicit binding
// variable, when the switch has a guard.
func (b *builder) typeSwitchClauseImplicit(clause *ast.CaseClause) (*types.Var, bool) {
	variable, ok := b.info.Implicits[clause].(*types.Var)
	return variable, ok
}

// bindNameOf returns the allocated emission name for an identifier that
// declares or reads a local binding. It resolves the identifier to its
// canonical *types.Var (a definition or a use — reused := targets and
// every reference resolve to the same object) and returns that binding's
// unique name. A package-level variable, blank, or unresolved identifier
// keeps its escaped source spelling (those never participate in local
// shadowing).
func (b *builder) bindNameOf(ident *ast.Ident) string {
	variable := b.identVar(ident)
	if variable == nil {
		panic(bindingInvariant("identifier " + ident.Name + " resolves to no variable"))
	}
	return b.bindNameVar(variable, ident.Name)
}

// bindNameVar returns a LOCAL binding's unique emission name from its
// *types.Var directly (receiver, parameter, named-result, and type-switch
// clause sites hold the object without an identifier at hand). It fails
// closed: reaching it with no binding table, or with a variable the
// deterministic prepass did not register, is a construction-invariant
// violation (a compiler defect that could silently recreate the shadowing
// bug), never a silent fall back to the raw source spelling. Callers must
// classify package variables, blanks, fields, and non-variable
// identifiers before reaching here.
func (b *builder) bindNameVar(variable *types.Var, spelling string) string {
	if b.bind == nil {
		panic(bindingInvariant("binding " + spelling + " reached with no binding table (missing prepass)"))
	}
	id, ok := b.bind.idOf(variable)
	if !ok {
		panic(bindingInvariant("local binding " + spelling + " absent from the binding table"))
	}
	return b.bind.name(id)
}

// identVar resolves an identifier to the canonical *types.Var it defines
// or uses.
func (b *builder) identVar(ident *ast.Ident) *types.Var {
	if variable, ok := b.info.Defs[ident].(*types.Var); ok {
		return variable
	}
	if variable, ok := b.info.Uses[ident].(*types.Var); ok {
		return variable
	}
	return nil
}

// bindingInvariant formats a fail-closed binding-invariant panic value: a
// local binding the deterministic prepass must have registered is missing
// or unclassified, so generation stops rather than risk a name collision.
func bindingInvariant(detail string) string {
	return "GOTOTS_BINDING_INVARIANT: " + detail
}

// isLocalBinding reports whether a *types.Var is a function-local binding
// (a local, parameter, receiver, named result, range or type-switch
// variable, or capture) — as opposed to a package-level variable or a
// struct field — so validation and name allocation consider exactly the
// objects the allocator owns.
func isLocalBinding(v *types.Var) bool {
	if v == nil || v.IsField() || v.Pkg() == nil {
		return false
	}
	return v.Parent() != v.Pkg().Scope()
}

// validateBindings fails closed if any local binding the type checker
// records for this scope — every Defs/Uses variable and every type-switch
// clause implicit, captures included — is absent from the allocated table.
// It makes an omitted binding kind an immediate invariant error instead of
// a silent spelling fallback that could recreate the shadowing defect.
func (b *builder) validateBindings(root ast.Node) {
	assertRegistered := func(v *types.Var, where string) {
		if !isLocalBinding(v) {
			return
		}
		if _, ok := b.bind.idOf(v); !ok {
			panic(bindingInvariant("local binding " + v.Name() + " (" + where + ") absent from the binding table"))
		}
	}
	ast.Inspect(root, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.Ident:
			if node.Name == "_" {
				return true
			}
			if v, ok := b.info.Defs[node].(*types.Var); ok {
				assertRegistered(v, "def")
			}
			if v, ok := b.info.Uses[node].(*types.Var); ok {
				assertRegistered(v, "use")
			}
		case *ast.TypeSwitchStmt:
			for _, clauseStmt := range node.Body.List {
				if clause, ok := clauseStmt.(*ast.CaseClause); ok {
					if v, ok := b.info.Implicits[clause].(*types.Var); ok {
						assertRegistered(v, "type-switch clause")
					}
				}
			}
		}
		return true
	})
}
