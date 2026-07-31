package namedstruct

import (
	"go/ast"
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

type operationAssembly struct {
	operation    api.NamedStructOperation
	capabilities []*api.GenericOperationContract
}

func EmitAssembly(
	context api.Context,
	children api.ChildEmitter,
	declaration *ast.GenDecl,
	typeName *types.TypeName,
	requirements []api.DeclarationRequirement,
) (api.DeclarationEmission, error) {
	seen := make(map[api.DeclarationRequirement]struct{}, len(requirements))
	operations := make(map[api.NamedStructOperation]operationAssembly)
	demanded := make(map[api.NamedStructOperation]bool)
	var representationRequirements []api.DeclarationRequirement
	for _, requirement := range requirements {
		if _, duplicate := seen[requirement]; duplicate {
			return api.DeclarationEmission{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "named struct received a duplicate declaration requirement",
			}
		}
		seen[requirement] = struct{}{}
		if requirement.Kind() ==
			api.DeclarationRequirementGenericRepresentation {
			owner, _, _, ok := requirement.GenericRepresentation()
			if !ok || owner != typeName {
				return api.DeclarationEmission{}, &api.InvariantError{
					Role:   context.Role(),
					Reason: "named struct received a foreign generic representation",
				}
			}
			representationRequirements = append(
				representationRequirements,
				requirement,
			)
			continue
		}
		if owner, operation, ok := requirement.NamedStructOperation(); ok {
			if owner != typeName {
				return api.DeclarationEmission{}, &api.InvariantError{
					Role:   context.Role(),
					Reason: "named struct received a foreign operation",
				}
			}
			selected := operations[operation]
			selected.operation = operation
			operations[operation] = selected
			demanded[operation] = true
			continue
		}
		owner, capability, ok := requirement.GenericOperation()
		if !ok || owner != typeName {
			return api.DeclarationEmission{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "named struct received a foreign declaration requirement",
			}
		}
		operation, ok := capability.Consumer().NamedStructOperation()
		if !ok {
			return api.DeclarationEmission{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "named struct received a non-member generic operation",
			}
		}
		selected := operations[operation]
		selected.operation = operation
		selected.capabilities = append(selected.capabilities, capability)
		operations[operation] = selected
	}
	ordered := make([]operationAssembly, 0, len(operations))
	for operation, selected := range operations {
		if !demanded[operation] {
			return api.DeclarationEmission{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "generic member operation has no owning struct operation",
			}
		}
		sort.Slice(selected.capabilities, func(left, right int) bool {
			return selected.capabilities[left].Key() <
				selected.capabilities[right].Key()
		})
		ordered = append(ordered, selected)
	}
	sort.Slice(ordered, func(left, right int) bool {
		return ordered[left].operation < ordered[right].operation
	})
	return emitClass(
		context,
		children,
		declaration,
		typeName,
		ordered,
		representationRequirements,
	)
}
