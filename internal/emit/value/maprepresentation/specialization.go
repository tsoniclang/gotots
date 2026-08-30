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
	heritage []tsgo.HeritageClause
	members  []tsgo.ClassElement
	requests []api.RootRequest
}

func (s Specialization) HeritageClauses() []tsgo.HeritageClause {
	return slices.Clone(s.heritage)
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
	storageKeyType tsgo.TypeNode,
	valueType tsgo.TypeNode,
) (Specialization, error) {
	if className == "" ||
		mapType == nil ||
		keyType == nil ||
		storageKeyType == nil ||
		valueType == nil {
		return Specialization{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "map specialization contract is invalid",
		}
	}
	model, represented := Source(context, mapType)
	if !represented || model.Storage() == StorageScalar {
		return Specialization{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "map specialization contract is invalid",
		}
	}
	operations, requests, err := specializationOperations(
		context,
		source,
		mapType,
		model.Storage(),
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
	mapValueReference, err := context.Names().Runtime(
		api.RuntimeMapValue,
		api.ImportPhaseValue,
	)
	if err != nil {
		return Specialization{}, err
	}
	var mapStoreRequests []api.RootRequest
	mapStoreName := ""
	if model.Storage() == StorageNative {
		mapStoreReference, err := context.Names().Runtime(
			api.RuntimeMapStore,
			api.ImportPhaseValue,
		)
		if err != nil {
			return Specialization{}, err
		}
		mapStoreName = mapStoreReference.Name()
		mapStoreRequests = mapStoreReference.Requests()
	}
	builder := specializationBuilder{
		factory:        context.Factory(),
		className:      className,
		keyType:        keyType,
		storageKeyType: storageKeyType,
		valueType:      valueType,
		panicName:      panicReference.Name(),
		mapStoreName:   mapStoreName,
		zero:           operations.zero,
		hash:           operations.hash,
		equal:          operations.equal,
		copyKey:        operations.copyKey,
		copyValue:      operations.copyValue,
		projectKey:     operations.projectKey,
		reifyKey:       operations.reifyKey,
		keyProjection:  operations.keyProjection,
		members:        memberNames,
	}
	var members []tsgo.ClassElement
	switch model.Storage() {
	case StorageNative:
		members = builder.buildNative()
	case StorageHashed:
		members = builder.buildHashed()
	default:
		return Specialization{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "map specialization storage is invalid",
		}
	}
	if err := validateSpecialization(
		context.Role(),
		members,
		model.Storage(),
		operations.keyProjection,
	); err != nil {
		return Specialization{}, err
	}
	return Specialization{
		heritage: []tsgo.HeritageClause{context.Factory().HeritageClause(
			tsgo.HeritageClauseTokenKindExtendsKeyword,
			[]tsgo.ExpressionWithTypeArguments{
				context.Factory().ExpressionWithTypeArguments(
					mapValueReference.Expression(context.Factory()),
					[]tsgo.TypeNode{keyType, valueType},
				),
			},
		)},
		members: members,
		requests: api.CombineRequests(
			requests,
			panicReference.Requests(),
			mapValueReference.Requests(),
			mapStoreRequests,
		),
	}, nil
}

func ValidateRequirements(
	role api.Role,
	artifact *api.GeneratedArtifact,
	requirements []api.DeclarationRequirement,
) error {
	if !artifact.Valid() ||
		artifact.Kind() != api.GeneratedArtifactMapSpecialization {
		return &api.InvariantError{
			Role:   role,
			Reason: "map specialization requirement owner is invalid",
		}
	}
	for _, requirement := range requirements {
		selected, _, valid := requirement.MapSpecialization()
		if !valid || selected != artifact {
			return &api.InvariantError{
				Role:   role,
				Reason: "map specialization received a foreign requirement",
			}
		}
	}
	return nil
}

func specializationNames() (specializationMemberNames, error) {
	resolved := make([]string, 0, mapruntime.MemberKeys)
	for member := mapruntime.MemberNil; member <= mapruntime.MemberKeys; member++ {
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
		keys:         resolved[9],
	}, nil
}

type specializationOperationSet struct {
	zero          operationBody
	hash          operationBody
	equal         operationBody
	copyKey       operationBody
	copyValue     operationBody
	projectKey    operationBody
	reifyKey      operationBody
	keyProjection bool
}

func specializationOperations(
	context api.Context,
	source ast.Node,
	mapType *types.Map,
	storage Storage,
) (specializationOperationSet, []api.RootRequest, error) {
	zero, err := context.Values().Zero(
		context.WithRole(api.RoleMapValue),
		source,
		mapType.Elem(),
	)
	if err != nil {
		return specializationOperationSet{}, nil, err
	}
	value := context.Factory().Identifier("$value")
	copyValue, err := context.Values().Transfer(
		context.WithRole(api.RoleMapValue),
		nil,
		mapType.Elem(),
		mapType.Elem(),
		api.ValueTransferCopy,
		api.DirectExpression(value),
	)
	if err != nil {
		return specializationOperationSet{}, nil, err
	}
	operations := specializationOperationSet{
		zero:      operation(zero),
		copyValue: operation(copyValue),
	}
	requests := api.CombineRequests(
		zero.Requests(),
		copyValue.Requests(),
	)
	if storage != StorageNative && storage != StorageHashed {
		return specializationOperationSet{}, nil, &api.InvariantError{
			Role:   context.Role(),
			Reason: "map specialization operation storage is invalid",
		}
	}
	keyType, err := storageKeyType(context, mapType.Key())
	if err != nil {
		return specializationOperationSet{}, nil, err
	}
	key := context.Factory().Identifier("$key")
	keyProjection := !types.Identical(mapType.Key(), keyType)
	var projectKey api.ExpressionEmission
	var reifyKey api.ExpressionEmission
	var projectKeyBody operationBody
	var reifyKeyBody operationBody
	if keyProjection {
		projectKey, err = context.Values().ToStorage(
			context.WithRole(api.RoleMapKey),
			source,
			mapType.Key(),
			api.DirectExpression(key),
		)
		if err != nil {
			return specializationOperationSet{}, nil, err
		}
		reifyKey, err = context.Values().FromStorage(
			context.WithRole(api.RoleMapKey),
			source,
			mapType.Key(),
			api.DirectExpression(
				context.Factory().Identifier("$storageKey"),
			),
		)
		if err != nil {
			return specializationOperationSet{}, nil, err
		}
		projectKeyBody = operation(projectKey)
		reifyKeyBody = operation(reifyKey)
	}
	operations.projectKey = projectKeyBody
	operations.reifyKey = reifyKeyBody
	operations.keyProjection = keyProjection
	requests = api.CombineRequests(
		requests,
		projectKey.Requests(),
		reifyKey.Requests(),
	)
	if storage == StorageNative {
		return operations, requests, nil
	}
	keyContext, err := storageKeyOperationContext(context, mapType.Key())
	if err != nil {
		return specializationOperationSet{}, nil, err
	}
	hash, err := context.Values().Hash(
		keyContext.WithRole(api.RoleMapKey),
		source,
		keyType,
		key,
	)
	if err != nil {
		return specializationOperationSet{}, nil, err
	}
	left := context.Factory().Identifier("$left")
	right := context.Factory().Identifier("$right")
	equal, err := context.Values().Equal(
		keyContext.WithRole(api.RoleMapKey),
		source,
		keyType,
		left,
		right,
	)
	if err != nil {
		return specializationOperationSet{}, nil, err
	}
	copyKey, err := context.Values().Transfer(
		keyContext.WithRole(api.RoleMapKey),
		nil,
		keyType,
		keyType,
		api.ValueTransferCopy,
		api.DirectExpression(key),
	)
	if err != nil {
		return specializationOperationSet{}, nil, err
	}
	operations.hash = operation(hash)
	operations.equal = operation(equal)
	operations.copyKey = operation(copyKey)
	return operations, api.CombineRequests(
		requests,
		hash.Requests(),
		equal.Requests(),
		copyKey.Requests(),
	), nil
}

func operation(emission api.ExpressionEmission) operationBody {
	return operationBody{
		before: emission.Before(),
		value:  emission.Value(),
	}
}
