package representation

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (owner Owner) RequiresStorageProjection(
	_ api.Context,
	sourceType types.Type,
) bool {
	if _, ok := definedtype.ResolveBasic(sourceType); ok {
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
	if defined, ok := definedtype.ResolveBasic(sourceType); ok {
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
		return api.DirectType(
			context.Factory().TypeReferenceNode(
				context.Factory().Identifier(reference.Name()),
				nil,
			),
			reference.Requests()...,
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
	if defined, ok := definedtype.ResolveBasic(sourceType); ok {
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
	if typeName, _, ok := namedStruct(sourceType); ok {
		reference, err := context.Names().NamedStructOperation(
			typeName,
			api.NamedStructOperationStorage,
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
	return value, nil
}

func (owner Owner) FromStorage(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if defined, ok := definedtype.ResolveBasic(sourceType); ok {
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
	if typeName, _, ok := namedStruct(sourceType); ok {
		reference, err := context.Names().NamedStructOperation(
			typeName,
			api.NamedStructOperationStorage,
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
	return value, nil
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
