package representation

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	mapruntime "github.com/tsoniclang/gotots/internal/emit/runtime/map"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	arrayvalue "github.com/tsoniclang/gotots/internal/emit/value/array"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (Owner) SupportsHash(
	context api.Context,
	sourceType types.Type,
) bool {
	return supportsHash(context, sourceType, make(map[types.Type]bool))
}

func supportsHash(
	context api.Context,
	sourceType types.Type,
	visiting map[types.Type]bool,
) bool {
	if sourceType == nil || visiting[sourceType] {
		return false
	}
	if defined, ok := definedtype.Resolve(sourceType); ok {
		visiting[sourceType] = true
		result := supportsHash(context, defined.Underlying(), visiting)
		delete(visiting, sourceType)
		return result
	}
	if basic, ok := types.Unalias(sourceType).(*types.Basic); ok {
		if basic.Info()&types.IsUntyped != 0 ||
			basic.Info()&(types.IsBoolean|types.IsInteger|types.IsString) == 0 {
			return false
		}
		_, represented := basictype.PrimitiveAlias(
			context.TypesSizes(),
			basic,
		)
		return represented
	}
	if array, ok := arrayvalue.Resolve(context, sourceType); ok {
		return types.Comparable(sourceType) &&
			supportsHash(context, array.ElementType(), visiting)
	}
	if structType, ok := isAnonymousStruct(sourceType); ok {
		if !types.Comparable(sourceType) {
			return false
		}
		visiting[sourceType] = true
		defer delete(visiting, sourceType)
		for index := range structType.NumFields() {
			if structType.Field(index).Name() == "_" {
				continue
			}
			if !supportsHash(
				context,
				structType.Field(index).Type(),
				visiting,
			) {
				return false
			}
		}
		return true
	}
	_, structType, ok := namedStruct(sourceType)
	if !ok || !types.Comparable(sourceType) {
		return false
	}
	visiting[sourceType] = true
	defer delete(visiting, sourceType)
	for index := range structType.NumFields() {
		if structType.Field(index).Name() == "_" {
			continue
		}
		if !supportsHash(context, structType.Field(index).Type(), visiting) {
			return false
		}
	}
	return true
}

func (Owner) Hash(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	value tsgo.Expression,
) (api.ExpressionEmission, error) {
	if defined, ok := definedtype.Resolve(sourceType); ok {
		return (Owner{}).Hash(
			context.WithRole(api.RoleDefinedValue),
			source,
			defined.Underlying(),
			defined.Unwrap(context.Factory(), value),
		)
	}
	if array, ok := arrayvalue.Resolve(context, sourceType); ok {
		return array.Hash(context, source, value)
	}
	if structType, ok := isAnonymousStruct(sourceType); ok {
		if !(Owner{}).SupportsHash(context, sourceType) {
			return api.ExpressionEmission{},
				api.Unsupported(context, api.CategoryExpression, source)
		}
		return anonymousStructHash(
			context,
			source,
			structType,
			value,
		)
	}
	if basic, ok := types.Unalias(sourceType).(*types.Basic); ok {
		member, valid := hashMember(context, basic)
		if !valid {
			return api.ExpressionEmission{},
				api.Unsupported(context, api.CategoryExpression, source)
		}
		reference, err := context.Names().Runtime(
			api.RuntimeMapHash,
			api.ImportPhaseValue,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return api.DirectExpression(
			context.Factory().CallExpression(
				context.Factory().PropertyAccessExpression(
					context.Factory().Identifier(reference.Name()),
					nil,
					context.Factory().Identifier(member),
					tsgo.NodeFlagsNone,
				),
				nil,
				nil,
				[]tsgo.Expression{value},
				tsgo.NodeFlagsNone,
			),
			reference.Requests()...,
		), nil
	}
	typeName, _, ok := namedStruct(sourceType)
	if !ok || !(Owner{}).SupportsHash(context, sourceType) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	reference, err := context.Names().NamedStructOperation(
		typeName,
		api.NamedStructOperationHash,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	call, err := namedStructOperationCall(
		context,
		reference.Name(),
		api.NamedStructOperationHash,
		[]tsgo.Expression{value},
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.DirectExpression(call, reference.Requests()...), nil
}

func hashMember(
	context api.Context,
	basic *types.Basic,
) (string, bool) {
	if basic == nil || basic.Info()&types.IsUntyped != 0 {
		return "", false
	}
	switch {
	case basic.Info()&types.IsBoolean != 0:
		return mapruntime.HashBooleanMember, true
	case basic.Info()&types.IsString != 0:
		return mapruntime.HashStringMember, true
	case basic.Info()&types.IsInteger != 0:
		if _, ok := basictype.PrimitiveAlias(
			context.TypesSizes(),
			basic,
		); !ok {
			return "", false
		}
		if context.IntegerRepresentation() == api.IntegerRepresentationBigInt {
			return mapruntime.HashBigIntMember, true
		}
		return mapruntime.HashNumberMember, true
	default:
		return "", false
	}
}
