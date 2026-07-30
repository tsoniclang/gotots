package representation

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	genericinstance "github.com/tsoniclang/gotots/internal/emit/generic/instance"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (owner Owner) RequiresStorageProjection(
	context api.Context,
	sourceType types.Type,
) bool {
	if panicNilRuntimeValue(context, sourceType) {
		return false
	}
	if _, ok := definedStorageModel(sourceType); ok {
		return true
	}
	if _, ok := isAnonymousStruct(sourceType); ok {
		return true
	}
	_, _, ok := namedStruct(sourceType)
	return ok
}

func (owner Owner) StorageType(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
) (api.TypeEmission, error) {
	if panicNilRuntimeValue(context, sourceType) {
		return owner.children.RepresentedType(
			context,
			source,
			sourceType,
		)
	}
	if defined, ok := definedStorageModel(sourceType); ok {
		return owner.StorageType(
			context.WithRole(api.RoleDefinedUnderlyingType),
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
				context.Factory().Identifier(reference.Name()),
				nil,
			),
			reference.Requests()...,
		), nil
	}
	if typeName, _, ok := namedStruct(sourceType); ok {
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
					named.TypeArgs(),
				)
			if err != nil {
				return api.TypeEmission{}, err
			}
		}
		return api.DirectType(
			context.Factory().TypeReferenceNode(
				context.Factory().Identifier(reference.Name()),
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
	if panicNilRuntimeValue(context, sourceType) {
		return value, nil
	}
	if defined, ok := definedStorageModel(sourceType); ok {
		projected, err := defined.Project(context, value)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return owner.ToStorage(
			context.WithRole(api.RoleDefinedUnderlyingType),
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
	if panicNilRuntimeValue(context, sourceType) {
		return value, nil
	}
	if defined, ok := definedStorageModel(sourceType); ok {
		restored, err := owner.FromStorage(
			context.WithRole(api.RoleDefinedUnderlyingType),
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
				context.Factory().Identifier(reference.Name()),
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
