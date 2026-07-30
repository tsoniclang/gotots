package interfacevalue

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	interfaceadapter "github.com/tsoniclang/gotots/internal/emit/declaration/interfaceadapter"
	interfacecontract "github.com/tsoniclang/gotots/internal/emit/runtime/interfacevalue/contract"
	panicnilruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panicnil"
	interfacetype "github.com/tsoniclang/gotots/internal/emit/type/interfacevalue"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Test(
	context api.Context,
	sourceType types.Type,
	targetType types.Type,
	value tsgo.Expression,
) (api.ExpressionEmission, error) {
	switch context.GoRuntimeType(targetType) {
	case api.GoRuntimeTypeBuiltinError:
		return runtimeInterfaceTest(context, api.RuntimeBuiltinErrorGuard, value)
	case api.GoRuntimeTypeError:
		return runtimeInterfaceTest(context, api.RuntimeErrorGuard, value)
	case api.GoRuntimeTypePanicNilPointer:
		reference, err := context.Names().Runtime(
			api.RuntimePanicNilValue,
			api.ImportPhaseValue,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return api.DirectExpression(
			context.Factory().CallExpression(
				context.Factory().PropertyAccessExpression(
					context.Factory().Identifier(reference.Name()),
					nil,
					context.Factory().Identifier(panicnilruntime.GuardName),
					tsgo.NodeFlagsNone,
				),
				nil,
				nil,
				[]tsgo.Expression{value},
				tsgo.NodeFlagsNone,
			),
			reference.Requests()...,
		), nil
	}
	if _, ok := interfacetype.Resolve(targetType); ok {
		demands, err := context.Names().InterfaceContractDemand(
			sourceType,
			targetType,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		contract, err := context.Names().InterfaceContract(targetType)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return api.DirectExpression(
			context.Factory().CallExpression(
				context.Factory().Identifier(contract.GuardName()),
				nil,
				nil,
				[]tsgo.Expression{value},
				tsgo.NodeFlagsNone,
			),
			api.CombineRequests(contract.Requests(), demands)...,
		), nil
	}
	adapter, err := context.Names().InterfaceAdapter(targetType, nil)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.DirectExpression(
		context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				context.Factory().Identifier(adapter.Name()),
				nil,
				context.Factory().Identifier(interfaceadapter.GuardMember),
				tsgo.NodeFlagsNone,
			),
			nil,
			nil,
			[]tsgo.Expression{value},
			tsgo.NodeFlagsNone,
		),
		adapter.Requests()...,
	), nil
}

func Extract(
	context api.Context,
	source ast.Node,
	targetType types.Type,
	value tsgo.Expression,
) (api.ExpressionEmission, error) {
	switch context.GoRuntimeType(targetType) {
	case api.GoRuntimeTypeBuiltinError:
		return runtimeInterfaceExtract(context, api.RuntimeBuiltinErrorType, value)
	case api.GoRuntimeTypeError:
		return runtimeInterfaceExtract(context, api.RuntimeErrorType, value)
	case api.GoRuntimeTypePanicNilPointer:
		reference, err := context.Names().Runtime(
			api.RuntimePanicNilValue,
			api.ImportPhaseType,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return api.DirectExpression(
			context.Factory().PropertyAccessExpression(
				value,
				nil,
				context.Factory().Identifier(
					interfacecontract.PayloadMember,
				),
				tsgo.NodeFlagsNone,
			),
			reference.Requests()...,
		), nil
	}
	if _, ok := interfacetype.Resolve(targetType); ok {
		contract, err := context.Names().InterfaceContract(targetType)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return api.DirectExpression(value, contract.Requests()...), nil
	}
	adapter, err := context.Names().InterfaceAdapter(targetType, nil)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	payload := context.Factory().PropertyAccessExpression(
		value,
		nil,
		context.Factory().Identifier(interfaceadapter.ValueMember),
		tsgo.NodeFlagsNone,
	)
	target, err := context.Values().Copy(
		context,
		source,
		targetType,
		api.DirectExpression(payload),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if len(target.Before()) != 0 {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	return api.NewExpressionEmission(
		nil,
		target.Value(),
		api.CombineRequests(target.Requests(), adapter.Requests()),
	)
}

func runtimeInterfaceTest(
	context api.Context,
	symbol api.RuntimeSymbol,
	value tsgo.Expression,
) (api.ExpressionEmission, error) {
	reference, err := context.Names().Runtime(symbol, api.ImportPhaseValue)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.DirectExpression(
		context.Factory().CallExpression(
			context.Factory().Identifier(reference.Name()),
			nil,
			nil,
			[]tsgo.Expression{value},
			tsgo.NodeFlagsNone,
		),
		reference.Requests()...,
	), nil
}

func runtimeInterfaceExtract(
	context api.Context,
	symbol api.RuntimeSymbol,
	value tsgo.Expression,
) (api.ExpressionEmission, error) {
	reference, err := context.Names().Runtime(symbol, api.ImportPhaseType)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.DirectExpression(value, reference.Requests()...), nil
}
