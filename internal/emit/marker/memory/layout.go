package memory

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/goabi"
	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	pointermarker "github.com/tsoniclang/gotots/internal/emit/marker/pointer"
	runtimecomplex "github.com/tsoniclang/gotots/internal/emit/runtime/complex"
	arrayvalue "github.com/tsoniclang/gotots/internal/emit/value/array"
	complexvalue "github.com/tsoniclang/gotots/internal/emit/value/complex"
	descriptorvalue "github.com/tsoniclang/gotots/internal/emit/value/memorydescriptor"
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
	if model, ok := descriptorvalue.Resolve(context, pointee); ok {
		return DescriptorLayout(context, model)
	}
	supported, err := SupportsLayout(context, pointee)
	if err != nil {
		return api.ExpressionEmission{}, api.TypeEmission{}, err
	}
	if !supported {
		return api.ExpressionEmission{}, api.TypeEmission{}, api.Unsupported(context, api.CategoryExpression, source)
	}
	if structure, ok := pointee.Underlying().(*types.Struct); ok {
		layout, represented, _, err := recordBindingLayout(context, children, source, pointee, structure, true, false)
		return layout, represented, err
	}
	represented, err := context.Values().MemoryStorageType(context.WithRole(api.RoleStorageType), source, pointee)
	if err != nil {
		return api.ExpressionEmission{}, api.TypeEmission{}, err
	}
	arguments, err := layoutDimensions(context, source, pointee)
	if err != nil {
		return api.ExpressionEmission{}, api.TypeEmission{}, err
	}
	layoutType := represented
	if array, ok := pointee.Underlying().(*types.Array); ok {
		child, element, childErr := Layout(context, children, source, array.Elem())
		if childErr != nil {
			return api.ExpressionEmission{}, api.TypeEmission{}, childErr
		}
		extent := arrayvalue.ExtentLiteral(context.Factory(), array)
		result, callErr := pointermarker.Operation(context, tsoniccore.SymbolMemoryArrayLayout,
			[]api.TypeEmission{element, api.DirectType(context.Factory().LiteralTypeNode(extent))},
			append(arguments, child, api.DirectExpression(extent)))
		return result, represented, callErr
	}
	if carrier, ok := complexvalue.Describe(pointee.Underlying()); ok {
		component := carrier.ComponentType()
		for index, name := range []string{runtimecomplex.RealMember, runtimecomplex.ImagMember} {
			selected, fieldErr := fieldLayout(context, children, source, represented, name, component, int64(index)*context.TypesSizes().Sizeof(component))
			if fieldErr != nil {
				return api.ExpressionEmission{}, api.TypeEmission{}, fieldErr
			}
			arguments = append(arguments, selected)
		}
	}
	result, err := pointermarker.Operation(context, tsoniccore.SymbolMemoryLayout, []api.TypeEmission{layoutType}, arguments)
	return result, represented, err
}
