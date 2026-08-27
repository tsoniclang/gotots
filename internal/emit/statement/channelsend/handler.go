package channelsend

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	channelmodel "github.com/tsoniclang/gotots/internal/emit/concurrency/channel"
	runtimechannel "github.com/tsoniclang/gotots/internal/emit/runtime/channel"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SendStmt,
) (api.StatementEmission, error) {
	if source == nil || source.Chan == nil || source.Value == nil {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	channelType := context.TypesInfo().TypeOf(source.Chan)
	model, ok := channelmodel.Resolve(channelType)
	valueType := context.TypesInfo().TypeOf(source.Value)
	if !ok ||
		model.Direction() == types.RecvOnly ||
		valueType == nil ||
		!types.AssignableTo(valueType, model.Element()) {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	channel, err := children.Expression(
		context.
			WithRole(api.RoleChannelOperand).
			WithExpectedType(channelType),
		source.Chan,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	channel, err = model.Project(context, channel)
	if err != nil {
		return api.StatementEmission{}, err
	}
	value, err := children.Expression(
		context.
			WithRole(api.RoleChannelElement).
			WithExpectedType(model.Element()),
		source.Value,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	value, err = context.Values().Transfer(
		context.WithRole(api.RoleChannelElement),
		source.Value,
		valueType,
		model.Element(),
		api.ValueTransferRepresentation,
		value,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	target, err := channelmodel.BlockingCall(
		context,
		source,
		runtimechannel.MemberSend,
		channel,
		value,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	statements := target.Before()
	statements = append(
		statements,
		context.Factory().ExpressionStatement(target.Value()),
	)
	return api.NewStatementEmission(statements, target.Requests())
}
