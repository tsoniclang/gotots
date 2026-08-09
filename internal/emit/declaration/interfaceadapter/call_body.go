package interfaceadapter

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func adapterCallBody(
	context api.Context,
	signature *types.Signature,
	call api.ExpressionEmission,
) []tsgo.Statement {
	body := call.Before()
	if signature.Results().Len() == 0 {
		return append(body, context.Factory().ExpressionStatement(call.Value()))
	}
	return append(body, context.Factory().ReturnStatement(call.Value()))
}
