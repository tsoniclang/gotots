package compositeliteral

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	genericinstance "github.com/tsoniclang/gotots/internal/emit/generic/instance"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func genericNamedStructTypeArguments(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	named *types.Named,
	requests []api.RootRequest,
) ([]tsgo.TypeNode, []api.RootRequest, error) {
	if named == nil ||
		named.TypeParams().Len() == 0 ||
		named.TypeArgs().Len() != named.TypeParams().Len() {
		return nil, nil, &api.InvariantError{
			Role:   context.Role(),
			Reason: "generic storage construction type is invalid",
		}
	}
	arguments, argumentRequests, err := genericinstance.EmitTypeArguments(
		context,
		children,
		source,
		named.Origin().Obj(),
		named.TypeArgs(),
	)
	if err != nil {
		return nil, nil, err
	}
	return arguments,
		api.CombineRequests(requests, argumentRequests),
		nil
}
