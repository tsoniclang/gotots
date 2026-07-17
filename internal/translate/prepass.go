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
func collectGenericInstances(unit ir.Scope, pkgs []*packages.Package) {
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
		for ident, instance := range p.TypesInfo.Instances {
			args := make([]types.Type, 0, instance.TypeArgs.Len())
			for i := range instance.TypeArgs.Len() {
				args = append(args, instance.TypeArgs.At(i))
			}
			switch object := p.TypesInfo.Uses[ident].(type) {
			case *types.Func:
				unit.AddGenericInstance(object, args)
			case *types.TypeName:
				unit.AddGenericTypeInstance(object, args)
			}
		}
	}
	collectAddressTakenFields(unit, pkgs)
	freezeExternUniverse(unit, pkgs)
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
	consider := func(t types.Type) {
		if t == nil {
			return
		}
		u := types.Unalias(t)
		if pointer, ok := u.(*types.Pointer); ok {
			u = types.Unalias(pointer.Elem())
		}
		named, ok := u.(*types.Named)
		if !ok || named.Obj().Pkg() == nil || unit.Owns(named.Obj().Pkg().Path()) {
			return
		}
		if named.TypeParams() != nil && named.TypeParams().Len() > 0 {
			return
		}
		if _, isIface := named.Underlying().(*types.Interface); isIface {
			return
		}
		candidates[named.Obj().Pkg().Path()+"."+named.Obj().Name()] = named.Obj()
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
func collectAddressTakenFields(unit ir.Scope, pkgs []*packages.Package) {
	for _, p := range pkgs {
		for _, file := range p.Syntax {
			ast.Inspect(file, func(node ast.Node) bool {
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
				if key, ok := ir.FieldStorageKeyOfSelection(selection); ok {
					unit.MarkFieldAddressTaken(key)
				}
				return true
			})
		}
	}
}

// collectGenericInstances records every generic-function instantiation
// across the unit: the closed-world evidence that admits generic
// declarations.
