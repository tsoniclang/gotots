package namedstruct

import (
	"go/ast"
	"go/types"
	"slices"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func EmitAssembly(
	context api.Context,
	children api.ChildEmitter,
	declaration *ast.GenDecl,
	typeName *types.TypeName,
	requirements []api.DeclarationRequirement,
) (api.DeclarationEmission, error) {
	ordered := slices.Clone(requirements)
	seen := make(map[api.DeclarationRequirement]struct{}, len(ordered))
	for _, requirement := range ordered {
		owner, _, ok := requirement.NamedStructOperation()
		if !ok || owner != typeName {
			return api.DeclarationEmission{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "named struct received a foreign declaration requirement",
			}
		}
		if _, duplicate := seen[requirement]; duplicate {
			return api.DeclarationEmission{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "named struct received a duplicate declaration requirement",
			}
		}
		seen[requirement] = struct{}{}
	}
	sort.Slice(ordered, func(left, right int) bool {
		_, leftOperation, _ := ordered[left].NamedStructOperation()
		_, rightOperation, _ := ordered[right].NamedStructOperation()
		return leftOperation < rightOperation
	})
	operations := make([]api.NamedStructOperation, 0, len(ordered))
	for _, requirement := range ordered {
		_, operation, _ := requirement.NamedStructOperation()
		operations = append(operations, operation)
	}
	return emitClass(context, children, declaration, typeName, operations)
}
