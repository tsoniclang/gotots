package representation

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	genericinstance "github.com/tsoniclang/gotots/internal/emit/generic/instance"
	genericoperation "github.com/tsoniclang/gotots/internal/emit/generic/operation"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	"github.com/tsoniclang/gotots/internal/emit/value/maprepresentation"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (owner Owner) RequiresStorageProjection(
	context api.Context,
	sourceType types.Type,
) (bool, error) {
	if _, generic := api.GenericTypeParameter(sourceType); generic {
		return true, nil
	}
	if panicNilRuntimeValue(context, sourceType) {
		return false, nil
	}
	if model, ok := maprepresentation.Source(context, sourceType); ok {
		return model.Nominal(), nil
	}
	if defined, ok := definedStorageModel(sourceType); ok {
		representation, err := defined.Representation(context)
		if err != nil {
			return false, err
		}
		switch representation.Kind() {
		case api.DefinedValueRepresentationGeneratedNumeric,
			api.DefinedValueRepresentationProviderCanonical:
			return false, nil
		case api.DefinedValueRepresentationGeneratedWrapper,
			api.DefinedValueRepresentationProviderOperations:
			return true, nil
		default:
			return false, &api.InvariantError{
				Role:   context.Role(),
				Reason: "defined storage projection representation is invalid",
			}
		}
	}
	if _, ok := isAnonymousStruct(sourceType); ok {
		return true, nil
	}
	typeName, _, ok := namedStruct(sourceType)
	if !ok {
		return false, nil
	}
	if context.Names() == nil {
		return false, &api.InvariantError{
			Role:   context.Role(),
			Reason: "storage projection has no name owner",
		}
	}
	providerOwned, err := context.Names().ProviderOwnedDeclaration(typeName)
	if err != nil {
		return false, err
	}
	return !providerOwned, nil
}

func (owner Owner) StorageType(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
) (api.TypeEmission, error) {
	if parameter, generic := api.GenericTypeParameter(sourceType); generic {
		return owner.genericStorageType(
			context,
			source,
			parameter,
			api.GenericRepresentationStorage,
			api.RuntimeStorageType,
		)
	}
	if panicNilRuntimeValue(context, sourceType) {
		return owner.children.RepresentedType(
			context,
			source,
			sourceType,
		)
	}
	if model, ok := maprepresentation.Source(context, sourceType); ok &&
		model.Nominal() {
		return maprepresentation.EmitType(
			context.WithRole(api.RoleDefinedUnderlyingType),
			owner.children,
			source,
			model.Map(),
		)
	}
	if defined, ok := definedStorageModel(sourceType); ok {
		operationContext, err := defined.OperationContext(context)
		if err != nil {
			return api.TypeEmission{}, err
		}
		return owner.StorageType(
			operationContext.WithRole(api.RoleDefinedUnderlyingType),
			source,
			defined.Underlying(),
		)
	}
	if structType, ok := isAnonymousStruct(sourceType); ok {
		reference, err := context.Names().AnonymousStructStorage(structType)
		if err != nil {
			return api.TypeEmission{}, err
		}
		return api.DirectType(
			context.Factory().TypeReferenceNode(
				reference.EntityName(context.Factory()),
				nil,
			),
			reference.Requests()...,
		), nil
	}
	if typeName, _, ok := namedStruct(sourceType); ok {
		required, err := owner.RequiresStorageProjection(context, sourceType)
		if err != nil {
			return api.TypeEmission{}, err
		}
		if !required {
			return owner.children.RepresentedType(
				context,
				source,
				sourceType,
			)
		}
		reference, err := context.Names().NamedStructStorage(typeName)
		if err != nil {
			return api.TypeEmission{}, err
		}
		var typeArguments []tsgo.TypeNode
		var argumentRequests []api.RootRequest
		named, namedOK := types.Unalias(sourceType).(*types.Named)
		if !namedOK {
			return api.TypeEmission{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "named-struct storage source type is invalid",
			}
		}
		if named.TypeArgs().Len() != 0 {
			typeArguments, argumentRequests, err =
				genericinstance.EmitTypeArguments(
					context,
					owner.children,
					source,
					named.Origin().Obj(),
					api.TypeArgumentsFromGo(named.TypeArgs()),
				)
			if err != nil {
				return api.TypeEmission{}, err
			}
		}
		return api.DirectType(
			context.Factory().TypeReferenceNode(
				reference.EntityName(context.Factory()),
				typeArguments,
			),
			api.CombineRequests(
				reference.Requests(),
				argumentRequests,
			)...,
		), nil
	}
	return owner.children.RepresentedType(
		context,
		source,
		sourceType,
	)
}

