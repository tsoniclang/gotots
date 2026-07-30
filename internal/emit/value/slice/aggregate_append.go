package slicevalue

import (
	"go/ast"
	"go/types"
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
	runtimeslice "github.com/tsoniclang/gotots/internal/emit/runtime/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func AppendAggregate(
	context api.Context,
	targetElement api.TypeEmission,
	source ast.Node,
	elementType types.Type,
	operands []tsgo.Expression,
	before []tsgo.Statement,
	requests []api.RootRequest,
) (api.ExpressionEmission, error) {
	if len(operands) == 0 {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "aggregate append has no slice receiver",
		}
	}
	receiverName, resultName, lengthName, indexName, err :=
		appendTemporaryNames(context)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	receiver := context.Factory().Identifier(receiverName)
	result := context.Factory().Identifier(resultName)
	newLength := context.Factory().Identifier(lengthName)
	index := context.Factory().Identifier(indexName)
	tailZero, err := context.Values().Zero(
		context.WithRole(api.RoleSliceElement),
		source,
		elementType,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	existingCopy, err := context.Values().Transfer(
		context.WithRole(api.RoleSliceElement),
		nil,
		elementType,
		elementType,
		api.ValueTransferCopy,
		api.DirectExpression(sliceCall(
			context,
			receiver,
			runtimeslice.MemberName(runtimeslice.MemberGet),
			index,
		)),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	allocation, err := context.Names().Runtime(
		api.RuntimeSliceStorage,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	runtime, err := context.Names().Runtime(
		api.RuntimeSlice,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	statements := append([]tsgo.Statement(nil), before...)
	statements = append(
		statements,
		sliceVariable(
			context,
			tsgo.NodeFlagsConst,
			receiverName,
			operands[0],
		),
		sliceVariable(
			context,
			tsgo.NodeFlagsConst,
			lengthName,
			sliceBinary(
				context,
				sliceProperty(
					context,
					receiver,
					runtimeslice.MemberName(runtimeslice.MemberLength),
				),
				tsgo.BinaryOperatorPlusToken,
				sliceNumber(context, strconv.Itoa(len(operands)-1)),
			),
		),
		sliceVariable(
			context,
			tsgo.NodeFlagsLet,
			resultName,
			receiver,
		),
		context.Factory().IfStatement(
			sliceBinary(
				context,
				newLength,
				tsgo.BinaryOperatorLessThanEqualsToken,
				sliceProperty(
					context,
					receiver,
					runtimeslice.MemberName(runtimeslice.MemberCapacity),
				),
			),
			context.Factory().Block(
				appendReuseStatements(
					context,
					receiver,
					result,
					newLength,
					operands[1:],
				),
				true,
			),
			context.Factory().Block(
				appendGrowthStatements(
					context,
					receiver,
					result,
					newLength,
					index,
					operands[1:],
					tailZero,
					existingCopy,
					targetElement.Value(),
					allocation.Name(),
					runtime.Name(),
				),
				true,
			),
		),
	)
	return api.NewExpressionEmission(
		statements,
		result,
		api.CombineRequests(
			requests,
			targetElement.Requests(),
			tailZero.Requests(),
			existingCopy.Requests(),
			allocation.Requests(),
			runtime.Requests(),
		),
	)
}

func appendTemporaryNames(
	context api.Context,
) (string, string, string, string, error) {
	names := make([]string, 4)
	for index := range names {
		name, err := context.Names().Temporary(api.TemporarySliceConstruction)
		if err != nil {
			return "", "", "", "", err
		}
		names[index] = name
	}
	return names[0], names[1], names[2], names[3], nil
}

func appendReuseStatements(
	context api.Context,
	receiver tsgo.Expression,
	result tsgo.Expression,
	newLength tsgo.Expression,
	values []tsgo.Expression,
) []tsgo.Statement {
	statements := []tsgo.Statement{sliceAssign(
		context,
		result,
		sliceCall(
			context,
			receiver,
			runtimeslice.StorageWithLengthMember,
			newLength,
		),
	)}
	base := sliceProperty(
		context,
		receiver,
		runtimeslice.MemberName(runtimeslice.MemberLength),
	)
	for index, value := range values {
		statements = append(statements, context.Factory().ExpressionStatement(
			sliceCall(
				context,
				result,
				runtimeslice.MemberName(runtimeslice.MemberSet),
				sliceBinary(
					context,
					base,
					tsgo.BinaryOperatorPlusToken,
					sliceNumber(context, strconv.Itoa(index)),
				),
				value,
			),
		))
	}
	return statements
}

func appendGrowthStatements(
	context api.Context,
	receiver tsgo.Expression,
	result tsgo.Expression,
	newLength tsgo.Expression,
	index tsgo.Identifier,
	values []tsgo.Expression,
	tailZero api.ExpressionEmission,
	existingCopy api.ExpressionEmission,
	targetElement tsgo.TypeNode,
	allocationName string,
	runtimeName string,
) []tsgo.Statement {
	nextCapacity := context.Factory().CallExpression(
		sliceProperty(
			context,
			context.Factory().Identifier(runtimeName),
			runtimeslice.StorageGrownCapacityMember,
		),
		nil,
		nil,
		[]tsgo.Expression{
			sliceProperty(
				context,
				receiver,
				runtimeslice.MemberName(runtimeslice.MemberCapacity),
			),
			newLength,
		},
		tsgo.NodeFlagsNone,
	)
	statements := []tsgo.Statement{sliceAssign(
		context,
		result,
		context.Factory().CallExpression(
			context.Factory().Identifier(allocationName),
			nil,
			[]tsgo.TypeNode{targetElement},
			[]tsgo.Expression{newLength, nextCapacity},
			tsgo.NodeFlagsNone,
		),
	)}
	statements = append(statements, sliceLoop(
		context,
		index,
		sliceProperty(
			context,
			receiver,
			runtimeslice.MemberName(runtimeslice.MemberLength),
		),
		"0",
		append(
			existingCopy.Before(),
			context.Factory().ExpressionStatement(sliceCall(
				context,
				result,
				runtimeslice.MemberName(runtimeslice.MemberSet),
				index,
				existingCopy.Value(),
			)),
		),
	))
	base := sliceProperty(
		context,
		receiver,
		runtimeslice.MemberName(runtimeslice.MemberLength),
	)
	for offset, value := range values {
		statements = append(statements, context.Factory().ExpressionStatement(
			sliceCall(
				context,
				result,
				runtimeslice.MemberName(runtimeslice.MemberSet),
				sliceBinary(
					context,
					base,
					tsgo.BinaryOperatorPlusToken,
					sliceNumber(context, strconv.Itoa(offset)),
				),
				value,
			),
		))
	}
	statements = append(statements, sliceLoopFrom(
		context,
		index,
		sliceProperty(
			context,
			result,
			runtimeslice.MemberName(runtimeslice.MemberCapacity),
		),
		newLength,
		append(
			tailZero.Before(),
			context.Factory().ExpressionStatement(sliceCall(
				context,
				result,
				runtimeslice.StorageInitializeMember,
				index,
				tailZero.Value(),
			)),
		),
	))
	return statements
}
