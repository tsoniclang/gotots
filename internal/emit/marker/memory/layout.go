package memory

import (
	"go/ast"
	"go/types"
	"strconv"

	"github.com/tsoniclang/gotots/internal/contracts/goabi"
	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	pointermarker "github.com/tsoniclang/gotots/internal/emit/marker/pointer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type sourceLayoutNames interface {
	SourceDataLayout(goabi.Layout) (api.NameReference, error)
}

func DataLayout(context api.Context) (api.ExpressionEmission, error) {
	layout, err := goabi.Select(context.MemoryByteOrder(), context.TypesSizes().Sizeof(types.Typ[types.UnsafePointer]))
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	names, ok := context.Names().(sourceLayoutNames)
	if !ok {
		return api.ExpressionEmission{}, &api.InvariantError{Role: context.Role(), Reason: "source ABI has no canonical name owner"}
	}
	reference, err := names.SourceDataLayout(layout)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.DirectExpression(reference.Expression(context.Factory()), reference.Requests()...), nil
}

func Layout(context api.Context, children api.ChildEmitter, source ast.Node, pointee types.Type) (api.ExpressionEmission, api.TypeEmission, error) {
	if pointee == nil || api.ContainsGenericTypeParameter(pointee) || !physicalLayoutRepresentable(pointee) {
		return api.ExpressionEmission{}, api.TypeEmission{}, api.Unsupported(context, api.CategoryExpression, source)
	}
	projected, err := context.Values().RequiresStorageProjection(context, pointee)
	if err != nil {
		return api.ExpressionEmission{}, api.TypeEmission{}, err
	}
	if _, structure := pointee.Underlying().(*types.Struct); structure && !projected {
		return api.ExpressionEmission{}, api.TypeEmission{}, api.Unsupported(context, api.CategoryExpression, source)
	}
	var represented api.TypeEmission
	if projected {
		represented, err = context.Values().StorageType(context.WithRole(api.RoleStorageType), source, pointee)
	} else {
		represented, err = children.RepresentedType(context.WithRole(api.RoleResultType), nil, pointee)
	}
	if err != nil {
		return api.ExpressionEmission{}, api.TypeEmission{}, err
	}
	abi, err := DataLayout(context)
	if err != nil {
		return api.ExpressionEmission{}, api.TypeEmission{}, err
	}
	size := context.TypesSizes().Sizeof(pointee)
	alignment := context.TypesSizes().Alignof(pointee)
	stride := context.TypesSizes().Sizeof(types.NewArray(pointee, 2)) - size
	if size < 0 || alignment <= 0 || stride < size || size > 9007199254740991 || stride > 9007199254740991 {
		return api.ExpressionEmission{}, api.TypeEmission{}, api.Unsupported(context, api.CategoryExpression, source)
	}
	number := func(value int64) api.ExpressionEmission {
		return api.DirectExpression(context.Factory().NumericLiteral(strconv.FormatInt(value, 10), tsgo.TokenFlagsNone))
	}
	arguments := []api.ExpressionEmission{abi, number(size), number(alignment), number(stride)}
	if structure, ok := pointee.Underlying().(*types.Struct); ok {
		fields := make([]*types.Var, structure.NumFields())
		for index := range fields {
			fields[index] = structure.Field(index)
		}
		offsets := context.TypesSizes().Offsetsof(fields)
		for index, field := range fields {
			name, nameErr := context.Names().Member(field)
			if nameErr != nil {
				return api.ExpressionEmission{}, api.TypeEmission{}, nameErr
			}
			parameter, nameErr := context.Names().Temporary(api.TemporaryConversionOperand)
			if nameErr != nil {
				return api.ExpressionEmission{}, api.TypeEmission{}, nameErr
			}
			selector := context.Factory().ArrowFunction(nil, nil, []tsgo.ParameterDeclaration{
				context.Factory().ParameterDeclaration(nil, nil, context.Factory().Identifier(parameter), nil, represented.Value(), nil),
			}, nil, context.Factory().EqualsGreaterThanToken(), context.Factory().PropertyAccessExpression(
				context.Factory().Identifier(parameter), nil, context.Factory().Identifier(name), tsgo.NodeFlagsNone))
			selected, fieldErr := pointermarker.Operation(context, tsoniccore.SymbolMemoryField, nil, []api.ExpressionEmission{
				api.DirectExpression(selector, represented.Requests()...), number(offsets[index]), number(context.TypesSizes().Alignof(field.Type())),
			})
			if fieldErr != nil {
				return api.ExpressionEmission{}, api.TypeEmission{}, fieldErr
			}
			arguments = append(arguments, selected)
		}
	}
	result, err := pointermarker.Operation(context, tsoniccore.SymbolMemoryLayout, []api.TypeEmission{represented}, arguments)
	return result, represented, err
}
