// Whole-unit pre-passes that freeze the translation universe before any
// body is built: the generic-instantiation and concrete-type evidence,
// the external-implementer universe, and the address-taken-field set.
package translate

import (
	"go/ast"
	"go/token"
	"go/types"
	"sort"

	"golang.org/x/tools/go/packages"

	"github.com/tsoniclang/gotots/internal/ir"
)

// collectGenericInstances records every generic-function instantiation
// across the unit: the closed-world evidence that admits generic
// declarations.
func collectGenericInstances(unit ir.Scope, pkgs []*packages.Package) error {
	// The dynamic-type universe: every named type declared in an owned
	// package, the closed-world set interface dispatch resolves over.
	for _, p := range pkgs {
		scope := p.Types.Scope()
		for _, name := range scope.Names() {
			if typeName, ok := scope.Lookup(name).(*types.TypeName); ok && !typeName.IsAlias() {
				if _, isNamed := typeName.Type().(*types.Named); isNamed {
					unit.AddConcreteType(typeName)
				}
			}
		}
	}
	for _, p := range pkgs {
		enclosing := genericDeclRanges(p)
		for ident, instance := range p.TypesInfo.Instances {
			args := make([]types.Type, 0, instance.TypeArgs.Len())
			for i := range instance.TypeArgs.Len() {
				args = append(args, instance.TypeArgs.At(i))
			}
			var inner types.Object
			switch object := p.TypesInfo.Uses[ident].(type) {
			case *types.Func:
				unit.AddGenericInstance(object, args)
				inner = object
			case *types.TypeName:
				unit.AddGenericTypeInstance(object, args)
				inner = object
			default:
				continue
			}
			// A vector mentioning type parameters is a FREE-parameter
			// instantiation inside an enclosing generic declaration: record
			// the edge so the closure can derive its concretizations.
			free := false
			for _, arg := range args {
				if ir.MentionsTypeParam(arg) {
					free = true
					break
				}
			}
			if !free {
				continue
			}
			if outer := enclosing.at(ident.Pos()); outer != nil {
				unit.AddGenericEdge(outer, inner, args)
			}
		}
	}
	// Close the evidence under substitution BEFORE anything consults it:
	// free-parameter vectors are then soundly skippable everywhere, their
	// concretizations being present.
	unit.CloseGenericEvidence()
	if err := collectAddressTakenFields(unit, pkgs); err != nil {
		return err
	}
	freezeExternUniverse(unit, pkgs)
	// Every named-type-universe contributor has now run (owned concrete
	// types, external freeze). Seal the universe: from here it is immutable,
	// so every interface union resolves over the complete closed world and
	// no later body can introduce an implementer a cached union already
	// missed.
	unit.SealUniverse()
	return nil
}

// freezeExternUniverse records, before any interface union is resolved,
// every external named type referenced anywhere in the unit's bodies.
// External implementers were otherwise registered lazily at box sites
// during body building, so a union resolved (and cached) by an early body
// could omit an external implementer first boxed in a later body. The
// whole-unit set is a superset of the boxed set; ifaceMembers admits only
// those an interface's method set proves it implements (types.Implements),
// so the closed universe is complete and order-independent — the two-phase
// interface/external-universe collection the closed union depends on.
func freezeExternUniverse(unit ir.Scope, pkgs []*packages.Package) {
	// The candidate types come from go/types maps (random iteration order),
	// so collect them by canonical identity and register in SORTED order:
	// the universe set is identical every run and its registration order is
	// deterministic, so all downstream evidence stays byte-stable.
	candidates := map[string]*types.TypeName{}
	// consider walks a type's full structure — through pointers, containers,
	// tuples, signatures, struct fields, and generic type arguments — and
	// records every external, non-generic, non-interface NAMED type it
	// reaches. A box site's source type is always a component of some
	// expression's or used object's type (an owned function's result is a
	// type-expression in its own AST; an EXTERNAL function's result appears
	// only inside its signature, reached here by descent), so the frozen set
	// is a superset of everything any body can box. Freezing the complete
	// set in this pre-pass — before any interface union is resolved and
	// cached — is what seals the universe: box sites never introduce an
	// external implementer a cached union could already have missed.
	seen := map[types.Type]bool{}
	var consider func(t types.Type)
	consider = func(t types.Type) {
		if t == nil || seen[t] {
			return
		}
		seen[t] = true
		switch u := types.Unalias(t).(type) {
		case *types.Named:
			// A named type's own type arguments carry distinct dynamic
			// identities (List[extern] surfaces extern); descend before the
			// generic/interface guards so nested externals are never lost.
			if args := u.TypeArgs(); args != nil {
				for i := range args.Len() {
					consider(args.At(i))
				}
			}
			if u.Obj().Pkg() != nil && !unit.Owns(u.Obj().Pkg().Path()) &&
				(u.TypeParams() == nil || u.TypeParams().Len() == 0) {
				if _, isIface := u.Underlying().(*types.Interface); !isIface {
					candidates[u.Obj().Pkg().Path()+"."+u.Obj().Name()] = u.Obj()
				}
			}
			// The underlying structure (struct fields, embedded types) may
			// name further externals reachable only through this type.
			consider(u.Underlying())
		case *types.Pointer:
			consider(u.Elem())
		case *types.Slice:
			consider(u.Elem())
		case *types.Array:
			consider(u.Elem())
		case *types.Chan:
			consider(u.Elem())
		case *types.Map:
			consider(u.Key())
			consider(u.Elem())
		case *types.Struct:
			for i := range u.NumFields() {
				consider(u.Field(i).Type())
			}
		case *types.Tuple:
			for i := range u.Len() {
				consider(u.At(i).Type())
			}
		case *types.Signature:
			consider(u.Params())
			consider(u.Results())
		case *types.Interface:
			// An interface is not a boxable dynamic type; its method
			// signatures reference no additional boxable concrete carriers
			// that a concrete value would not itself surface.
		}
	}
	for _, p := range pkgs {
		for _, tv := range p.TypesInfo.Types {
			consider(tv.Type)
		}
		for _, obj := range p.TypesInfo.Uses {
			if obj != nil {
				consider(obj.Type())
			}
		}
	}
	ids := make([]string, 0, len(candidates))
	for id := range candidates {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		unit.AddExternConcrete(candidates[id])
	}
}

