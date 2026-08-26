package channel

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Resolve(sourceType types.Type) (*types.Chan, bool) {
	if sourceType == nil {
		return nil, false
	}
	channel, ok := types.Unalias(sourceType).(*types.Chan)
	return channel, ok
}

func EmitSyntax(
	context api.Context,
	children api.ChildEmitter,
	source *ast.ChanType,
	sourceType types.Type,
) (api.TypeEmission, error) {
	channel, ok := Resolve(sourceType)
	if !ok || source == nil || source.Value == nil {
		return api.TypeEmission{},
			api.Unsupported(context, api.CategoryType, source)
	}
	return emit(context, children, source.Value, channel)
}

func EmitRepresented(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	sourceType types.Type,
) (api.TypeEmission, error) {
	channel, ok := Resolve(sourceType)
	if !ok {
		return api.TypeEmission{},
			api.Unsupported(context, api.CategoryType, source)
	}
	return emit(context, children, source, channel)
}

func emit(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	channel *types.Chan,
) (api.TypeEmission, error) {
	element, err := children.RepresentedType(
		context.WithRole(api.RoleChannelElementType),
		source,
		channel.Elem(),
	)
	if err != nil {
		return api.TypeEmission{}, err
	}
	symbol, ok := runtimeSymbol(channel.Dir())
	if !ok {
		return api.TypeEmission{},
			api.Unsupported(context, api.CategoryType, source)
	}
	runtime, err := context.Names().Runtime(symbol, api.ImportPhaseType)
	if err != nil {
		return api.TypeEmission{}, err
	}
	target := context.Factory().TypeReferenceNode(
		context.Factory().Identifier(runtime.Name()),
		[]tsgo.TypeNode{element.Value()},
	)
	return api.DirectType(
		context.Factory().UnionTypeNode([]tsgo.TypeNode{
			target,
			context.Factory().KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
			),
		}),
		api.CombineRequests(
			element.Requests(),
			runtime.Requests(),
		)...,
	), nil
}

func runtimeSymbol(direction types.ChanDir) (api.RuntimeSymbol, bool) {
	switch direction {
	case types.SendRecv:
		return api.RuntimeChannel, true
	case types.RecvOnly:
		return api.RuntimeReceiveChannel, true
	case types.SendOnly:
		return api.RuntimeSendChannel, true
	default:
		return api.RuntimeInvalid, false
	}
}
