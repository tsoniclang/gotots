package pointer

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	pointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
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
	sourceRepresentation, err := pointertype.Observe(
		context,
		sourcePointer,
		api.PointerRepresentationDemandStableLocation,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	targetRepresentation, err := pointertype.Observe(
		context,
		targetPointer,
		api.PointerRepresentationDemandStableLocation,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	if sourceRepresentation.Representation().DirectClass() &&
		targetRepresentation.Representation().DirectClass() {
		target, err := convertDirectClassPointer(
			context,
			source,
			sourcePointer.Elem(),
			targetPointer.Elem(),
			value,
			api.CombineRequests(
				sourceRepresentation.Requests(),
				targetRepresentation.Requests(),
			),
		)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
		if targetDefined.Type() != nil {
			target, err = targetDefined.Wrap(context, target)
		}
		return target, true, err
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
	sourceStorage, err := context.Values().StorageType(
		context.WithRole(api.RoleStorageType),
		source,
		sourcePointer.Elem(),
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	targetStorage, err := context.Values().StorageType(
		context.WithRole(api.RoleStorageType),
		source,
		targetPointer.Elem(),
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	runtime, err := context.Names().Runtime(
		api.RuntimePointer,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	target, err := api.NewExpressionEmission(
		value.Before(),
		context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				context.Factory().Identifier(runtime.Name()),
				nil,
				context.Factory().Identifier(pointerruntime.ViewName),
				tsgo.NodeFlagsNone,
			),
			nil,
			[]tsgo.TypeNode{
				sourceLogical.Value(),
				targetLogical.Value(),
				targetStorage.Value(),
			},
			[]tsgo.Expression{value.Value()},
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			value.Requests(),
			sourceLogical.Requests(),
			sourceStorage.Requests(),
			targetLogical.Requests(),
			targetStorage.Requests(),
			runtime.Requests(),
			sourceRepresentation.Requests(),
			targetRepresentation.Requests(),
		),
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	if targetDefined.Type() != nil {
		target, err = targetDefined.Wrap(context, target)
	}
	return target, true, err
}

func convertDirectClassPointer(
	context api.Context,
	source ast.Node,
	sourceElement types.Type,
	targetElement types.Type,
	value api.ExpressionEmission,
	requests []api.RootRequest,
) (api.ExpressionEmission, error) {
	before := value.Before()
	pointer := value.Value()
	if pointer.Kind() != tsgo.SyntaxKindIdentifier {
		name, err := context.Names().Temporary(api.TemporaryConversionOperand)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		before = append(before, context.Factory().VariableStatement(
			nil,
			context.Factory().VariableDeclarationList(
				[]tsgo.VariableDeclaration{context.Factory().VariableDeclaration(
					context.Factory().Identifier(name),
					nil,
					nil,
					pointer,
				)},
				tsgo.NodeFlagsConst,
			),
		))
		pointer = context.Factory().Identifier(name)
	}
	stored, err := context.Values().ToStorage(
		context.WithRole(api.RoleConversionOperand),
		source,
		sourceElement,
		api.DirectExpression(pointer),
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
	if len(converted.Before()) != 0 {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "direct pointer conversion produced deferred storage work",
		}
	}
	undefined := context.Factory().Identifier("undefined")
	return api.NewExpressionEmission(
		before,
		context.Factory().ConditionalExpression(
			context.Factory().BinaryExpression(
				nil,
				pointer,
				nil,
				context.Factory().BinaryOperatorToken(
					tsgo.BinaryOperatorEqualsEqualsEqualsToken,
				),
				undefined,
			),
			context.Factory().QuestionToken(),
			undefined,
			context.Factory().ColonToken(),
			converted.Value(),
		),
		api.CombineRequests(
			value.Requests(),
			converted.Requests(),
			requests,
		),
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
