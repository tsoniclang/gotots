package unsafepointer

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	integerbinary "github.com/tsoniclang/gotots/internal/emit/expression/binary/integer"
	expressionoperands "github.com/tsoniclang/gotots/internal/emit/expression/operands"
	unsafepointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/unsafepointer"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	integervalue "github.com/tsoniclang/gotots/internal/emit/value/integer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func EmitClosedOffset(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	sourceType types.Type,
	targetType types.Type,
) (api.ExpressionEmission, bool, error) {
	binary, pointerSource, pointerType, ok := closedOffset(
		context,
		source,
		sourceType,
		targetType,
	)
	if !ok {
		return api.ExpressionEmission{}, false, nil
	}
	pointer, err := children.Expression(
		context.
			WithRole(api.RoleBinaryLeft).
			WithExpectedType(pointerType),
		pointerSource,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	if model, defined := definedtype.ResolveBasic(pointerType); defined {
		pointer, err = model.Project(context, pointer)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
	}
	pointer, before, err := capturePointer(context, pointer)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	delta, err := children.Expression(
		context.
			WithRole(api.RoleBinaryRight).
			WithExpectedType(types.Typ[types.Uintptr]),
		binary.Y,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	if model, defined := definedtype.ResolveBasic(
		context.TypesInfo().TypeOf(binary.Y),
	); defined {
		delta, err = model.Project(context, delta)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
	}
	reference, err := context.Names().Runtime(
		api.RuntimeUnsafePointer,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	address, err := integerBoundary(
		context,
		reference,
		unsafepointerruntime.ToIntegerName,
		pointer,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	pair, err := expressionoperands.PreservePair(
		context,
		address,
		delta,
		api.TemporaryBinaryOperand,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	carrier, ok := integervalue.Describe(
		context.TypesSizes(),
		types.Typ[types.Uintptr],
	)
	if !ok {
		return api.ExpressionEmission{}, true, &api.InvariantError{
			Role:   context.Role(),
			Reason: "uintptr carrier is unavailable for unsafe offset fusion",
		}
	}
	address, handled, err := integerbinary.Apply(
		context,
		token.ADD,
		carrier,
		carrier,
		pair.Left(),
		pair.Right(),
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	if !handled {
		return api.ExpressionEmission{}, true, &api.InvariantError{
			Role:   context.Role(),
			Reason: "uintptr addition is unavailable for unsafe offset fusion",
		}
	}
	address, err = expressionoperands.Finish(pair, address)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	zero, err := integervalue.Literal(
		context,
		types.Typ[types.Uintptr],
		"0",
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	before = append(before, address.Before()...)
	target, err := api.NewExpressionEmission(
		before,
		call(
			context,
			reference.Name(),
			unsafepointerruntime.FromRelativeName,
			nil,
			pointer.Value(),
			address.Value(),
			zero,
		),
		api.CombineRequests(
			pointer.Requests(),
			address.Requests(),
			reference.Requests(),
		),
	)
	if err == nil {
		target, err = wrapDefinedUnsafe(context, targetType, target)
	}
	return target, true, err
}

func closedOffset(
	context api.Context,
	source *ast.CallExpr,
	sourceType types.Type,
	targetType types.Type,
) (*ast.BinaryExpr, ast.Expr, types.Type, bool) {
	if source == nil ||
		len(source.Args) != 1 ||
		source.Ellipsis != token.NoPos ||
		!types.Identical(sourceType, types.Typ[types.Uintptr]) ||
		!basictype.SupportsUnsafePointer(targetType) {
		return nil, nil, nil, false
	}
	binary, ok := ast.Unparen(source.Args[0]).(*ast.BinaryExpr)
	if !ok ||
		binary.Op != token.ADD ||
		!types.Identical(
			context.TypesInfo().TypeOf(binary),
			types.Typ[types.Uintptr],
		) ||
		!types.AssignableTo(
			context.TypesInfo().TypeOf(binary.Y),
			types.Typ[types.Uintptr],
		) {
		return nil, nil, nil, false
	}
	conversion, ok := ast.Unparen(binary.X).(*ast.CallExpr)
	if !ok || len(conversion.Args) != 1 || conversion.Ellipsis != token.NoPos {
		return nil, nil, nil, false
	}
	callee, ok := context.TypesInfo().TypeAndValue(conversion.Fun)
	if !ok ||
		!callee.IsType() ||
		!types.Identical(
			context.TypesInfo().TypeOf(conversion),
			types.Typ[types.Uintptr],
		) {
		return nil, nil, nil, false
	}
	pointerSource := conversion.Args[0]
	pointerType := context.TypesInfo().TypeOf(pointerSource)
	if !basictype.SupportsUnsafePointer(pointerType) {
		return nil, nil, nil, false
	}
	return binary, pointerSource, pointerType, true
}

func capturePointer(
	context api.Context,
	pointer api.ExpressionEmission,
) (api.ExpressionEmission, []tsgo.Statement, error) {
	name, err := context.Names().Temporary(api.TemporaryConversionOperand)
	if err != nil {
		return api.ExpressionEmission{}, nil, err
	}
	before := pointer.Before()
	before = append(before, context.Factory().VariableStatement(
		nil,
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{context.Factory().VariableDeclaration(
				context.Factory().Identifier(name),
				nil,
				nil,
				pointer.Value(),
			)},
			tsgo.NodeFlagsConst,
		),
	))
	return api.DirectExpression(
		context.Factory().Identifier(name),
		pointer.Requests()...,
	), before, nil
}
