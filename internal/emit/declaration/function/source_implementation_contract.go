package function

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func sourceImplementationContractBody(
	context api.Context,
	function *types.Func,
) (api.BlockEmission, error) {
	identity := "certified source implementation owns " + function.FullName()
	panicReference, err := context.Names().Runtime(
		api.RuntimePanic,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.BlockEmission{}, err
	}
	return api.DirectBlock(
		context.Factory().Block(
			[]tsgo.Statement{context.Factory().ReturnStatement(
				panicruntime.Call(
					context.Factory(),
					panicReference.Name(),
					context.Factory().StringLiteral(
						identity,
						tsgo.TokenFlagsNone,
					),
				),
			)},
			true,
		),
		panicReference.Requests()...,
	), nil
}
