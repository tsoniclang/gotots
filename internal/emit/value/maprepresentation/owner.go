package maprepresentation

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	mapruntime "github.com/tsoniclang/gotots/internal/emit/runtime/map"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func EmitType(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	sourceType types.Type,
) (api.TypeEmission, error) {
	model, ok := Source(context, sourceType)
	if !ok {
		return api.TypeEmission{},
			api.Unsupported(context, api.CategoryType, source)
	}
	if model.Nominal() {
		reference, err := context.Names().TypeReference(model.TypeName())
		if err != nil {
			return api.TypeEmission{}, err
		}
		return api.DirectType(
			context.Factory().UnionTypeNode([]tsgo.TypeNode{
				context.Factory().TypeReferenceNode(
					reference.EntityName(context.Factory()),
					nil,
				),
				context.Factory().KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
				),
			}),
			reference.Requests()...,
		), nil
	}
	key, err := children.RepresentedType(
		context.WithRole(api.RoleMapKey),
		source,
		model.Key(),
	)
	if err != nil {
		return api.TypeEmission{}, err
	}
	value, err := children.RepresentedType(
		context.WithRole(api.RoleMapValue),
		source,
		model.Element(),
	)
	if err != nil {
		return api.TypeEmission{}, err
	}
	reference, err := context.Names().Runtime(
		api.RuntimeMapValue,
		api.ImportPhaseType,
	)
	if err != nil {
		return api.TypeEmission{}, err
	}
	return api.DirectType(
		context.Factory().TypeReferenceNode(
			reference.EntityName(context.Factory()),
			[]tsgo.TypeNode{key.Value(), value.Value()},
		),
		api.CombineRequests(
			key.Requests(),
			value.Requests(),
			reference.Requests(),
		)...,
	), nil
}

