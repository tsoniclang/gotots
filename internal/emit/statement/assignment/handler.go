package assignment

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.AssignStmt,
) (api.StatementEmission, error) {
	switch source.Tok {
	case token.DEFINE:
		return emitDefinition(context, children, source)
	case token.ASSIGN:
		return emitAssignment(context, children, source)
	default:
		operator, ok := compoundOperator(source.Tok)
		if !ok {
			return api.StatementEmission{},
				api.Unsupported(context, api.CategoryStatement, source)
		}
		return emitCompound(context, children, source, operator)
	}
}

func emitCompound(
	context api.Context,
	children api.ChildEmitter,
	source *ast.AssignStmt,
	operator token.Token,
) (api.StatementEmission, error) {
	if len(source.Lhs) != 1 || len(source.Rhs) != 1 {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	target, err := children.StoreTarget(
		context.WithRole(api.RoleAssignmentTarget),
		source.Lhs[0],
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	if !target.IsAccessor() &&
		!target.IsProperty() &&
		(!target.UsesCanonicalStorage() ||
			!context.Values().RequiresStorageProjection(
				context,
				target.SourceType(),
			)) &&
		source.Tok == token.ADD_ASSIGN &&
		basictype.SupportsInteger(context.TypesSizes(), target.SourceType()) &&
		types.AssignableTo(
			context.TypesInfo().TypeOf(source.Rhs[0]),
			target.SourceType(),
		) {
		value, err := children.Expression(
			context.
				WithRole(api.RoleAssignmentValue).
				WithExpectedType(target.SourceType()),
			source.Rhs[0],
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
		if len(value.Before()) == 0 {
			expression := context.Factory().BinaryExpression(
				nil,
				target.Value(),
				nil,
				context.Factory().BinaryOperatorToken(
					tsgo.BinaryOperatorPlusEqualsToken,
				),
				value.Value(),
			)
			statements := target.Before()
			statements = append(
				statements,
				context.Factory().ExpressionStatement(expression),
			)
			return api.NewStatementEmission(
				statements,
				api.CombineRequests(target.Requests(), value.Requests()),
			)
		}
		return emitCustomCompound(
			context,
			children,
			source,
			operator,
			target,
			&value,
		)
	}
	return emitCustomCompound(
		context,
		children,
		source,
		operator,
		target,
		nil,
	)
}

func emitCustomCompound(
	context api.Context,
	children api.ChildEmitter,
	source *ast.AssignStmt,
	operator token.Token,
	target api.StoreTargetEmission,
	preparedRight *api.ExpressionEmission,
) (api.StatementEmission, error) {
	target, err := target.CaptureLocation(context)
	if err != nil {
		return api.StatementEmission{}, err
	}
	left, err := target.ReadValue(context, source)
	if err != nil {
		return api.StatementEmission{}, err
	}
	rightType := context.TypesInfo().TypeOf(source.Rhs[0])
	expectedRight := rightType
	assignable := rightType != nil &&
		types.AssignableTo(rightType, target.SourceType())
	if assignable {
		expectedRight = target.SourceType()
	}
	if operator != token.SHL &&
		operator != token.SHR &&
		!assignable {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	var right api.ExpressionEmission
	if preparedRight != nil {
		right = *preparedRight
	} else {
		right, err = children.Expression(
			context.
				WithRole(api.RoleAssignmentValue).
				WithExpectedType(expectedRight),
			source.Rhs[0],
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
	}
	if context.EvaluationOrder() == api.EvaluationOrderPreserveGo {
		right, err = captureCompoundRight(context, right)
		if err != nil {
			return api.StatementEmission{}, err
		}
	}
	result, handled, err := context.Values().BinaryUpdate(
		context,
		source,
		source.Rhs[0],
		target.SourceType(),
		expectedRight,
		operator,
		left.Value(),
		right,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	if !handled {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	stored, err := target.StoreValue(
		context.WithRole(api.RoleAssignmentTarget),
		source,
		result,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	statements := append(
		stored.Before(),
		context.Factory().ExpressionStatement(stored.Value()),
	)
	return api.NewStatementEmission(statements, stored.Requests())
}

func captureCompoundRight(
	context api.Context,
	right api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	name, err := context.Names().Temporary(api.TemporaryAssignmentValue)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	before := append(
		right.Before(),
		variableStatement(
			context,
			tsgo.NodeFlagsConst,
			name,
			right.Value(),
		),
	)
	return api.NewExpressionEmission(
		before,
		context.Factory().Identifier(name),
		right.Requests(),
	)
}

func compoundOperator(source token.Token) (token.Token, bool) {
	switch source {
	case token.ADD_ASSIGN:
		return token.ADD, true
	case token.SUB_ASSIGN:
		return token.SUB, true
	case token.MUL_ASSIGN:
		return token.MUL, true
	case token.QUO_ASSIGN:
		return token.QUO, true
	case token.REM_ASSIGN:
		return token.REM, true
	case token.AND_ASSIGN:
		return token.AND, true
	case token.OR_ASSIGN:
		return token.OR, true
	case token.XOR_ASSIGN:
		return token.XOR, true
	case token.SHL_ASSIGN:
		return token.SHL, true
	case token.SHR_ASSIGN:
		return token.SHR, true
	case token.AND_NOT_ASSIGN:
		return token.AND_NOT, true
	default:
		return token.ILLEGAL, false
	}
}

func emitDefinition(
	context api.Context,
	children api.ChildEmitter,
	source *ast.AssignStmt,
) (api.StatementEmission, error) {
	if len(source.Lhs) > 1 {
		return emitParallel(context, children, source)
	}
	declarations, before, requests, err := emitDefinitionList(
		context,
		children,
		source,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	statements := append(
		before,
		context.Factory().VariableStatement(nil, declarations),
	)
	return api.NewStatementEmission(statements, requests)
}

func EmitForInitializer(
	context api.Context,
	children api.ChildEmitter,
	source *ast.AssignStmt,
) (api.ForInitializerEmission, error) {
	if source.Tok != token.DEFINE {
		expression, err := EmitExpression(context, children, source)
		if err != nil {
			return api.ForInitializerEmission{}, err
		}
		if len(expression.Before()) != 0 {
			return api.ForInitializerEmission{},
				api.Unsupported(context, api.CategoryStatement, source)
		}
		return api.ExpressionForInitializer(
			expression.Value(),
			expression.Requests()...,
		)
	}
	declarations, before, requests, err := emitDefinitionList(
		context,
		children,
		source,
	)
	if err != nil {
		return api.ForInitializerEmission{}, err
	}
	if len(before) != 0 {
		return api.ForInitializerEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	return api.DirectForInitializer(declarations, requests...), nil
}

func EmitExpression(
	context api.Context,
	children api.ChildEmitter,
	source *ast.AssignStmt,
) (api.ExpressionEmission, error) {
	target, err := Emit(context, children, source)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	statements := target.Statements()
	if len(statements) != 1 {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	statement, ok := statements[0].(tsgo.ExpressionStatement)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	return api.DirectExpression(
		statement.Expression(),
		target.Requests()...,
	), nil
}

func emitDefinitionList(
	context api.Context,
	children api.ChildEmitter,
	source *ast.AssignStmt,
) (
	tsgo.VariableDeclarationList,
	[]tsgo.Statement,
	[]api.RootRequest,
	error,
) {
	if len(source.Lhs) != 1 || len(source.Rhs) != 1 {
		return nil, nil, nil, api.Unsupported(context, api.CategoryStatement, source)
	}
	name, ok := source.Lhs[0].(*ast.Ident)
	if !ok {
		return nil, nil, nil, api.Unsupported(context, api.CategoryStatement, source)
	}
	object, ok := context.TypesInfo().Defs[name].(*types.Var)
	if !ok {
		return nil, nil, nil, api.Unsupported(context, api.CategoryStatement, source)
	}
	targetName, selected := context.AddressableStorage().Name(context, object)
	var err error
	if !selected {
		targetName, err = context.Names().Declare(object)
	}
	if err != nil {
		return nil, nil, nil, err
	}
	value, err := children.Expression(
		context.
			WithRole(api.RoleLocalValue).
			WithExpectedType(object.Type()),
		source.Rhs[0],
	)
	if err != nil {
		return nil, nil, nil, err
	}
	value, err = context.Values().Copy(
		context.WithRole(api.RoleLocalValue),
		source.Rhs[0],
		object.Type(),
		value,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	if selected {
		value, err = context.AddressableStorage().Cell(
			context,
			children,
			name,
			object.Type(),
			value,
		)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	targetType, typeRequests, err := pointerAnnotation(
		context.WithRole(api.RoleLocalType),
		children,
		name,
		object.Type(),
	)
	if err != nil {
		return nil, nil, nil, err
	}
	if selected {
		targetType = nil
		typeRequests = nil
	}
	declaration := context.Factory().VariableDeclaration(
		context.Factory().Identifier(targetName),
		nil,
		targetType,
		value.Value(),
	)
	return context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{declaration},
			tsgo.NodeFlagsLet,
		),
		value.Before(),
		api.CombineRequests(value.Requests(), typeRequests),
		nil
}

func emitAssignment(
	context api.Context,
	children api.ChildEmitter,
	source *ast.AssignStmt,
) (api.StatementEmission, error) {
	if len(source.Lhs) > 1 {
		return emitParallel(context, children, source)
	}
	if len(source.Lhs) != 1 || len(source.Rhs) != 1 {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	if identifier, ok := source.Lhs[0].(*ast.Ident); ok &&
		identifier.Name == "_" {
		return emitBlankAssignment(context, children, source)
	}
	target, err := children.StoreTarget(
		context.WithRole(api.RoleAssignmentTarget),
		source.Lhs[0],
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	sourceType := context.TypesInfo().TypeOf(source.Rhs[0])
	if sourceType == nil || !types.AssignableTo(sourceType, target.SourceType()) {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	value, err := children.Expression(
		context.
			WithRole(api.RoleAssignmentValue).
			WithExpectedType(target.SourceType()),
		source.Rhs[0],
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	if !target.CopiesValue() {
		value, err = context.Values().Copy(
			context.WithRole(api.RoleAssignmentValue),
			source.Rhs[0],
			target.SourceType(),
			value,
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
	}
	stored, err := target.StoreValue(
		context.WithRole(api.RoleAssignmentTarget),
		source,
		value,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	statements := append(
		stored.Before(),
		context.Factory().ExpressionStatement(stored.Value()),
	)
	return api.NewStatementEmission(statements, stored.Requests())
}

func variableStatement(
	context api.Context,
	flags tsgo.NodeFlags,
	name string,
	value tsgo.Expression,
) tsgo.VariableStatement {
	return typedVariableStatement(context, flags, name, nil, value)
}

func typedVariableStatement(
	context api.Context,
	flags tsgo.NodeFlags,
	name string,
	targetType tsgo.TypeNode,
	value tsgo.Expression,
) tsgo.VariableStatement {
	declaration := context.Factory().VariableDeclaration(
		context.Factory().Identifier(name),
		nil,
		targetType,
		value,
	)
	return context.Factory().VariableStatement(
		nil,
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{declaration},
			flags,
		),
	)
}

func pointerAnnotation(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	sourceType types.Type,
) (tsgo.TypeNode, []api.RootRequest, error) {
	if !context.Values().RequiresExplicitType(context, sourceType) {
		return nil, nil, nil
	}
	target, err := children.RepresentedType(context, source, sourceType)
	if err != nil {
		return nil, nil, err
	}
	return target.Value(), target.Requests(), nil
}

func assignmentStatement(
	context api.Context,
	name string,
	value tsgo.Expression,
) tsgo.ExpressionStatement {
	target := context.Factory().BinaryExpression(
		nil,
		context.Factory().Identifier(name),
		nil,
		context.Factory().BinaryOperatorToken(tsgo.BinaryOperatorEqualsToken),
		value,
	)
	return context.Factory().ExpressionStatement(target)
}
