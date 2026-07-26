package pointer

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Scalar(
	sizes types.Sizes,
	sourceType types.Type,
) (*types.Pointer, types.Type, bool) {
	if sizes == nil || sourceType == nil {
		return nil, nil, false
	}
	pointer, ok := types.Unalias(sourceType).(*types.Pointer)
	if !ok {
		return nil, nil, false
	}
	element := pointer.Elem()
	if _, represented := api.PrimitiveAliasFor(sizes, element); !represented {
		return nil, nil, false
	}
	return pointer, element, true
}

func EmitSyntax(
	context api.Context,
	children api.ChildEmitter,
	source *ast.StarExpr,
	sourceType types.Type,
) (api.TypeEmission, error) {
	pointer, element, ok := Scalar(context.TypesSizes(), sourceType)
	if !ok ||
		source == nil ||
		source.X == nil ||
		!types.Identical(context.TypesInfo().TypeOf(source), pointer) ||
		!types.Identical(context.TypesInfo().TypeOf(source.X), element) {
		return api.TypeEmission{},
			api.Unsupported(context, api.CategoryType, source)
	}
	return EmitRepresented(context, children, source, pointer)
}

func EmitRepresented(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	sourceType types.Type,
) (api.TypeEmission, error) {
	_, element, ok := Scalar(context.TypesSizes(), sourceType)
	if !ok {
		return api.TypeEmission{},
			api.Unsupported(context, api.CategoryType, source)
	}
	elementType, err := children.RepresentedType(context, source, element)
	if err != nil {
		return api.TypeEmission{}, err
	}
	reference, err := context.Names().Runtime(
		api.RuntimePointer,
		api.ImportPhaseType,
	)
	if err != nil {
		return api.TypeEmission{}, err
	}
	cell := context.Factory().TypeReferenceNode(
		context.Factory().Identifier(reference.Name()),
		[]tsgo.TypeNode{elementType.Value()},
	)
	return api.DirectType(
		context.Factory().UnionTypeNode([]tsgo.TypeNode{
			cell,
			context.Factory().KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
			),
		}),
		api.CombineRequests(
			elementType.Requests(),
			reference.Requests(),
		)...,
	), nil
}