// collectAddressTakenFields records, across the whole unit, every struct
// field whose address is taken (&x.f). This is inherently whole-program:
// a field addressed in one body fixes that field's representation (a
// stable per-instance cell) in every body, so the scan must complete
// before any struct or body is built.
func collectAddressTakenFields(unit ir.Scope, pkgs []*packages.Package) error {
	var walkErr error
	for _, p := range pkgs {
		for _, file := range p.Syntax {
			ast.Inspect(file, func(node ast.Node) bool {
				if walkErr != nil {
					return false
				}
				unary, ok := node.(*ast.UnaryExpr)
				if !ok || unary.Op != token.AND {
					return true
				}
				selector, ok := ast.Unparen(unary.X).(*ast.SelectorExpr)
				if !ok {
					return true
				}
				selection, ok := p.TypesInfo.Selections[selector]
				if !ok || selection.Kind() != types.FieldVal {
					return true
				}
				key, ok, err := ir.FieldStorageKeyOfSelection(selection)
				if err != nil {
					walkErr = err
					return false
				}
				if ok {
					unit.MarkFieldAddressTaken(key)
				}
				return true
			})
			if walkErr != nil {
				return walkErr
			}
		}
	}
	return nil
}

// collectGenericInstances records every generic-function instantiation
// across the unit: the closed-world evidence that admits generic
// declarations.

// genericRanges locates the generic declarations of one package by source
// range, so an instantiation identifier resolves to its innermost
// enclosing generic declaration (the owner of the free type parameters
// its arguments mention).
type genericRanges struct {
	entries []genericRange
}

type genericRange struct {
	pos, end token.Pos
	obj      types.Object
}

// at returns the innermost generic declaration containing pos.
func (g genericRanges) at(pos token.Pos) types.Object {
	var best *genericRange
	for i := range g.entries {
		entry := &g.entries[i]
		if pos < entry.pos || pos >= entry.end {
			continue
		}
		if best == nil || (entry.pos >= best.pos && entry.end <= best.end) {
			best = entry
		}
	}
	if best == nil {
		return nil
	}
	return best.obj
}

// genericDeclRanges collects every generic function declaration (own or
// receiver type parameters) and generic type declaration of the package.
func genericDeclRanges(p *packages.Package) genericRanges {
	var out genericRanges
	for _, file := range p.Syntax {
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				object, ok := p.TypesInfo.Defs[d.Name].(*types.Func)
				if !ok {
					continue
				}
				signature := object.Type().(*types.Signature)
				generic := signature.TypeParams() != nil && signature.TypeParams().Len() > 0
				if recv := signature.RecvTypeParams(); recv != nil && recv.Len() > 0 {
					generic = true
				}
				if generic {
					out.entries = append(out.entries, genericRange{pos: d.Pos(), end: d.End(), obj: object})
				}
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok || typeSpec.TypeParams == nil || len(typeSpec.TypeParams.List) == 0 {
						continue
					}
					if object, ok := p.TypesInfo.Defs[typeSpec.Name].(*types.TypeName); ok {
						out.entries = append(out.entries, genericRange{pos: typeSpec.Pos(), end: typeSpec.End(), obj: object})
					}
				}
			}
		}
	}
	return out
}
