package goruntime

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
) (api.TypeEmission, bool, error) {
	return emit(context, source, sourceType, true)
}

func EmitNonNil(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
) (api.TypeEmission, bool, error) {
	return emit(context, source, sourceType, false)
}

func emit(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	includeNil bool,
) (api.TypeEmission, bool, error) {
	var symbol api.RuntimeSymbol
	nilCapable := false
	switch context.GoRuntimeType(sourceType) {
	case api.GoRuntimeTypeBuiltinError:
		symbol = api.RuntimeBuiltinErrorType
		nilCapable = true
	case api.GoRuntimeTypeError:
		symbol = api.RuntimeErrorType
		nilCapable = true
	case api.GoRuntimeTypePanicNilError:
		symbol = api.RuntimePanicNilError
	default:
		return api.TypeEmission{}, false, nil
	}
	reference, err := context.Names().Runtime(symbol, api.ImportPhaseType)
	if err != nil {
		return api.TypeEmission{}, true, err
	}
	target := tsgo.TypeNode(context.Factory().TypeReferenceNode(
		context.Factory().Identifier(reference.Name()),
		nil,
	))
	if nilCapable && includeNil {
		target = context.Factory().UnionTypeNode([]tsgo.TypeNode{
			target,
			context.Factory().KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
			),
		})
	}
	return api.DirectType(target, reference.Requests()...), true, nil
}
