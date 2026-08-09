package array

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	pointermarker "github.com/tsoniclang/gotots/internal/emit/marker/pointer"
	arraymember "github.com/tsoniclang/gotots/internal/emit/runtime/array/member"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (a RuntimeArray) Address(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	parent api.ExpressionEmission,
	index api.ExpressionEmission,
	checkNil bool,
) (api.ExpressionEmission, error) {
	parentValue, indexValue, before, requests, err := captureAddressOperands(
		context,
		parent,
		index,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	logical, err := a.EmitType(
		context.WithRole(api.RoleArrayReceiver),
		children,
		source,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	parentPointer := api.DirectExpression(parentValue)
	if checkNil {
		parentPointer, err = pointermarker.Guard(context, parentPointer)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	loaded, err := pointermarker.Operation(
		context,
		tsoniccore.SymbolLoadPointer,
		[]api.TypeEmission{logical},
		[]api.ExpressionEmission{parentPointer},
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	parentValue, before, requests, err = captureAddressValue(
		context,
		loaded,
		before,
		api.CombineRequests(requests, logical.Requests()),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	stored, err := a.storage(
		context.WithRole(api.RoleArrayReceiver),
		api.DirectExpression(parentValue),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	storedValue, before, requests, err := captureAddressValue(
		context,
		stored,
		before,
		requests,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	before = append(before, context.Factory().ExpressionStatement(callMember(
		context,
		storedValue,
		arraymember.Get,
		indexValue,
	)))
	locationReference, err := context.Names().Runtime(
		api.RuntimeArrayLocation,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	location, before, requests, err := captureAddressValue(
		context,
		api.DirectExpression(
			context.Factory().CallExpression(
				locationReference.Expression(context.Factory()),
				nil,
				nil,
				[]tsgo.Expression{storedValue},
				tsgo.NodeFlagsNone,
			),
			locationReference.Requests()...,
		),
		before,
		requests,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	storageType, err := context.ContainerStorage().ContainerStorageType(
		context.WithRole(api.RoleStorageType),
		source,
		a.ElementType(),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	locationPart := func(index string) tsgo.ElementAccessExpression {
		return context.Factory().ElementAccessExpression(
			location,
			nil,
			context.Factory().NumericLiteral(index, tsgo.TokenFlagsNone),
			tsgo.NodeFlagsNone,
		)
	}
	numericIndex := context.Factory().CallExpression(
		api.TargetIntrinsicNumber.Expression(context.Factory()),
		nil,
		nil,
		[]tsgo.Expression{indexValue},
		tsgo.NodeFlagsNone,
	)
	storageLocation := context.Factory().ElementAccessExpression(
		locationPart("0"),
		nil,
		context.Factory().BinaryExpression(
			nil,
			locationPart("1"),
			nil,
			context.Factory().BinaryOperatorToken(
				tsgo.BinaryOperatorPlusToken,
			),
			numericIndex,
		),
		tsgo.NodeFlagsNone,
	)
	storagePointer, err := pointermarker.Operation(
		context,
		tsoniccore.SymbolAddressOf,
		[]api.TypeEmission{storageType},
		[]api.ExpressionEmission{api.DirectExpression(storageLocation)},
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	projected, err := context.Values().ProjectStoragePointer(
		context,
		source,
		a.ElementType(),
		storagePointer,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		append(before, projected.Before()...),
		projected.Value(),
		api.CombineRequests(
			requests,
			storageType.Requests(),
			projected.Requests(),
		),
	)
}

func captureAddressOperands(
	context api.Context,
	parent api.ExpressionEmission,
	index api.ExpressionEmission,
) (tsgo.Expression, tsgo.Expression, []tsgo.Statement, []api.RootRequest, error) {
	parentValue, before, requests, err := captureAddressValue(
		context,
		parent,
		nil,
		nil,
	)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	indexValue, before, requests, err := captureAddressValue(
		context,
		index,
		before,
		requests,
	)
	return parentValue, indexValue, before, requests, err
}

func captureAddressValue(
	context api.Context,
	value api.ExpressionEmission,
	before []tsgo.Statement,
	requests []api.RootRequest,
) (tsgo.Expression, []tsgo.Statement, []api.RootRequest, error) {
	name, err := context.Names().Temporary(api.TemporaryAddressOperand)
	if err != nil {
		return nil, nil, nil, err
	}
	before = append(before, value.Before()...)
	before = append(before, context.Factory().VariableStatement(
		nil,
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{context.Factory().VariableDeclaration(
				context.Factory().Identifier(name),
				nil,
				nil,
				value.Value(),
			)},
			tsgo.NodeFlagsConst,
		),
	))
	return context.Factory().Identifier(name),
		before,
		api.CombineRequests(requests, value.Requests()),
		nil
}
