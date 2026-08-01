package function

import (
	"go/types"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/emit/api"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitExternalBody(
	context api.Context,
	function *types.Func,
) (api.BlockEmission, error) {
	contract, err := environmentcontract.Describe(function)
	if err != nil {
		return api.BlockEmission{}, err
	}
	panicReference, err := context.Names().Runtime(
		api.RuntimePanic,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.BlockEmission{}, err
	}
	return api.DirectBlock(
		context.Factory().Block(
			[]tsgo.Statement{
				context.Factory().ReturnStatement(
					panicruntime.Call(
						context.Factory(),
						panicReference.Name(),
						context.Factory().StringLiteral(
							"unresolved external Go function "+contract.Identity(),
							tsgo.TokenFlagsNone,
						),
					),
				),
			},
			true,
		),
		panicReference.Requests()...,
	), nil
}