func Nil(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	sourceType types.Type,
) (api.ExpressionEmission, error) {
	model, ok := Source(context, sourceType)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if api.ContainsGenericTypeParameter(sourceType) {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "open generic map zero bypassed generic operation ownership",
		}
	}
	if model.Nominal() {
		return api.DirectExpression(
			context.Factory().VoidExpression(
				context.Factory().NumericLiteral("0", tsgo.TokenFlagsNone),
			),
		), nil
	}
	if model.Storage() == StorageSpecialized {
		reference, err := context.Names().MapSpecialization(
			sourceType,
			api.MapSpecializationDemandStatic,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		nilName, err := mapruntime.Name(mapruntime.MemberNil)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return api.DirectExpression(
			context.Factory().CallExpression(
				staticMember(context, reference.Name(), nilName),
				nil,
				nil,
				nil,
				tsgo.NodeFlagsNone,
			),
			reference.Requests()...,
		), nil
	}
	if children == nil {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "nil map has no type emitter",
		}
	}
	keyType, err := children.RepresentedType(
		context.WithRole(api.RoleMapKey),
		source,
		model.Key(),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	valueType, err := children.RepresentedType(
		context.WithRole(api.RoleMapValue),
		source,
		model.Element(),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	zero, err := context.Values().Zero(
		context.WithRole(api.RoleMapValue),
		source,
		model.Element(),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	reference, err := context.Names().Runtime(
		api.RuntimeMap,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	nilName, err := mapruntime.Name(mapruntime.MemberNil)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		zero.Before(),
		context.Factory().CallExpression(
			staticMember(context, reference.Name(), nilName),
			nil,
			[]tsgo.TypeNode{keyType.Value(), valueType.Value()},
			[]tsgo.Expression{zero.Value()},
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			keyType.Requests(),
			valueType.Requests(),
			zero.Requests(),
			reference.Requests(),
		),
	)
}

func Reference(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	phase api.ImportPhase,
) (api.NameReference, []tsgo.TypeNode, error) {
	model, ok := Source(context, sourceType)
	if !ok || model.Storage() != StorageScalar {
		return api.NameReference{}, nil,
			api.Unsupported(context, api.CategoryType, source)
	}
	mapType := model.Map()
	keyBasic, _ := directKey(context, model.Key())
	key, keyRequests, err := primitiveType(context, source, keyBasic)
	if err != nil {
		return api.NameReference{}, nil, err
	}
	value, valueRequests, err := representedType(
		context,
		source,
		mapType.Elem(),
	)
	if err != nil {
		return api.NameReference{}, nil, err
	}
	reference, err := context.Names().Runtime(api.RuntimeMap, phase)
	if err != nil {
		return api.NameReference{}, nil, err
	}
	requests := api.CombineRequests(
		reference.Requests(),
		keyRequests,
		valueRequests,
	)
	reference, err = reference.WithRequests(requests...)
	if err != nil {
		return api.NameReference{}, nil, err
	}
	return reference, []tsgo.TypeNode{key, value}, nil
}

func Make(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	zero tsgo.Expression,
	size tsgo.Expression,
	entries []tsgo.Expression,
	requests ...[]api.RootRequest,
) (api.ExpressionEmission, error) {
	model, ok := Source(context, sourceType)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if model.Storage() == StorageSpecialized {
		reference, err := context.Names().MapSpecialization(
			sourceType,
			api.MapSpecializationDemandStatic,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		makeName, err := mapruntime.Name(mapruntime.MemberMake)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		allRequests := append(
			[][]api.RootRequest{reference.Requests()},
			requests...,
		)
		return model.Wrap(context, api.DirectExpression(
			context.Factory().CallExpression(
				staticMember(context, reference.Name(), makeName),
				nil,
				nil,
				[]tsgo.Expression{
					size,
					context.Factory().ArrayLiteralExpression(entries, false),
				},
				tsgo.NodeFlagsNone,
			),
			api.CombineRequests(allRequests...)...,
		))
	}
	reference, typeArguments, err := Reference(
		context,
		source,
		sourceType,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	makeName, err := mapruntime.Name(mapruntime.MemberMake)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	allRequests := append(
		[][]api.RootRequest{reference.Requests()},
		requests...,
	)
	return model.Wrap(context, api.DirectExpression(
		context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				reference.Expression(context.Factory()),
				nil,
				context.Factory().Identifier(makeName),
				tsgo.NodeFlagsNone,
			),
			nil,
			typeArguments,
			[]tsgo.Expression{
				zero,
				size,
				context.Factory().ArrayLiteralExpression(entries, false),
			},
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(allRequests...)...,
	))
}

func storageKeyType(
	context api.Context,
	sourceType types.Type,
) (types.Type, error) {
	if model, ok := definedtype.ResolveBasic(sourceType); ok {
		representation, err := model.Representation(context)
		if err != nil {
			return nil, err
		}
		if representation.Kind() ==
			api.DefinedValueRepresentationGeneratedNumeric {
			return sourceType, nil
		}
		return model.Underlying(), nil
	}
	return sourceType, nil
}

func underlyingStorageKeyType(sourceType types.Type) types.Type {
	if model, ok := definedtype.ResolveBasic(sourceType); ok {
		return model.Underlying()
	}
	return sourceType
}

func storageKeyOperationContext(
	context api.Context,
	sourceType types.Type,
) (api.Context, error) {
	model, defined := definedtype.ResolveBasic(sourceType)
	if !defined {
		return context, nil
	}
	return model.OperationContext(context)
}

func EmitStorageKeyType(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	sourceType types.Type,
) (api.TypeEmission, error) {
	if children == nil || sourceType == nil || !types.Comparable(sourceType) {
		return api.TypeEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "map storage key type contract is invalid",
		}
	}
	operationContext, err := storageKeyOperationContext(context, sourceType)
	if err != nil {
		return api.TypeEmission{}, err
	}
	storageType, err := storageKeyType(context, sourceType)
	if err != nil {
		return api.TypeEmission{}, err
	}
	return children.RepresentedType(
		operationContext.WithRole(api.RoleStorageType),
		source,
		storageType,
	)
}

func directKey(
	context api.Context,
	sourceType types.Type,
) (*types.Basic, bool) {
	var basic *types.Basic
	if model, ok := definedtype.ResolveBasic(sourceType); ok {
		basic, _ = model.Basic()
	} else {
		basic, _ = types.Unalias(sourceType).(*types.Basic)
	}
	if basic == nil ||
		basic.Info()&types.IsUntyped != 0 ||
		basic.Info()&(types.IsBoolean|types.IsInteger|types.IsString) == 0 {
		return nil, false
	}
	_, represented := basictype.PrimitiveAlias(context.TypesSizes(), basic)
	return basic, represented
}

func representedBasic(
	context api.Context,
	sourceType types.Type,
) bool {
	if _, ok := definedtype.ResolveBasic(sourceType); ok {
		return true
	}
	_, ok := basictype.PrimitiveAlias(context.TypesSizes(), sourceType)
	return ok
}

func representedType(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
) (tsgo.TypeNode, []api.RootRequest, error) {
	if model, ok := definedtype.ResolveBasic(sourceType); ok {
		reference, err := context.Names().TypeReference(model.TypeName())
		if err != nil {
			return nil, nil, err
		}
		return context.Factory().TypeReferenceNode(
			reference.EntityName(context.Factory()),
			nil,
		), reference.Requests(), nil
	}
	basic, ok := types.Unalias(sourceType).(*types.Basic)
	if !ok {
		return nil, nil,
			api.Unsupported(context, api.CategoryType, source)
	}
	return primitiveType(context, source, basic)
}

func primitiveType(
	context api.Context,
	source ast.Node,
	basic *types.Basic,
) (tsgo.TypeNode, []api.RootRequest, error) {
	alias, ok := basictype.PrimitiveAlias(context.TypesSizes(), basic)
	if !ok {
		return nil, nil,
			api.Unsupported(context, api.CategoryType, source)
	}
	reference, err := context.Names().Primitive(alias)
	if err != nil {
		return nil, nil, err
	}
	return context.Factory().TypeReferenceNode(
		reference.EntityName(context.Factory()),
		nil,
	), reference.Requests(), nil
}
