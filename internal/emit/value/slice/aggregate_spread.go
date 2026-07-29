package slicevalue

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	runtimeslice "github.com/tsoniclang/gotots/internal/emit/runtime/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func AppendSpreadAggregate(
	context api.Context,
	targetElement api.TypeEmission,
	source ast.Node,
	elementType types.Type,
	operands []tsgo.Expression,
	before []tsgo.Statement,
	requests []api.RootRequest,
) (api.ExpressionEmission, error) {
	if len(operands) != 2 {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "aggregate spread append requires two slices",
		}
	}
	names, err := spreadTemporaryNames(context)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	receiver := context.Factory().Identifier(names.receiver)
	appended := context.Factory().Identifier(names.appended)
	snapshot := context.Factory().Identifier(names.snapshot)
	result := context.Factory().Identifier(names.result)
	newLength := context.Factory().Identifier(names.length)
	index := context.Factory().Identifier(names.index)
	snapshotNext, existingCopy, tailZero, err :=
		spreadElementOperations(
			context,
			source,
			elementType,
			receiver,
			appended,
			index,
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
			names.receiver,
			operands[0],
		),
		sliceVariable(
			context,
			tsgo.NodeFlagsConst,
			names.appended,
			operands[1],
		),
		sliceVariable(
			context,
			tsgo.NodeFlagsLet,
			names.snapshot,
			appended,
		),
		context.Factory().IfStatement(
			sliceBinary(
				context,
				sliceProperty(
					context,
					appended,
					runtimeslice.MemberName(runtimeslice.MemberLength),
				),
				tsgo.BinaryOperatorGreaterThanToken,
				sliceNumber(context, "0"),
			),
			context.Factory().Block(
				spreadSnapshotStatements(
					context,
					appended,
					snapshot,
					index,
					snapshotNext,
					targetElement.Value(),
					allocation.Name(),
				),
				true,
			),
			nil,
		),
		sliceVariable(
			context,
			tsgo.NodeFlagsConst,
			names.length,
			sliceBinary(
				context,
				sliceProperty(
					context,
					receiver,
					runtimeslice.MemberName(runtimeslice.MemberLength),
				),
				tsgo.BinaryOperatorPlusToken,
				sliceProperty(
					context,
					snapshot,
					runtimeslice.MemberName(runtimeslice.MemberLength),
				),
			),
		),
		sliceVariable(
			context,
			tsgo.NodeFlagsLet,
			names.result,
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
				spreadReuseStatements(
					context,
					receiver,
					snapshot,
					result,
					newLength,
					index,
				),
				true,
			),
			context.Factory().Block(
				spreadGrowthStatements(
					context,
					receiver,
					snapshot,
					result,
					newLength,
					index,
					existingCopy,
					tailZero,
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
			snapshotNext.Requests(),
			existingCopy.Requests(),
			tailZero.Requests(),
			allocation.Requests(),
			runtime.Requests(),
		),
	)
}

type spreadNames struct {
	receiver string
	appended string
	snapshot string
	result   string
	length   string
	index    string
}

func spreadTemporaryNames(context api.Context) (spreadNames, error) {
	names := make([]string, 6)
	for index := range names {
		name, err := context.Names().Temporary(api.TemporarySliceConstruction)
		if err != nil {
			return spreadNames{}, err
		}
		names[index] = name
	}
	return spreadNames{
		receiver: names[0],
		appended: names[1],
		snapshot: names[2],
		result:   names[3],
		length:   names[4],
		index:    names[5],
	}, nil
}

