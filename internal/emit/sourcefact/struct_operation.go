package sourcefact

import (
	"go/types"
	"sort"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/emit/api"
	attribute "github.com/tsoniclang/gotots/internal/emit/marker/attribute"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const structOperationSchema = "gotots-go-struct-operation-fact-v1"

type structOperation struct {
	operation    api.NamedStructOperation
	capabilities []*api.GenericOperationContract
}

func StructOperations(
	context api.Context,
	owner *types.TypeName,
	requirements []api.DeclarationRequirement,
) (api.StatementEmission, error) {
	if owner == nil || owner.IsAlias() {
		return api.StatementEmission{}, &Error{Reason: "struct-operation owner is invalid"}
	}
	contract, err := environmentcontract.Describe(owner)
	if err != nil {
		return api.StatementEmission{}, err
	}
	return structOperations(context, owner, contract.Identity(), requirements)
}

func LocalStructOperations(
	context api.Context,
	owner *types.TypeName,
	requirements []api.DeclarationRequirement,
) (api.StatementEmission, error) {
	contract, err := localTypeDeclarationContract(context, owner)
	if err != nil {
		return api.StatementEmission{}, err
	}
	return structOperations(context, owner, contract.identity, requirements)
}

func structOperations(
	context api.Context,
	owner *types.TypeName,
	ownerIdentity string,
	requirements []api.DeclarationRequirement,
) (api.StatementEmission, error) {
	if owner == nil || owner.IsAlias() || ownerIdentity == "" {
		return api.StatementEmission{}, &Error{Reason: "struct-operation owner is invalid"}
	}
	operations, err := selectedStructOperations(owner, requirements)
	if err != nil {
		return api.StatementEmission{}, err
	}
	name, err := context.Names().Declare(owner)
	if err != nil {
		return api.StatementEmission{}, err
	}
	return structOperationFacts(
		context,
		context.Factory().TypeQueryNode(context.Factory().Identifier(name), nil),
		ownerIdentity,
		operations,
	)
}

func AnonymousStructOperations(
	context api.Context,
	artifact *api.GeneratedArtifact,
	operations []api.NamedStructOperation,
) (api.StatementEmission, error) {
	if artifact == nil || artifact.Kind() != api.GeneratedArtifactAnonymousStruct {
		return api.StatementEmission{}, &Error{Reason: "anonymous struct-operation owner is invalid"}
	}
	selected := make([]structOperation, 0, len(operations))
	seen := make(map[api.NamedStructOperation]struct{}, len(operations))
	for _, operation := range operations {
		if !operation.Valid() {
			return api.StatementEmission{}, &Error{Subject: artifact.TargetName(), Reason: "anonymous struct operation is invalid"}
		}
		if _, duplicate := seen[operation]; duplicate {
			return api.StatementEmission{}, &Error{Subject: artifact.TargetName(), Reason: "anonymous struct operation is duplicated"}
		}
		seen[operation] = struct{}{}
		selected = append(selected, structOperation{operation: operation})
	}
	sort.Slice(selected, func(left, right int) bool {
		return selected[left].operation < selected[right].operation
	})
	return structOperationFacts(
		context,
		context.Factory().TypeQueryNode(
			context.Factory().Identifier(artifact.TargetName()),
			nil,
		),
		artifact.ArtifactKey(),
		selected,
	)
}

func selectedStructOperations(
	owner *types.TypeName,
	requirements []api.DeclarationRequirement,
) ([]structOperation, error) {
	byOperation := make(map[api.NamedStructOperation]*structOperation)
	for _, requirement := range requirements {
		if selectedOwner, operation, ok := requirement.NamedStructOperation(); ok {
			if selectedOwner != owner {
				return nil, &Error{Subject: owner.Name(), Reason: "struct operation has a foreign owner"}
			}
			if byOperation[operation] != nil {
				return nil, &Error{Subject: owner.Name(), Reason: "struct operation is duplicated"}
			}
			byOperation[operation] = &structOperation{operation: operation}
		}
	}
	for _, requirement := range requirements {
		selectedOwner, capability, ok := requirement.GenericOperation()
		if !ok || selectedOwner != owner {
			continue
		}
		operation, selected := capability.Consumer().NamedStructOperation()
		if !selected {
			continue
		}
		entry := byOperation[operation]
		if entry == nil {
			return nil, &Error{Subject: owner.Name(), Reason: "generic struct capability has no owning operation"}
		}
		entry.capabilities = append(entry.capabilities, capability)
	}
	result := make([]structOperation, 0, len(byOperation))
	for _, operation := range byOperation {
		sort.Slice(operation.capabilities, func(left, right int) bool {
			return operation.capabilities[left].Key() < operation.capabilities[right].Key()
		})
		result = append(result, *operation)
	}
	sort.Slice(result, func(left, right int) bool {
		return result[left].operation < result[right].operation
	})
	return result, nil
}

func structOperationFacts(
	context api.Context,
	target tsgo.TypeNode,
	ownerIdentity string,
	operations []structOperation,
) (api.StatementEmission, error) {
	emissions := make([]api.StatementEmission, 0, len(operations)+1)
	for _, selected := range operations {
		members, err := structOperationMembers(selected.operation)
		if err != nil {
			return api.StatementEmission{}, err
		}
		for _, member := range members {
			arguments := []tsgo.Expression{
				text(context.Factory(), structOperationSchema),
				text(context.Factory(), ownerIdentity),
				text(context.Factory(), selected.operation.String()),
				text(context.Factory(), member.role),
				count(context.Factory(), len(selected.capabilities)),
			}
			for index, capability := range selected.capabilities {
				arguments = append(
					arguments,
					count(context.Factory(), index),
					text(context.Factory(), capability.Key()),
					text(context.Factory(), capability.Operation().Identifier()),
				)
			}
			emission, err := attribute.ApplyMember(
				context,
				target,
				attribute.MemberMethod,
				member.name,
				api.RuntimeSourceOperationFact,
				arguments...,
			)
			if err != nil {
				return api.StatementEmission{}, err
			}
			emissions = append(emissions, emission)
		}
	}
	return combine(emissions)
}

type structOperationMember struct {
	name string
	role string
}

func structOperationMembers(
	operation api.NamedStructOperation,
) ([]structOperationMember, error) {
	if operation == api.NamedStructOperationStorage {
		return []structOperationMember{
			{name: api.StructStorageOfMember, role: "to-storage"},
			{name: api.StructFromStorageMember, role: "from-storage"},
		}, nil
	}
	name, err := api.NamedStructOperationMemberName(operation)
	if err != nil {
		return nil, err
	}
	return []structOperationMember{{name: name, role: operation.String()}}, nil
}
