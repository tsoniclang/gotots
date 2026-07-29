package interfacevalue

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Resolve(sourceType types.Type) (*types.Interface, bool) {
	if sourceType == nil {
		return nil, false
	}
	sourceType = types.Unalias(sourceType)
	if _, parameter := sourceType.(*types.TypeParam); parameter {
		return nil, false
	}
	source, ok := sourceType.Underlying().(*types.Interface)
	if !ok {
		return nil, false
	}
	source = source.Complete()
	return source, source.IsMethodSet()
}

func Emit(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
) (api.TypeEmission, error) {
	if _, ok := Resolve(sourceType); !ok {
		return api.TypeEmission{},
			api.Unsupported(context, api.CategoryType, source)
	}
	reference, err := context.Names().InterfaceType(sourceType)
	if err != nil {
		return api.TypeEmission{}, err
	}
	return api.DirectType(
		context.Factory().UnionTypeNode([]tsgo.TypeNode{
			context.Factory().TypeReferenceNode(
				context.Factory().Identifier(reference.Name()),
				nil,
			),
			context.Factory().KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
			),
		}),
		reference.Requests()...,
	), nil
}
