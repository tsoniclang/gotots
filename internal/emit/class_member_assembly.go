package emit

import (
	"go/types"
	"slices"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type classMemberContribution struct {
	owner   *types.TypeName
	method  *types.Func
	members []tsgo.ClassElement
}

func (s *programSession) artifactTargetSite(
	site declarationSite,
) (declarationSite, error) {
	method, ok := site.object.(*types.Func)
	if !ok || method.Signature().Recv() == nil {
		return site, nil
	}
	owner := api.MethodReceiverTypeName(method)
	target, ok := s.sites[owner]
	if owner == nil || !ok {
		return declarationSite{}, &ScheduleError{
			Object: site.object.Name(),
			Reason: "receiver method has no target class declaration",
		}
	}
	return target, nil
}

func (s *programSession) commitClassMemberContribution(
	owner types.Object,
	contribution *classMemberContribution,
) {
	method, ok := owner.(*types.Func)
	if !ok {
		return
	}
	method = method.Origin()
	if contribution == nil {
		delete(s.classMembers, method)
		return
	}
	s.classMembers[method] = classMemberContribution{
		owner:   contribution.owner,
		method:  contribution.method,
		members: slices.Clone(contribution.members),
	}
}

func (s *programSession) partitionClassMethodRequirements(
	owner types.Object,
	requirements []api.DeclarationRequirement,
) ([]api.DeclarationRequirement, []*types.Func, error) {
	ordinary := make([]api.DeclarationRequirement, 0, len(requirements))
	methods := make([]*types.Func, 0, len(requirements))
	typeName, typeOwner := owner.(*types.TypeName)
	for _, requirement := range requirements {
		if requirement.Kind() != api.DeclarationRequirementClassMethod {
			ordinary = append(ordinary, requirement)
			continue
		}
		selectedOwner, method, ok := requirement.ClassMethod()
		if !ok || !typeOwner || selectedOwner != typeName {
			return nil, nil, &ScheduleError{
				Object: owner.Name(),
				Reason: "class-method requirement has a foreign class owner",
			}
		}
		methods = append(methods, method)
	}
	sort.Slice(methods, func(left, right int) bool {
		return compareObjects(methods[left], methods[right]) < 0
	})
	return ordinary, methods, nil
}

func (s *programSession) classArtifactRequests(
	owner types.Object,
	methods []*types.Func,
	requests []api.RootRequest,
) ([]api.RootRequest, error) {
	if _, ok := owner.(*types.TypeName); !ok {
		if len(methods) != 0 {
			return nil, &ScheduleError{
				Object: owner.Name(),
				Reason: "non-class artifact selected class methods",
			}
		}
		return requests, nil
	}
	result := slices.Clone(requests)
	for _, method := range methods {
		request, err := api.NewArtifactDependencyRequest(
			method,
			api.ArtifactFacetImplementation,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, request)
	}
	return result, nil
}

func (s *programSession) attachClassMemberContributions(
	builder *targetFileBuilder,
	owner types.Object,
	statements []tsgo.Statement,
	methods []*types.Func,
) ([]tsgo.Statement, error) {
	typeName, ok := owner.(*types.TypeName)
	if !ok {
		return nil, &ScheduleError{
			Object: owner.Name(),
			Reason: "class-member attachment owner is not a type",
		}
	}
	className, err := builder.context.Names().Declare(typeName)
	if err != nil {
		return nil, err
	}
	members := make([]tsgo.ClassElement, 0, len(methods))
	for _, method := range methods {
		contribution, ok := s.classMembers[method]
		if !ok ||
			contribution.owner != typeName ||
			contribution.method != method ||
			len(contribution.members) == 0 {
			return nil, &ScheduleError{
				Object: method.Name(),
				Reason: "selected class method has no exact contribution",
			}
		}
		members = append(members, contribution.members...)
	}
	result := slices.Clone(statements)
	found := false
	for index, statement := range result {
		class, ok := statement.(tsgo.ClassDeclaration)
		if !ok || class.Name().Text() != className {
			continue
		}
		if found {
			return nil, &ScheduleError{
				Object: owner.Name(),
				Reason: "target class declaration is duplicated",
			}
		}
		found = true
		targetMembers := append(class.Members(), members...)
		result[index] = s.factory.ClassDeclaration(
			class.Modifiers(),
			class.Name(),
			class.TypeParameters(),
			class.HeritageClauses(),
			targetMembers,
		)
	}
	if !found {
		return nil, &ScheduleError{
			Object: owner.Name(),
			Reason: "class-member owner emitted no target class",
		}
	}
	return result, nil
}
