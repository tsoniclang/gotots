package array

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	pointermarker "github.com/tsoniclang/gotots/internal/emit/marker/pointer"
	arraymember "github.com/tsoniclang/gotots/internal/emit/runtime/array/member"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	runtimeslice "github.com/tsoniclang/gotots/internal/emit/runtime/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (a RuntimeArray) FromSlice(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	operand api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	sourceName, err := context.Names().Temporary(
		api.TemporaryArrayConstruction,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	resultName, err := context.Names().Temporary(
		api.TemporaryArrayConstruction,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	indexName, err := context.Names().Temporary(
		api.TemporaryArrayConstruction,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	sourceValue := context.Factory().Identifier(sourceName)
	result := context.Factory().Identifier(resultName)
	index := context.Factory().Identifier(indexName)
	panicReference, err := context.Names().Runtime(
		api.RuntimePanic,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	zero, err := a.Zero(context, children, source)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	storedZero, err := a.storage(
		context.WithRole(api.RoleConversionOperand),
		zero,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	element, err := a.loadElement(
		context,
		source,
		callSlice(
			context,
			sourceValue,
			runtimeslice.MemberName(runtimeslice.MemberGet),
			index,
		),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	elementCopy, err := context.Values().Transfer(
		context.WithRole(api.RoleArrayElement),
		nil,
		a.ElementType(),
		a.ElementType(),
		api.ValueTransferCopy,
		element,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	elementCopy, err = a.storeElement(context, source, elementCopy)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	before := append(
		operand.Before(),
		arrayComparisonVariable(
			context,
			tsgo.NodeFlagsConst,
			sourceName,
			operand.Value(),
		),
		context.Factory().IfStatement(
			context.Factory().BinaryExpression(
				nil,
				sliceProperty(
					context,
					sourceValue,
					runtimeslice.MemberName(runtimeslice.MemberLength),
				),
				nil,
				context.Factory().BinaryOperatorToken(
					tsgo.BinaryOperatorLessThanToken,
				),
				a.lengthLiteral(context),
			),
			context.Factory().Block([]tsgo.Statement{
				context.Factory().ExpressionStatement(panicruntime.Call(
					context.Factory(),
					panicReference.Name(),
					context.Factory().StringLiteral(
						"slice to array conversion out of range",
						tsgo.TokenFlagsNone,
					),
				)),
			}, true),
			nil,
		),
	)
	before = append(before, storedZero.Before()...)
	before = append(
		before,
		arrayComparisonVariable(
			context,
			tsgo.NodeFlagsConst,
			resultName,
			storedZero.Value(),
		),
		arrayConstructionLoop(
			context,
			index,
			a.lengthLiteral(context),
			"0",
			append(
				elementCopy.Before(),
				context.Factory().ExpressionStatement(callMember(
					context,
					result,
					arraymember.Set,
					index,
					elementCopy.Value(),
				)),
			),
		),
	)
	emission, err := api.NewExpressionEmission(
		before,
		result,
		api.CombineRequests(
			operand.Requests(),
			panicReference.Requests(),
			storedZero.Requests(),
			elementCopy.Requests(),
		),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return a.wrap(context, emission)
}

func (a RuntimeArray) PointerFromSlice(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	operand api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	logical, err := a.EmitType(
		context.WithRole(api.RoleConversionOperand),
		children,
		source,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	typeArguments, typeRequests, err := a.targetTypeArguments(
		context.WithRole(api.RoleConversionOperand),
		children,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	arrayRuntime, err := context.Names().Runtime(
		api.RuntimeArray,
		api.ImportPhaseType,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	storageType := api.DirectType(
		context.Factory().TypeReferenceNode(
			arrayRuntime.EntityName(context.Factory()),
			typeArguments,
		),
		api.CombineRequests(typeRequests, arrayRuntime.Requests())...,
	)
	runtime, err := context.Names().Runtime(
		api.RuntimeSliceArrayPointer,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	pointer, err := api.NewExpressionEmission(
		operand.Before(),
		context.Factory().CallExpression(
			context.Factory().Identifier(runtime.Name()),
			nil,
			[]tsgo.TypeNode{
				typeArguments[0],
				typeArguments[1],
			},
			[]tsgo.Expression{
				operand.Value(),
				a.lengthLiteral(context),
			},
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			operand.Requests(),
			typeRequests,
			runtime.Requests(),
		),
	)
	if err != nil || !a.nominal {
		return pointer, err
	}
	fromSource, err := a.wrap(
		context.WithRole(api.RoleConversionOperand),
		api.DirectExpression(context.Factory().Identifier("$go$source")),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	toSource, err := a.storage(
		context.WithRole(api.RoleConversionOperand),
		api.DirectExpression(context.Factory().Identifier("$go$target")),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return pointermarker.Operation(
		context,
		tsoniccore.SymbolProjectPointer,
		[]api.TypeEmission{storageType, logical},
		[]api.ExpressionEmission{
			pointer,
			api.DirectExpression(
				arrayPointerProjection(
					context,
					"$go$source",
					storageType.Value(),
					logical.Value(),
					fromSource,
				),
				fromSource.Requests()...,
			),
			api.DirectExpression(
				arrayPointerProjection(
					context,
					"$go$target",
					logical.Value(),
					storageType.Value(),
					toSource,
				),
				toSource.Requests()...,
			),
		},
	)
}

func arrayPointerProjection(
	context api.Context,
	parameter string,
	input tsgo.TypeNode,
	output tsgo.TypeNode,
	value api.ExpressionEmission,
) tsgo.ArrowFunction {
	statements := append(
		value.Before(),
		context.Factory().ReturnStatement(value.Value()),
	)
	return context.Factory().ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{context.Factory().ParameterDeclaration(
			nil,
			nil,
			context.Factory().Identifier(parameter),
			nil,
			input,
			nil,
		)},
		output,
		context.Factory().EqualsGreaterThanToken(),
		context.Factory().Block(statements, true),
	)
}

func sliceProperty(
	context api.Context,
	receiver tsgo.Expression,
	name string,
) tsgo.PropertyAccessExpression {
	return context.Factory().PropertyAccessExpression(
		receiver,
		nil,
		context.Factory().Identifier(name),
		tsgo.NodeFlagsNone,
	)
}

func callSlice(
	context api.Context,
	receiver tsgo.Expression,
	name string,
	arguments ...tsgo.Expression,
) tsgo.CallExpression {
	return context.Factory().CallExpression(
		sliceProperty(context, receiver, name),
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	)
}
