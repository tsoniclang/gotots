package emit

import (
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
	environmentcontract "github.com/tsoniclang/gotots/internal/emit/environmentcontract"
)

func (b *environmentContractBuilder) environmentRequirements(
	object types.Object,
) []api.DeclarationRequirement {
	selected := b.requirements[object]
	requirements := make(
		[]api.DeclarationRequirement,
		0,
		len(selected),
	)
	for requirement := range selected {
		requirements = append(requirements, requirement)
	}
	sort.Slice(requirements, func(left, right int) bool {
		return compareDeclarationRequirements(
			requirements[left],
			requirements[right],
		) < 0
	})
	return requirements
}

func (s *programSession) buildEnvironmentDeclaration(
	builder *environmentContractBuilder,
	object types.Object,
	requirements []api.DeclarationRequirement,
) (environmentDeclaration, error) {
	target, err := environmentcontract.Declaration(
		builder.context,
		builder.emitter,
		object,
		requirements,
	)
	if err != nil {
		return environmentDeclaration{}, environmentContractError(object, err)
	}
	if err := s.applyRootRequests(
		builder.placement,
		target.Requests(),
	); err != nil {
		return environmentDeclaration{}, environmentContractError(object, err)
	}
	return environmentDeclaration{
		object:     object,
		name:       object.Name(),
		statements: target.Declarations(),
	}, nil
}

func (s *programSession) applyEnvironmentCallableControl(
	builder *environmentContractBuilder,
	object types.Object,
	requirement api.DeclarationRequirement,
) error {
	owner, enclosing, callable, control, ok :=
		requirement.CallableControl()
	source, sourceOwned := owner.Source()
	if !ok ||
		!sourceOwned ||
		source != object ||
		enclosing != nil ||
		callable != nil ||
		control != api.CallableControlRecovery {
		return &ScheduleError{
			Object: object.Name(),
			Reason: "environment callable-control requirement is invalid",
		}
	}
	return s.applyEnvironmentDeclarationRequirement(
		builder,
		object,
		requirement,
	)
}

func (s *programSession) applyEnvironmentGenericCallableProfile(
	builder *environmentContractBuilder,
	object types.Object,
	requirement api.DeclarationRequirement,
) error {
	function, callable := object.(*types.Func)
	profile, profiled := requirement.GenericCallableProfile()
	if !callable ||
		!profiled ||
		profile.Owner() != function.Origin() ||
		function != function.Origin() {
		return &ScheduleError{
			Object: object.Name(),
			Reason: "environment generic callable profile requirement is invalid",
		}
	}
	return s.applyEnvironmentDeclarationRequirement(
		builder,
		object,
		requirement,
	)
}

func (s *programSession) applyEnvironmentDeclarationRequirement(
	builder *environmentContractBuilder,
	object types.Object,
	requirement api.DeclarationRequirement,
) error {
	current := builder.requirements[object]
	if _, duplicate := current[requirement]; duplicate {
		return nil
	}
	next := make(
		map[api.DeclarationRequirement]struct{},
		len(current)+1,
	)
	for existing := range current {
		next[existing] = struct{}{}
	}
	next[requirement] = struct{}{}
	builder.requirements[object] = next
	if _, emitted := builder.declarations[object]; !emitted {
		return nil
	}
	target, err := s.buildEnvironmentDeclaration(
		builder,
		object,
		builder.environmentRequirements(object),
	)
	if err != nil {
		builder.requirements[object] = current
		return err
	}
	target.reconstructions =
		builder.declarations[object].reconstructions + 1
	builder.declarations[object] = target
	return nil
}
