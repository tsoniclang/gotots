package channelreceive

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	channelmodel "github.com/tsoniclang/gotots/internal/emit/concurrency/channel"
	runtimechannel "github.com/tsoniclang/gotots/internal/emit/runtime/channel"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.UnaryExpr,
) (api.ExpressionEmission, error) {
	if source == nil || source.Op != token.ARROW || source.X == nil {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	channelType := context.TypesInfo().TypeOf(source.X)
	model, ok := channelmodel.Resolve(channelType)
	if !ok ||
		model.Direction() == types.SendOnly ||
		!validResultContext(context, source, model.Element()) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	channel, err := children.Expression(
		context.
			WithRole(api.RoleChannelOperand).
			WithExpectedType(channelType),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	channel, err = model.Project(context, channel)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	result, err := channelmodel.BlockingCall(
		context,
		source,
		runtimechannel.MemberReceive,
		channel,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if context.ExpectedResults() != nil {
		return result, nil
	}
	return api.NewExpressionEmission(
		result.Before(),
		context.Factory().ElementAccessExpression(
			result.Value(),
			nil,
			context.Factory().NumericLiteral("0", tsgo.TokenFlagsNone),
			tsgo.NodeFlagsNone,
		),
		result.Requests(),
	)
}

func validResultContext(
	context api.Context,
	source *ast.UnaryExpr,
	elementType types.Type,
) bool {
	if elementType == nil {
		return false
	}
	if expected := context.ExpectedResults(); expected != nil {
		actual, ok := context.TypesInfo().TypeOf(source).(*types.Tuple)
		return ok &&
			actual.Len() == 2 &&
			types.Identical(actual, expected) &&
			types.Identical(actual.At(0).Type(), elementType) &&
			types.Identical(actual.At(1).Type(), types.Typ[types.Bool])
	}
	actual := context.TypesInfo().TypeOf(source)
	expected := context.ExpectedType()
	return actual != nil &&
		expected != nil &&
		types.Identical(actual, elementType) &&
		types.AssignableTo(actual, expected)
}