func spreadElementOperations(
	context api.Context,
	source ast.Node,
	elementType types.Type,
	receiver tsgo.Expression,
	appended tsgo.Expression,
	index tsgo.Expression,
) (
	api.ExpressionEmission,
	api.ExpressionEmission,
	api.ExpressionEmission,
	error,
) {
	copyElement := func(value tsgo.Expression) (api.ExpressionEmission, error) {
		return context.Values().Copy(
			context.WithRole(api.RoleSliceElement),
			nil,
			elementType,
			api.DirectExpression(value),
		)
	}
	snapshotNext, err := copyElement(sliceCall(
		context,
		appended,
		runtimeslice.MemberName(runtimeslice.MemberGet),
		index,
	))
	if err != nil {
		return api.ExpressionEmission{}, api.ExpressionEmission{},
			api.ExpressionEmission{}, err
	}
	existingCopy, err := copyElement(sliceCall(
		context,
		receiver,
		runtimeslice.MemberName(runtimeslice.MemberGet),
		index,
	))
	if err != nil {
		return api.ExpressionEmission{}, api.ExpressionEmission{},
			api.ExpressionEmission{}, err
	}
	tailZero, err := context.Values().Zero(
		context.WithRole(api.RoleSliceElement),
		source,
		elementType,
	)
	return snapshotNext, existingCopy, tailZero, err
}

func spreadSnapshotStatements(
	context api.Context,
	appended tsgo.Expression,
	snapshot tsgo.Expression,
	index tsgo.Identifier,
	next api.ExpressionEmission,
	targetElement tsgo.TypeNode,
	allocationName string,
) []tsgo.Statement {
	length := sliceProperty(
		context,
		appended,
		runtimeslice.MemberName(runtimeslice.MemberLength),
	)
	statements := []tsgo.Statement{sliceAssign(
		context,
		snapshot,
		context.Factory().CallExpression(
			context.Factory().Identifier(allocationName),
			nil,
			[]tsgo.TypeNode{targetElement},
			[]tsgo.Expression{
				length,
				context.Factory().NullLiteral(),
			},
			tsgo.NodeFlagsNone,
		),
	)}
	return append(statements, sliceLoop(
		context,
		index,
		length,
		"0",
		append(
			next.Before(),
			context.Factory().ExpressionStatement(sliceCall(
				context,
				snapshot,
				runtimeslice.MemberName(runtimeslice.MemberSet),
				index,
				next.Value(),
			)),
		),
	))
}

func spreadReuseStatements(
	context api.Context,
	receiver tsgo.Expression,
	snapshot tsgo.Expression,
	result tsgo.Expression,
	newLength tsgo.Expression,
	index tsgo.Identifier,
) []tsgo.Statement {
	base := sliceProperty(
		context,
		receiver,
		runtimeslice.MemberName(runtimeslice.MemberLength),
	)
	return []tsgo.Statement{
		sliceAssign(
			context,
			result,
			sliceCall(
				context,
				receiver,
				runtimeslice.StorageWithLengthMember,
				newLength,
			),
		),
		sliceLoop(
			context,
			index,
			sliceProperty(
				context,
				snapshot,
				runtimeslice.MemberName(runtimeslice.MemberLength),
			),
			"0",
			[]tsgo.Statement{context.Factory().ExpressionStatement(sliceCall(
				context,
				result,
				runtimeslice.MemberName(runtimeslice.MemberSet),
				sliceBinary(
					context,
					base,
					tsgo.BinaryOperatorPlusToken,
					index,
				),
				sliceCall(
					context,
					snapshot,
					runtimeslice.MemberName(runtimeslice.MemberGet),
					index,
				),
			))},
		),
	}
}

func spreadGrowthStatements(
	context api.Context,
	receiver tsgo.Expression,
	snapshot tsgo.Expression,
	result tsgo.Expression,
	newLength tsgo.Expression,
	index tsgo.Identifier,
	existingCopy api.ExpressionEmission,
	tailZero api.ExpressionEmission,
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
	statements = append(statements, sliceLoop(
		context,
		index,
		sliceProperty(
			context,
			snapshot,
			runtimeslice.MemberName(runtimeslice.MemberLength),
		),
		"0",
		[]tsgo.Statement{context.Factory().ExpressionStatement(sliceCall(
			context,
			result,
			runtimeslice.MemberName(runtimeslice.MemberSet),
			sliceBinary(
				context,
				base,
				tsgo.BinaryOperatorPlusToken,
				index,
			),
			sliceCall(
				context,
				snapshot,
				runtimeslice.MemberName(runtimeslice.MemberGet),
				index,
			),
		))},
	))
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
