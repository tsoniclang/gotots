package maprepresentation

import (
	"go/ast"
	"go/types"
	"slices"

	"github.com/tsoniclang/gotots/internal/emit/api"
	mapruntime "github.com/tsoniclang/gotots/internal/emit/runtime/map"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type Specialization struct {
	members  []tsgo.ClassElement
	requests []api.RootRequest
}

type SpecializationCapabilities struct {
	Clear bool
}

func (s Specialization) Members() []tsgo.ClassElement {
	return slices.Clone(s.members)
}

func (s Specialization) Requests() []api.RootRequest {
	return slices.Clone(s.requests)
}

func BuildSpecialization(
	context api.Context,
	source ast.Node,
	className string,
	mapType *types.Map,
	keyType tsgo.TypeNode,
	valueType tsgo.TypeNode,
	capabilities SpecializationCapabilities,
) (Specialization, error) {
	if className == "" ||
		mapType == nil ||
		keyType == nil ||
		valueType == nil ||
		!types.Comparable(mapType.Key()) ||
		!context.Values().SupportsHash(context, mapType.Key()) {
		return Specialization{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "aggregate map specialization contract is invalid",
		}
	}
	operations, requests, err := specializationOperations(
		context,
		source,
		mapType,
	)
	if err != nil {
		return Specialization{}, err
	}
	memberNames, err := specializationNames()
	if err != nil {
		return Specialization{}, err
	}
	panicReference, err := context.Names().Runtime(
		api.RuntimePanic,
		api.ImportPhaseValue,
	)
	if err != nil {
		return Specialization{}, err
	}
	builder := specializationBuilder{
		factory:   context.Factory(),
		className: className,
		keyType:   keyType,
		valueType: valueType,
		panicName: panicReference.Name(),
		zero:      operations.zero,
		hash:      operations.hash,
		equal:     operations.equal,
		copyKey:   operations.copyKey,
		copyValue: operations.copyValue,
		members:   memberNames,
		clear:     capabilities.Clear,
	}
	members := builder.build()
	if err := validateSpecialization(
		context.Role(),
		members,
		capabilities,
	); err != nil {
		return Specialization{}, err
	}
	return Specialization{
		members:  members,
		requests: api.CombineRequests(requests, panicReference.Requests()),
	}, nil
}

func CapabilitiesFromRequirements(
	role api.Role,
	artifact *api.GeneratedArtifact,
	requirements []api.DeclarationRequirement,
) (SpecializationCapabilities, error) {
	if !artifact.Valid() ||
		artifact.Kind() != api.GeneratedArtifactMapSpecialization {
		return SpecializationCapabilities{}, &api.InvariantError{
			Role:   role,
			Reason: "map specialization capability owner is invalid",
		}
	}
	var capabilities SpecializationCapabilities
	for _, requirement := range requirements {
		selected, demand, valid := requirement.MapSpecialization()
		if !valid || selected != artifact {
			return SpecializationCapabilities{}, &api.InvariantError{
				Role:   role,
				Reason: "map specialization received a foreign requirement",
			}
		}
		if demand == api.MapSpecializationDemandClear {
			capabilities.Clear = true
		}
	}
	return capabilities, nil
}

func specializationNames() (specializationMemberNames, error) {
	resolved := make([]string, 0, mapruntime.MemberClear)
	for member := mapruntime.MemberNil; member <= mapruntime.MemberClear; member++ {
		name, err := mapruntime.Name(member)
		if err != nil {
			return specializationMemberNames{}, err
		}
		resolved = append(resolved, name)
	}
	return specializationMemberNames{
		nilMember:    resolved[0],
		makeMember:   resolved[1],
		lookup:       resolved[2],
		lookupOK:     resolved[3],
		store:        resolved[4],
		deleteMember: resolved[5],
		length:       resolved[6],
		isNil:        resolved[7],
		clear:        resolved[8],
	}, nil
}

type specializationOperationSet struct {
	zero      operationBody
	hash      operationBody
	equal     operationBody
	copyKey   operationBody
	copyValue operationBody
}

func specializationOperations(
	context api.Context,
	source ast.Node,
	mapType *types.Map,
) (specializationOperationSet, []api.RootRequest, error) {
	zero, err := context.Values().Zero(
		context.WithRole(api.RoleMapValue),
		source,
		mapType.Elem(),
	)
	if err != nil {
		return specializationOperationSet{}, nil, err
	}
	key := context.Factory().Identifier("$key")
	hash, err := context.Values().Hash(
		context.WithRole(api.RoleMapKey),
		source,
		mapType.Key(),
		key,
	)
	if err != nil {
		return specializationOperationSet{}, nil, err
	}
	left := context.Factory().Identifier("$left")
	right := context.Factory().Identifier("$right")
	equal, err := context.Values().Equal(
		context.WithRole(api.RoleMapKey),
		source,
		mapType.Key(),
		left,
		right,
	)
	if err != nil {
		return specializationOperationSet{}, nil, err
	}
	copyKey, err := context.Values().Copy(
		context.WithRole(api.RoleMapKey),
		nil,
		mapType.Key(),
		api.DirectExpression(key),
	)
	if err != nil {
		return specializationOperationSet{}, nil, err
	}
	value := context.Factory().Identifier("$value")
	copyValue, err := context.Values().Copy(
		context.WithRole(api.RoleMapValue),
		nil,
		mapType.Elem(),
		api.DirectExpression(value),
	)
	if err != nil {
		return specializationOperationSet{}, nil, err
	}
	return specializationOperationSet{
			zero:      operation(zero),
			hash:      operation(hash),
			equal:     operation(equal),
			copyKey:   operation(copyKey),
			copyValue: operation(copyValue),
		}, api.CombineRequests(
			zero.Requests(),
			hash.Requests(),
			equal.Requests(),
			copyKey.Requests(),
			copyValue.Requests(),
		), nil
}

func operation(emission api.ExpressionEmission) operationBody {
	return operationBody{
		before: emission.Before(),
		value:  emission.Value(),
	}
}
