package basic

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	complexvalue "github.com/tsoniclang/gotots/internal/emit/value/complex"
	floatvalue "github.com/tsoniclang/gotots/internal/emit/value/float"
	integervalue "github.com/tsoniclang/gotots/internal/emit/value/integer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(context api.Context, source ast.Expr) (api.TypeEmission, error) {
	sourceType := context.TypesInfo().TypeOf(source)
	if sourceType == nil {
		return api.TypeEmission{}, api.Unsupported(context, api.CategoryType, source)
	}
	return EmitRepresented(context, source, sourceType)
}

func EmitRepresented(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
) (api.TypeEmission, error) {
	if SupportsUnsafePointer(sourceType) {
		reference, err := context.Names().Runtime(
			api.RuntimeUnsafePointer,
			api.ImportPhaseType,
		)
		if err != nil {
			return api.TypeEmission{}, err
		}
		return api.DirectType(
			context.Factory().UnionTypeNode(
				[]tsgo.TypeNode{
					context.Factory().TypeReferenceNode(
						reference.EntityName(context.Factory()),
						nil,
					),
					context.Factory().KeywordTypeNode(
						tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
					),
				},
			),
			reference.Requests()...,
		), nil
	}
	if _, ok := complexvalue.Describe(sourceType); ok {
		return complexvalue.EmitType(context, source, sourceType)
	}
	alias, ok := PrimitiveAlias(context.TypesSizes(), sourceType)
	if !ok {
		return api.TypeEmission{}, api.Unsupported(context, api.CategoryType, source)
	}
	reference, err := context.Names().Primitive(alias)
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

func SupportsUnsafePointer(sourceType types.Type) bool {
	basic, ok := types.Unalias(sourceType).(*types.Basic)
	return ok && basic.Kind() == types.UnsafePointer
}

func SupportsInteger(sizes types.Sizes, sourceType types.Type) bool {
	_, ok := integervalue.Describe(sizes, sourceType)
	return ok
}

func PrimitiveAlias(
	sizes types.Sizes,
	sourceType types.Type,
) (api.PrimitiveAlias, bool) {
	if sourceType != nil {
		if basic, ok := types.Unalias(sourceType).(*types.Basic); ok {
			switch basic.Kind() {
			case types.Bool:
				return api.PrimitiveBool, true
			case types.String:
				return api.PrimitiveString, true
			}
		}
	}
	if floatCarrier, ok := floatvalue.Describe(sourceType); ok {
		return floatCarrier.Alias(), true
	}
	carrier, ok := integervalue.Describe(sizes, sourceType)
	if !ok {
		return api.PrimitiveInvalid, false
	}
	return carrier.Alias(), true
}

func SupportsString(sourceType types.Type) bool {
	basic, ok := types.Unalias(sourceType).(*types.Basic)
	return ok && basic.Info()&types.IsString != 0
}

func SupportsStringIndex(sizes types.Sizes, sourceType types.Type) bool {
	if basic, ok := types.Unalias(sourceType).(*types.Basic); ok &&
		basic.Info()&types.IsUntyped != 0 &&
		basic.Info()&types.IsInteger != 0 {
		sourceType = types.Typ[types.Int]
	}
	_, ok := integervalue.Describe(sizes, sourceType)
	return ok
}
