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
}

// reservedTSBindings are identifiers legal in Go but reserved or hazardous
// as bindings in strict-mode ECMAScript modules (globals the generated
// runtime relies on included). A binding whose source spelling is one of
// these gains a "$" suffix; Go identifiers can never contain "$", so an
// escaped name never collides with a source name.
var reservedTSBindings = map[string]bool{
	"arguments": true, "await": true, "catch": true, "class": true,
	"debugger": true, "delete": true, "do": true, "enum": true,
	"eval": true, "export": true, "extends": true, "false": true,
	"finally": true, "function": true, "globalThis": true, "implements": true,
	"in": true, "Infinity": true, "instanceof": true, "let": true,
	"NaN": true, "new": true, "null": true, "private": true,
	"protected": true, "public": true, "static": true, "super": true,
	"this": true, "throw": true, "true": true, "try": true,
	"typeof": true, "undefined": true, "void": true, "while": true,
	"with": true, "yield": true,
}

// tsBinding folds a reserved/hazardous source spelling to its escaped
// form; it is the identity for every ordinary Go identifier.
func tsBinding(name string) string {
	if reservedTSBindings[name] {
		return name + "$"
	}
	return name
}

// share maps one more source var onto an existing binding id (type-switch
// clause bindings, which are disjoint scopes sharing one emission name).
func (b *bindings) share(v *types.Var, id BindingID) { b.ids[v] = id }

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
			variable, ok := b.info.Defs[node].(*types.Var)
			if !ok || node.Name == "_" {
				return true
			}
			if variable.Pkg() != nil && variable.Parent() == variable.Pkg().Scope() {
				return true // package-level variable: routed through module.symbol
			}
			b.bind.newVar(variable, node.Name)
		}
		return true
	})
	var seeds []string
	for _, tp := range typeParams {
		seeds = append(seeds, "zero$"+tp, "eq$"+tp)
	}
	b.bind.allocate(seeds)
}

// assignTypeSwitchBinding registers a type switch's clause bindings: every
// case clause has its own implicit variable (info.Implicits), but the
// clauses are disjoint scopes that emit one shared name, so they share one
// BindingID seeded from the guard spelling.
func (b *builder) assignTypeSwitchBinding(node *ast.TypeSwitchStmt) {
	assign, ok := node.Assign.(*ast.AssignStmt)
	if !ok {
		return
	}
	bindIdent, ok := assign.Lhs[0].(*ast.Ident)
	if !ok || bindIdent.Name == "_" {
		return
	}
	id := noBinding
	for _, clauseStmt := range node.Body.List {
		clause, ok := clauseStmt.(*ast.CaseClause)
		if !ok {
			continue
		}
		variable, ok := b.info.Implicits[clause].(*types.Var)
		if !ok {
			continue
		}
		if id == noBinding {
			id = b.bind.newVar(variable, bindIdent.Name)
		} else {
			b.bind.share(variable, id)
		}
	}
}

// bindNameOf returns the allocated emission name for an identifier that
// declares or reads a local binding. It resolves the identifier to its
// canonical *types.Var (a definition or a use — reused := targets and
// every reference resolve to the same object) and returns that binding's
// unique name. A package-level variable, blank, or unresolved identifier
// keeps its escaped source spelling (those never participate in local
// shadowing).
func (b *builder) bindNameOf(ident *ast.Ident) string {
	if b.bind != nil {
		if variable := b.identVar(ident); variable != nil {
			if id, ok := b.bind.idOf(variable); ok {
				return b.bind.name(id)
			}
		}
	}
	return tsBinding(ident.Name)
}

// bindNameVar returns a binding's unique emission name from its
// *types.Var directly (receiver, parameter, and named-result sites hold
// the object without an identifier at hand).
func (b *builder) bindNameVar(variable *types.Var, fallback string) string {
	if b.bind != nil && variable != nil {
		if id, ok := b.bind.idOf(variable); ok {
			return b.bind.name(id)
		}
	}
	return tsBinding(fallback)
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
