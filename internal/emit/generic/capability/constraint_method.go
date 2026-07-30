package capability

import (
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	cooperativecall "github.com/tsoniclang/gotots/internal/emit/concurrency/cooperative"
	"github.com/tsoniclang/gotots/internal/emit/expression/call/interfaceoperation"
	"github.com/tsoniclang/gotots/internal/emit/value/interfacevalue"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitConstraintMethod(
	context api.Context,
	children api.ChildEmitter,
	operation api.GenericOperation,
	selection api.GenericOperationSelection,
	signature *types.Signature,
	arguments []tsgo.Expression,
) (api.ExpressionEmission, error) {
	method, ok := selection.Method()
	if !ok ||
		signature == nil ||
		len(arguments) == 0 ||
		signature.Params().Len() != len(arguments) {
		return api.ExpressionEmission{}, shapeError(context, operation)
	}
	interfaceType, concreteMethod, methodSignature, err :=
		concreteConstraintMethod(method, signature)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	receiverType := signature.Params().At(0).Type()
	receiver, handled, err := interfacevalue.Convert(
		context,
		nil,
		receiverType,
		interfaceType,
		api.DirectExpression(arguments[0]),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if !handled {
		return api.ExpressionEmission{}, invariant(
			context,
			"generic constraint-method receiver has no concrete interface adaptation",
		)
	}
	target, err := interfaceoperation.Apply(
		context,
		children,
		nil,
		interfaceType,
		receiver,
		concreteMethod,
		arguments[1:],
		nil,
		nil,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	valueSignature, ok := callable.ValueSignature(methodSignature)
	if !ok {
		return api.ExpressionEmission{}, shapeError(context, operation)
	}
	return cooperativecall.GeneratedValueCall(
		context,
		valueSignature,
		target,
	)
}

func concreteConstraintMethod(
	method *types.Func,
	operation *types.Signature,
) (*types.Interface, *types.Func, *types.Signature, error) {
	methodSignature, ok := method.Type().(*types.Signature)
	if !ok ||
		methodSignature.Recv() == nil ||
		operation == nil ||
		operation.Params().Len() != methodSignature.Params().Len()+1 ||
		operation.Results().Len() != methodSignature.Results().Len() {
		return nil, nil, nil, &api.InvariantError{
			Role:   api.RoleFunctionBody,
			Reason: "constraint-method capability signature is inconsistent",
		}
	}
	parameters := concreteMethodTuple(
		methodSignature.Params(),
		operation.Params(),
		1,
	)
	results := concreteMethodTuple(
		methodSignature.Results(),
		operation.Results(),
		0,
	)
	concreteSignature := types.NewSignatureType(
		nil,
		nil,
		nil,
		parameters,
		results,
		methodSignature.Variadic(),
	)
	concreteMethod := types.NewFunc(
		method.Pos(),
		method.Pkg(),
		method.Name(),
		concreteSignature,
	)
	contract := types.NewInterfaceType(
		[]*types.Func{concreteMethod},
		nil,
	).Complete()
	return contract, concreteMethod, concreteSignature, nil
}

func concreteMethodTuple(
	source *types.Tuple,
	concrete *types.Tuple,
	offset int,
) *types.Tuple {
	variables := make([]*types.Var, 0, source.Len())
	for index := range source.Len() {
		template := source.At(index)
		variables = append(variables, types.NewVar(
			token.NoPos,
			template.Pkg(),
			template.Name(),
			concrete.At(index+offset).Type(),
		))
	}
	return types.NewTuple(variables...)
}
