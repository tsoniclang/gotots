package selectstatement

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	channelmodel "github.com/tsoniclang/gotots/internal/emit/concurrency/channel"
	runtimechannel "github.com/tsoniclang/gotots/internal/emit/runtime/channel"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func prepareClauses(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SelectStmt,
) (prepared, []api.RootRequest, error) {
	result := prepared{
		clauses: make([]clause, 0, len(source.Body.List)),
	}
	var requests []api.RootRequest
	for _, statement := range source.Body.List {
		sourceClause, ok := statement.(*ast.CommClause)
		if !ok {
			return prepared{}, nil, api.Unsupported(
				context.WithRole(api.RoleSelectClause),
				api.CategoryStatement,
				statement,
			)
		}
		if sourceClause.Comm == nil {
			if result.hasDefault {
				return prepared{}, nil, api.Unsupported(
					context.WithRole(api.RoleSelectClause),
					api.CategoryStatement,
					sourceClause,
				)
			}
			result.hasDefault = true
			result.clauses = append(result.clauses, clause{
				source:    sourceClause,
				selection: -1,
			})
			continue
		}
		var target clause
		var before []tsgo.Statement
		var selectedRequests []api.RootRequest
		var err error
		switch communication := sourceClause.Comm.(type) {
		case *ast.SendStmt:
			target, before, selectedRequests, err = prepareSend(
				context,
				children,
				sourceClause,
				communication,
				len(result.alternatives),
			)
		case *ast.ExprStmt:
			receive, ok := receiveExpression(communication.X)
			if !ok {
				return prepared{}, nil, api.Unsupported(
					context.WithRole(api.RoleSelectClause),
					api.CategoryStatement,
					communication,
				)
			}
			target, before, selectedRequests, err = prepareReceive(
				context,
				children,
				sourceClause,
				receive,
				nil,
				len(result.alternatives),
			)
		case *ast.AssignStmt:
			if len(communication.Rhs) != 1 ||
				(communication.Tok != token.DEFINE &&
					communication.Tok != token.ASSIGN) {
				return prepared{}, nil, api.Unsupported(
					context.WithRole(api.RoleSelectClause),
					api.CategoryStatement,
					communication,
				)
			}
			receive, ok := receiveExpression(communication.Rhs[0])
			if !ok {
				return prepared{}, nil, api.Unsupported(
					context.WithRole(api.RoleSelectClause),
					api.CategoryStatement,
					communication,
				)
			}
			target, before, selectedRequests, err = prepareReceive(
				context,
				children,
				sourceClause,
				receive,
				communication,
				len(result.alternatives),
			)
		default:
			return prepared{}, nil, api.Unsupported(
				context.WithRole(api.RoleSelectClause),
				api.CategoryStatement,
				sourceClause.Comm,
			)
		}
		if err != nil {
			return prepared{}, nil, err
		}
		result.before = append(result.before, before...)
		requests = append(requests, selectedRequests...)
		result.alternatives = append(
			result.alternatives,
			target.alternative,
		)
		result.clauses = append(result.clauses, target)
	}
	return result, requests, nil
}

func prepareSend(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CommClause,
	send *ast.SendStmt,
	selection int,
) (clause, []tsgo.Statement, []api.RootRequest, error) {
	channelType := context.TypesInfo().TypeOf(send.Chan)
	model, ok := channelmodel.Resolve(channelType)
	valueType := context.TypesInfo().TypeOf(send.Value)
	if !ok ||
		model.Direction() == types.RecvOnly ||
		valueType == nil ||
		!types.AssignableTo(valueType, model.Element()) {
		return clause{}, nil, nil, api.Unsupported(
			context.WithRole(api.RoleSelectClause),
			api.CategoryStatement,
			send,
		)
	}
	channel, err := children.Expression(
		context.
			WithRole(api.RoleChannelOperand).
			WithExpectedType(channelType),
		send.Chan,
	)
	if err != nil {
		return clause{}, nil, nil, err
	}
	channel, err = model.Project(context, channel)
	if err != nil {
		return clause{}, nil, nil, err
	}
	value, err := children.Expression(
		context.
			WithRole(api.RoleChannelElement).
			WithExpectedType(model.Element()),
		send.Value,
	)
	if err != nil {
		return clause{}, nil, nil, err
	}
	value, err = context.Values().Transfer(
		context.WithRole(api.RoleChannelElement),
		send.Value,
		valueType,
		model.Element(),
		api.ValueTransferRepresentation,
		value,
	)
	if err != nil {
		return clause{}, nil, nil, err
	}
	alternative, err := channelmodel.StaticCall(
		context,
		send,
		runtimechannel.MemberSelectSend,
		channel,
		value,
	)
	if err != nil {
		return clause{}, nil, nil, err
	}
	name, err := context.Names().Temporary(api.TemporarySelectCase)
	if err != nil {
		return clause{}, nil, nil, err
	}
	identifier := context.Factory().Identifier(name)
	before := alternative.Before()
	before = append(before, constant(context, identifier, alternative.Value()))
	return clause{
		source:      source,
		selection:   selection,
		alternative: identifier,
		channel:     model,
	}, before, alternative.Requests(), nil
}

