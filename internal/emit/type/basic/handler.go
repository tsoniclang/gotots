package basic

import (
	"go/ast"
	"go/types"

	representationcontract "github.com/tsoniclang/gotots/internal/contracts/representation"
	"github.com/tsoniclang/gotots/internal/emit/api"
	rawpointermarker "github.com/tsoniclang/gotots/internal/emit/marker/rawpointer"
	complexvalue "github.com/tsoniclang/gotots/internal/emit/value/complex"
	integervalue "github.com/tsoniclang/gotots/internal/emit/value/integer"
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
		return rawpointermarker.Type(context, true)
	}
	if _, ok := complexvalue.Describe(sourceType); ok {
		return complexvalue.EmitType(context, source, sourceType)
	}
	alias, ok := PrimitiveAlias(context.TypesSizes(), sourceType)
	if !ok {
		return api.TypeEmission{}, api.Unsupported(context, api.CategoryType, source)
	}
	var reference api.NameReference
	var err error
	if context.ProviderScalarRepresentation() {
		reference, err = context.Names().ProviderPrimitive(alias)
	} else {
		reference, err = context.Names().Primitive(alias)
	}
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
	if sourceType == nil {
		return false
	}
	basic, ok := types.Unalias(sourceType).Underlying().(*types.Basic)
	return ok && basic.Kind() == types.UnsafePointer
}

func SupportsInteger(sizes types.Sizes, sourceType types.Type) bool {
	_, ok := integervalue.Describe(sizes, sourceType)
	return ok
}

func IntegerAlias(
	sizes types.Sizes,
	sourceType types.Type,
) (api.PrimitiveAlias, bool) {
	if !SupportsInteger(sizes, sourceType) {
		return api.PrimitiveInvalid, false
	}
	return PrimitiveAlias(sizes, sourceType)
}

func PrimitiveAlias(
	_ types.Sizes,
	sourceType types.Type,
) (api.PrimitiveAlias, bool) {
	return representationcontract.PrimitiveAliasFor(sourceType)
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
