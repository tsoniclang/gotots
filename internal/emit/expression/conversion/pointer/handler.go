package pointer

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	pointermarker "github.com/tsoniclang/gotots/internal/emit/marker/pointer"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Convert(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	sourceType types.Type,
	targetType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	sourcePointer, sourceDefined, sourceOK := resolve(sourceType)
	targetPointer, targetDefined, targetOK := resolve(targetType)
	if !sourceOK || !targetOK {
		return api.ExpressionEmission{}, false, nil
	}
	if !types.ConvertibleTo(sourceType, targetType) {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	var err error
	if sourceDefined.Type() != nil {
		value, err = sourceDefined.Project(context, value)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
	}
	sourceLogical, err := children.RepresentedType(
		context.WithRole(api.RoleConversionOperand),
		source.Args[0],
		sourcePointer.Elem(),
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	targetLogical, err := children.RepresentedType(
		context.WithRole(api.RoleConversionOperand),
		source.Fun,
		targetPointer.Elem(),
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	fromSource, err := convertPointee(
		context,
		source,
		sourcePointer.Elem(),
		targetPointer.Elem(),
		"$go$source",
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	toSource, err := convertPointee(
		context,
		source,
		targetPointer.Elem(),
		sourcePointer.Elem(),
		"$go$target",
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	target, err := pointermarker.Operation(
		context,
		tsoniccore.SymbolProjectPointer,
		[]api.TypeEmission{sourceLogical, targetLogical},
		[]api.ExpressionEmission{
			value,
			api.DirectExpression(
				conversionArrow(
					context,
					"$go$source",
					sourceLogical.Value(),
					targetLogical.Value(),
					fromSource,
				),
				fromSource.Requests()...,
			),
			api.DirectExpression(
				conversionArrow(
					context,
					"$go$target",
					targetLogical.Value(),
					sourceLogical.Value(),
					toSource,
				),
				toSource.Requests()...,
			),
		},
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	if targetDefined.Type() != nil {
		target, err = targetDefined.Wrap(context, target)
	}
	return target, true, err
}

func convertPointee(
	context api.Context,
	source ast.Node,
	sourceElement types.Type,
	targetElement types.Type,
	parameter string,
) (api.ExpressionEmission, error) {
	stored, err := context.Values().ToStorage(
		context.WithRole(api.RoleConversionOperand),
		source,
		sourceElement,
		api.DirectExpression(context.Factory().Identifier(parameter)),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	converted, err := context.Values().FromStorage(
		context.WithRole(api.RoleConversionOperand),
		source,
		targetElement,
		stored,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return converted, nil
}

func conversionArrow(
	context api.Context,
	parameter string,
	sourceType tsgo.TypeNode,
	targetType tsgo.TypeNode,
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
			sourceType,
			nil,
		)},
		targetType,
		context.Factory().EqualsGreaterThanToken(),
		context.Factory().Block(statements, true),
	)
}

func resolve(sourceType types.Type) (*types.Pointer, definedtype.Model, bool) {
	if pointer, ok := types.Unalias(sourceType).(*types.Pointer); ok {
		return pointer, definedtype.Model{}, true
	}
	defined, ok := definedtype.ResolvePointer(sourceType)
	if !ok {
		return nil, definedtype.Model{}, false
	}
	pointer, _ := defined.Pointer()
	return pointer, defined, true
}
