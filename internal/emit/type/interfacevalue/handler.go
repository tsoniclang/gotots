package interfacevalue

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	genericinstance "github.com/tsoniclang/gotots/internal/emit/generic/instance"
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
	children api.ChildEmitter,
	source ast.Node,
	sourceType types.Type,
) (api.TypeEmission, error) {
	target, err := EmitNonNil(context, children, source, sourceType)
	if err != nil {
		return api.TypeEmission{}, err
	}
	return api.DirectType(
		context.Factory().UnionTypeNode([]tsgo.TypeNode{
			target.Value(),
			context.Factory().KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
			),
		}),
		target.Requests()...,
	), nil
}

func EmitNonNil(
	context api.Context,
	children api.ChildEmitter,
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
	var arguments []tsgo.TypeNode
	requests := reference.Requests()
	named, namedType := types.Unalias(sourceType).(*types.Named)
	if namedType && named.TypeArgs().Len() != 0 {
		var argumentRequests []api.RootRequest
		arguments, argumentRequests, err = genericinstance.EmitTypeArguments(
			context,
			children,
			source,
			named.Origin().Obj(),
			named.TypeArgs(),
		)
		if err != nil {
			return api.TypeEmission{}, err
		}
		requests = api.CombineRequests(requests, argumentRequests)
	}
	return api.DirectType(
		context.Factory().TypeReferenceNode(
			context.Factory().Identifier(reference.Name()),
			arguments,
		),
		requests...,
	), nil
}
