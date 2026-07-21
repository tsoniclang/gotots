// Object-model recovery for one translation unit (ADR-0012): the sealed
// whole-program named-type set is analyzed once, after every package's
// types are loaded and before any emitter runs, so the class/method/
// dispatch emitters consume one immutable plan.
package translate

import (
	"go/types"
	"sort"

	"golang.org/x/tools/go/packages"

	"github.com/tsoniclang/gotots/internal/objectmodel"
)

// buildObjectModel recovers the object model from every production
// package's package-level named types. Recognition is by typed facts
// (a self-reference field plus embedding, per ADR-0012), so a unit with
// no object-model family yields an empty plan and the emitter keeps the
// flat-class representation. The named-type set is deterministic:
// packages arrive pre-sorted and each package's scope names are sorted.
func buildObjectModel(pkgs []*packages.Package) *objectmodel.Plan {
	var named []*types.Named
	for _, p := range pkgs {
		if p.Types == nil {
			continue
		}
		scope := p.Types.Scope()
		names := scope.Names()
		sort.Strings(names)
		for _, name := range names {
			obj, ok := scope.Lookup(name).(*types.TypeName)
			if !ok || obj.IsAlias() {
				continue
			}
			if n, ok := types.Unalias(obj.Type()).(*types.Named); ok {
				named = append(named, n)
			}
		}
	}
	return objectmodel.Analyze(named)
}