func (owner Owner) ToStorage(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if parameter, generic := api.GenericTypeParameter(sourceType); generic {
		return genericoperation.Call(
			context,
			source,
			api.GenericOperationToStorage,
			[]types.Type{parameter},
			[]types.Type{parameter},
			[]api.ExpressionEmission{value},
		)
	}
	if panicNilRuntimeValue(context, sourceType) {
		return value, nil
	}
	if model, ok := maprepresentation.Source(context, sourceType); ok &&
		model.Nominal() {
		return model.ReadReceiver(context, source, value)
	}
	if defined, ok := definedStorageModel(sourceType); ok {
		operationContext, err := defined.OperationContext(context)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		projected, err := defined.Project(context, value)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return owner.ToStorage(
			operationContext.WithRole(api.RoleDefinedUnderlyingType),
			source,
			defined.Underlying(),
			projected,
		)
	}
	if structType, ok := isAnonymousStruct(sourceType); ok {
		reference, err := context.Names().AnonymousStruct(
			structType,
			api.AnonymousStructDemandStorage,
			api.ImportPhaseValue,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return callStorageMember(
			context,
			reference,
			api.StructStorageOfMember,
			value,
		)
	}
	if _, _, ok := namedStruct(sourceType); ok {
		required, err := owner.RequiresStorageProjection(context, sourceType)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		if !required {
			return value, nil
		}
		converted, err := owner.namedStructOperationMember(
			context,
			source,
			sourceType,
			api.NamedStructOperationStorage,
			api.StructStorageOfMember,
			[]tsgo.Expression{value.Value()},
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return api.NewExpressionEmission(
			value.Before(),
			converted.Value(),
			api.CombineRequests(value.Requests(), converted.Requests()),
		)
	}
	return value, nil
}

func (owner Owner) FromStorage(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if parameter, generic := api.GenericTypeParameter(sourceType); generic {
		return genericoperation.Call(
			context,
			source,
			api.GenericOperationFromStorage,
			[]types.Type{parameter},
			[]types.Type{parameter},
			[]api.ExpressionEmission{value},
		)
	}
	if panicNilRuntimeValue(context, sourceType) {
		return value, nil
	}
	if model, ok := maprepresentation.Source(context, sourceType); ok &&
		model.Nominal() {
		return model.Wrap(context, value)
	}
	if defined, ok := definedStorageModel(sourceType); ok {
		operationContext, err := defined.OperationContext(context)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		restored, err := owner.FromStorage(
			operationContext.WithRole(api.RoleDefinedUnderlyingType),
			source,
			defined.Underlying(),
			value,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return defined.Wrap(context, restored)
	}
	if structType, ok := isAnonymousStruct(sourceType); ok {
		reference, err := context.Names().AnonymousStruct(
			structType,
			api.AnonymousStructDemandStorage,
			api.ImportPhaseValue,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return callStorageMember(
			context,
			reference,
			api.StructFromStorageMember,
			value,
		)
	}
	if _, _, ok := namedStruct(sourceType); ok {
		required, err := owner.RequiresStorageProjection(context, sourceType)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		if !required {
			return value, nil
		}
		converted, err := owner.namedStructOperationMember(
			context,
			source,
			sourceType,
			api.NamedStructOperationStorage,
			api.StructFromStorageMember,
			[]tsgo.Expression{value.Value()},
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return api.NewExpressionEmission(
			value.Before(),
			converted.Value(),
			api.CombineRequests(value.Requests(), converted.Requests()),
		)
	}
	return value, nil
}

func definedStorageModel(sourceType types.Type) (definedtype.Model, bool) {
	defined, ok := definedtype.Resolve(sourceType)
	if !ok {
		return definedtype.Model{}, false
	}
	return defined, defined.Family() == definedtype.FamilyBasic ||
		defined.Family() == definedtype.FamilyArray
}

func callStorageMember(
	context api.Context,
	reference api.NameReference,
	member string,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	return api.NewExpressionEmission(
		value.Before(),
		context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				reference.Expression(context.Factory()),
				nil,
				context.Factory().Identifier(member),
				tsgo.NodeFlagsNone,
			),
			nil,
			nil,
			[]tsgo.Expression{value.Value()},
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(value.Requests(), reference.Requests()),
	)
}