func prepareReceive(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CommClause,
	receive *ast.UnaryExpr,
	assignment *ast.AssignStmt,
	selection int,
) (clause, []tsgo.Statement, []api.RootRequest, error) {
	channelType := context.TypesInfo().TypeOf(receive.X)
	model, ok := channelmodel.Resolve(channelType)
	if !ok || model.Direction() == types.SendOnly {
		return clause{}, nil, nil, api.Unsupported(
			context.WithRole(api.RoleSelectClause),
			api.CategoryExpression,
			receive,
		)
	}
	channel, err := children.Expression(
		context.
			WithRole(api.RoleChannelOperand).
			WithExpectedType(channelType),
		receive.X,
	)
	if err != nil {
		return clause{}, nil, nil, err
	}
	channel, err = model.Project(context, channel)
	if err != nil {
		return clause{}, nil, nil, err
	}
	elementType, err := children.RepresentedType(
		context.WithRole(api.RoleChannelElementType),
		receive,
		model.Element(),
	)
	if err != nil {
		return clause{}, nil, nil, err
	}
	resultName, err := context.Names().Temporary(
		api.TemporaryChannelResult,
	)
	if err != nil {
		return clause{}, nil, nil, err
	}
	result := context.Factory().Identifier(resultName)
	callback := receiveCallback(context, result, elementType.Value())
	alternative, err := channelmodel.StaticCall(
		context,
		receive,
		runtimechannel.MemberSelectReceive,
		channel,
		api.DirectExpression(callback),
	)
	if err != nil {
		return clause{}, nil, nil, err
	}
	caseName, err := context.Names().Temporary(api.TemporarySelectCase)
	if err != nil {
		return clause{}, nil, nil, err
	}
	caseIdentifier := context.Factory().Identifier(caseName)
	before := alternative.Before()
	before = append(
		before,
		receiveResult(context, result, elementType.Value()),
		constant(context, caseIdentifier, alternative.Value()),
	)
	return clause{
			source:        source,
			selection:     selection,
			alternative:   caseIdentifier,
			receive:       receive,
			assignment:    assignment,
			receiveResult: result,
			channel:       model,
		},
		before,
		api.CombineRequests(
			elementType.Requests(),
			alternative.Requests(),
		),
		nil
}

func receiveExpression(source ast.Expr) (*ast.UnaryExpr, bool) {
	receive, ok := source.(*ast.UnaryExpr)
	return receive, ok && receive.Op == token.ARROW && receive.X != nil
}

func receiveCallback(
	context api.Context,
	result tsgo.Identifier,
	elementType tsgo.TypeNode,
) tsgo.ArrowFunction {
	value := context.Factory().Identifier("value")
	ok := context.Factory().Identifier("ok")
	return context.Factory().ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			context.Factory().ParameterDeclaration(
				nil,
				nil,
				value,
				nil,
				elementType,
				nil,
			),
			context.Factory().ParameterDeclaration(
				nil,
				nil,
				ok,
				nil,
				context.Factory().KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindBooleanKeyword,
				),
				nil,
			),
		},
		context.Factory().KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindVoidKeyword,
		),
		context.Factory().EqualsGreaterThanToken(),
		context.Factory().Block(
			[]tsgo.Statement{
				context.Factory().ExpressionStatement(
					context.Factory().BinaryExpression(
						nil,
						result,
						nil,
						context.Factory().BinaryOperatorToken(
							tsgo.BinaryOperatorEqualsToken,
						),
						context.Factory().ArrayLiteralExpression(
							[]tsgo.Expression{value, ok},
							false,
						),
					),
				),
			},
			true,
		),
	)
}

func receiveResult(
	context api.Context,
	name tsgo.Identifier,
	elementType tsgo.TypeNode,
) tsgo.VariableStatement {
	targetType := context.Factory().UnionTypeNode([]tsgo.TypeNode{
		context.Factory().TupleTypeNode([]tsgo.TypeNode{
			elementType,
			context.Factory().KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindBooleanKeyword,
			),
		}),
		context.Factory().KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
		),
	})
	return context.Factory().VariableStatement(
		nil,
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				context.Factory().VariableDeclaration(
					name,
					nil,
					targetType,
					context.Factory().Identifier("undefined"),
				),
			},
			tsgo.NodeFlagsLet,
		),
	)
}

func constant(
	context api.Context,
	name tsgo.Identifier,
	value tsgo.Expression,
) tsgo.VariableStatement {
	return context.Factory().VariableStatement(
		nil,
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				context.Factory().VariableDeclaration(
					name,
					nil,
					nil,
					value,
				),
			},
			tsgo.NodeFlagsConst,
		),
	)
}
