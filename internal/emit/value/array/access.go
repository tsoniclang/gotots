package array

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	expressionoperands "github.com/tsoniclang/gotots/internal/emit/expression/operands"
	arraymember "github.com/tsoniclang/gotots/internal/emit/runtime/array/member"
	integeroperand "github.com/tsoniclang/gotots/internal/emit/value/integer/operand"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (a RuntimeArray) EmitIndex(
	context api.Context,
	children api.ChildEmitter,
	source *ast.IndexExpr,
) (api.ExpressionEmission, error) {
	if !types.Identical(context.TypesInfo().TypeOf(source), a.ElementType()) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	receiver, err := children.Expression(
		context.
			WithRole(api.RoleArrayReceiver).
			WithExpectedType(a.sourceType),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	index, err := emitIndex(context, children, source.Index)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return a.ApplyIndex(context, receiver, index)
}

func (a RuntimeArray) ApplyIndex(
	context api.Context,
	receiver api.ExpressionEmission,
	index api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if a.source == nil {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "runtime array index has no source model",
		}
	}
	ordered, err := expressionoperands.Preserve(
		context,
		api.TemporaryArrayReceiver,
		expressionoperands.Present(receiver),
		expressionoperands.Present(index),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	values := ordered.Values()
	storedReceiver, err := a.storage(
		context.WithRole(api.RoleArrayReceiver),
		api.DirectExpression(values[0]),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	stored, err := api.NewExpressionEmission(
		append(ordered.Before(), storedReceiver.Before()...),
		callMember(
			context,
			storedReceiver.Value(),
			arraymember.Get,
			values[1],
		),
		api.CombineRequests(ordered.Requests(), storedReceiver.Requests()),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return context.ContainerStorage().FromContainerStorage(
		context.WithRole(api.RoleArrayElement),
		nil,
		a.ElementType(),
		stored,
	)
}

func (a RuntimeArray) EmitStoreTarget(
	context api.Context,
	children api.ChildEmitter,
	source *ast.IndexExpr,
) (api.StoreTargetEmission, error) {
	if !types.Identical(
		context.TypesInfo().TypeOf(source.X),
		a.sourceType,
	) {
		return api.StoreTargetEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	receiver, err := children.Expression(
		context.
			WithRole(api.RoleArrayReceiver).
			WithExpectedType(a.sourceType),
		source.X,
	)
	if err != nil {
		return api.StoreTargetEmission{}, err
	}
	index, err := emitIndex(context, children, source.Index)
	if err != nil {
		return api.StoreTargetEmission{}, err
	}
	targetReceiver, err := a.storage(
		context.WithRole(api.RoleArrayReceiver),
		receiver,
	)
	if err != nil {
		return api.StoreTargetEmission{}, err
	}
	target, err := api.NewContainerStorageAccessorStoreTargetEmission(
		targetReceiver,
		arraymember.Get.Name(),
		arraymember.Set.Name(),
		[]api.ExpressionEmission{index},
		a.ElementType(),
	)
	if err != nil {
		return api.StoreTargetEmission{}, err
	}
	return target, nil
}

func (a RuntimeArray) EmitLength(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
) (api.ExpressionEmission, error) {
	if len(source.Args) != 1 ||
		!types.Identical(
			context.TypesInfo().TypeOf(source.Args[0]),
			a.sourceType,
		) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	value, err := children.Expression(
		context.
			WithRole(api.RoleBuiltinArgument).
			WithExpectedType(a.sourceType),
		source.Args[0],
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return a.Measure(context, value)
}

func (a RuntimeArray) Measure(
	context api.Context,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	stored, err := a.storage(
		context.WithRole(api.RoleBuiltinArgument),
		value,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	target := tsgo.Expression(memberProperty(
		context,
		stored.Value(),
		arraymember.Length,
	))
	if context.ScalarABI().UsesBigInt(types.Typ[types.Int]) {
		target = context.Factory().CallExpression(
			api.TargetIntrinsicBigInt.Expression(context.Factory()),
			nil,
			nil,
			[]tsgo.Expression{target},
			tsgo.NodeFlagsNone,
		)
	}
	return api.NewExpressionEmission(
		stored.Before(),
		target,
		stored.Requests(),
	)
}

func (a RuntimeArray) RangeElement(
	context api.Context,
	source ast.Node,
	receiver tsgo.Expression,
	index tsgo.Expression,
) (api.ExpressionEmission, error) {
	stored, err := a.storage(
		context.WithRole(api.RoleArrayReceiver),
		api.DirectExpression(receiver),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	loaded, err := api.NewExpressionEmission(
		stored.Before(),
		callMember(
			context,
			stored.Value(),
			arraymember.Get,
			index,
		),
		stored.Requests(),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return context.ContainerStorage().FromContainerStorage(
		context.WithRole(api.RoleArrayElement),
		source,
		a.ElementType(),
		loaded,
	)
}

func emitIndex(
	context api.Context,
	children api.ChildEmitter,
	source ast.Expr,
) (api.ExpressionEmission, error) {
	return integeroperand.Emit(
		context.WithRole(api.RoleArrayIndex),
		children,
		source,
	)
}

func variable(
	context api.Context,
	name string,
	value tsgo.Expression,
) tsgo.VariableStatement {
	return context.Factory().VariableStatement(
		nil,
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				context.Factory().VariableDeclaration(
					context.Factory().Identifier(name),
					nil,
					nil,
					value,
				),
			},
			tsgo.NodeFlagsConst,
		),
	)
}
