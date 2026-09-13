package memory

import (
	"go/ast"
	"go/types"
	"strconv"

	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	pointermarker "github.com/tsoniclang/gotots/internal/emit/marker/pointer"
	"github.com/tsoniclang/gotots/internal/emit/value/structconstruction"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func RecordBindingLayout(context api.Context, children api.ChildEmitter, source ast.Node, sourceType types.Type, physical bool) (api.ExpressionEmission, api.TypeEmission, []api.ExpressionEmission, error) {
	structure, ok := sourceType.Underlying().(*types.Struct)
	if !ok {
		return api.ExpressionEmission{}, api.TypeEmission{}, nil, api.Unsupported(context, api.CategoryExpression, source)
	}
	return recordBindingLayout(context, children, source, sourceType, structure, physical, true)
}

func recordBindingLayout(context api.Context, children api.ChildEmitter, source ast.Node, sourceType types.Type, structure *types.Struct, physical, capture bool) (api.ExpressionEmission, api.TypeEmission, []api.ExpressionEmission, error) {
	represented, err := bindingStorageType(context, source, sourceType, physical)
	if err != nil {
		return api.ExpressionEmission{}, api.TypeEmission{}, nil, err
	}
	rootType, before, err := recordSchema(context, source, structure, represented, physical)
	if err != nil {
		return api.ExpressionEmission{}, api.TypeEmission{}, nil, err
	}
	arguments, err := layoutDimensions(context, source, sourceType)
	if err != nil {
		return api.ExpressionEmission{}, api.TypeEmission{}, nil, err
	}
	fields := make([]*types.Var, structure.NumFields())
	for index := range fields {
		fields[index] = structure.Field(index)
	}
	offsets := context.TypesSizes().Offsetsof(fields)
	selected := make([]api.ExpressionEmission, len(fields))
	for index, field := range fields {
		name, err := structconstruction.FieldName(context.Names(), field, index)
		if err != nil {
			return api.ExpressionEmission{}, api.TypeEmission{}, nil, err
		}
		child, _, err := bindingLayout(context, children, source, field.Type(), physical)
		if err != nil {
			return api.ExpressionEmission{}, api.TypeEmission{}, nil, err
		}
		value, err := selectedFieldLayout(context, rootType, name, offsets[index], context.TypesSizes().Alignof(field.Type()), child)
		if err != nil {
			return api.ExpressionEmission{}, api.TypeEmission{}, nil, err
		}
		if capture {
			value, before, err = captureLayout(context, value, before)
			if err != nil {
				return api.ExpressionEmission{}, api.TypeEmission{}, nil, err
			}
		}
		selected[index] = value
		arguments = append(arguments, value)
	}
	result, err := pointermarker.Operation(context, tsoniccore.SymbolMemoryLayout, []api.TypeEmission{rootType}, arguments)
	if err != nil {
		return api.ExpressionEmission{}, api.TypeEmission{}, nil, err
	}
	if capture {
		result, before, err = captureLayout(context, result, before)
		if err != nil {
			return api.ExpressionEmission{}, api.TypeEmission{}, nil, err
		}
	}
	result, err = api.NewExpressionEmission(append(before, result.Before()...), result.Value(), result.Requests())
	return result, represented, selected, err
}

func bindingLayout(context api.Context, children api.ChildEmitter, source ast.Node, sourceType types.Type, physical bool) (api.ExpressionEmission, api.TypeEmission, error) {
	if physical {
		return Layout(context, children, source, sourceType)
	}
	if structure, ok := sourceType.Underlying().(*types.Struct); ok {
		layout, represented, _, err := recordBindingLayout(context, children, source, sourceType, structure, false, false)
		return layout, represented, err
	}
	represented, err := bindingStorageType(context, source, sourceType, false)
	if err != nil {
		return api.ExpressionEmission{}, api.TypeEmission{}, err
	}
	arguments, err := layoutDimensions(context, source, sourceType)
	if err != nil {
		return api.ExpressionEmission{}, api.TypeEmission{}, err
	}
	result, err := pointermarker.Operation(context, tsoniccore.SymbolMemoryLayout, []api.TypeEmission{represented}, arguments)
	return result, represented, err
}

func bindingStorageType(context api.Context, source ast.Node, sourceType types.Type, physical bool) (api.TypeEmission, error) {
	if physical {
		return context.Values().MemoryStorageType(context, source, sourceType)
	}
	return context.Values().StorageType(context, source, sourceType)
}

func layoutDimensions(context api.Context, source ast.Node, sourceType types.Type) ([]api.ExpressionEmission, error) {
	abi, err := DataLayout(context)
	if err != nil {
		return nil, err
	}
	size := context.TypesSizes().Sizeof(sourceType)
	alignment := context.TypesSizes().Alignof(sourceType)
	stride := context.TypesSizes().Sizeof(types.NewArray(sourceType, 2)) - size
	if size < 0 || alignment <= 0 || stride < size || size > 9007199254740991 || stride > 9007199254740991 {
		return nil, api.Unsupported(context, api.CategoryExpression, source)
	}
	arguments := []api.ExpressionEmission{abi}
	for _, value := range []int64{size, alignment, stride} {
		arguments = append(arguments, api.DirectExpression(context.Factory().NumericLiteral(strconv.FormatInt(value, 10), tsgo.TokenFlagsNone)))
	}
	return arguments, nil
}

func captureLayout(context api.Context, value api.ExpressionEmission, before []tsgo.Statement) (api.ExpressionEmission, []tsgo.Statement, error) {
	name, err := context.Names().Temporary(api.TemporaryConversionOperand)
	if err != nil {
		return api.ExpressionEmission{}, nil, err
	}
	factory := context.Factory()
	before = append(before, value.Before()...)
	before = append(before, factory.VariableStatement(nil, factory.VariableDeclarationList([]tsgo.VariableDeclaration{
		factory.VariableDeclaration(factory.Identifier(name), nil, nil, value.Value()),
	}, tsgo.NodeFlagsConst)))
	return api.DirectExpression(factory.Identifier(name), value.Requests()...), before, nil
}
