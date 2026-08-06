package localdeclaration

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	conversionexpression "github.com/tsoniclang/gotots/internal/emit/expression/conversion"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type binding struct {
	sourceName      *ast.Ident
	object          *types.Var
	sourceType      types.Type
	sourceValue     ast.Expr
	value           api.ExpressionEmission
	omitInitializer bool
}

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.DeclStmt,
) (api.StatementEmission, error) {
	declaration, ok := source.Decl.(*ast.GenDecl)
	if !ok || declaration.Tok != token.VAR ||
		len(declaration.Specs) == 0 {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}

	var statements []tsgo.Statement
	var requests []api.RootRequest
	for _, sourceSpec := range declaration.Specs {
		spec, ok := sourceSpec.(*ast.ValueSpec)
		if !ok {
			return api.StatementEmission{},
				api.Unsupported(
					context.WithRole(api.RoleLocalDeclaration),
					api.CategoryStatement,
					sourceSpec,
				)
		}
		target, targetRequests, err := emitSpec(context, children, spec)
		if err != nil {
			return api.StatementEmission{}, err
		}
		statements = append(statements, target...)
		requests = append(requests, targetRequests...)
	}
	return api.NewStatementEmission(statements, requests)
}

func emitSpec(
	context api.Context,
	children api.ChildEmitter,
	source *ast.ValueSpec,
) ([]tsgo.Statement, []api.RootRequest, error) {
	if len(source.Names) == 0 ||
		(len(source.Values) == 0 && source.Type == nil) {
		return nil, nil,
			api.Unsupported(
				context.WithRole(api.RoleLocalDeclaration),
				api.CategoryStatement,
				source,
			)
	}
	if len(source.Values) == 1 && len(source.Names) > 1 {
		if results, ok := context.TypesInfo().TypeOf(source.Values[0]).(*types.Tuple); ok {
			return emitMultipleResultSpec(context, children, source, results)
		}
	}
	if len(source.Values) != 0 && len(source.Names) != len(source.Values) {
		return nil, nil,
			api.Unsupported(
				context.WithRole(api.RoleLocalDeclaration),
				api.CategoryStatement,
				source,
			)
	}
	if hasBlankName(source.Names) {
		return emitBlankAwareSpec(context, children, source)
	}

	bindings := make([]binding, 0, len(source.Names))
	var requests []api.RootRequest
	for index, sourceName := range source.Names {
		object, ok := context.TypesInfo().DefOf(sourceName).(*types.Var)
		if !ok {
			return nil, nil,
				api.Unsupported(
					context.WithRole(api.RoleLocalDeclaration),
					api.CategoryStatement,
					sourceName,
				)
		}
		objectType := context.TypesInfo().TypeOfObject(object)
		if source.Type != nil &&
			!types.Identical(context.TypesInfo().TypeOf(source.Type), objectType) {
			return nil, nil,
				api.Unsupported(
					context.WithRole(api.RoleLocalType),
					api.CategoryType,
					source.Type,
				)
		}

		callableZero := len(source.Values) == 0 &&
			omitCallableZeroInitializer(objectType)
		var value api.ExpressionEmission
		var err error
		if callableZero {
			value = api.DirectExpression(
				context.Factory().VoidExpression(
					context.Factory().NumericLiteral("0", tsgo.TokenFlagsNone),
				),
			)
		} else {
			value, err = localValue(
				context,
				children,
				source,
				sourceName,
				index,
				objectType,
			)
			if err != nil {
				return nil, nil, err
			}
		}
		var sourceValue ast.Expr
		if len(source.Values) != 0 {
			sourceValue = source.Values[index]
		}
		bindings = append(bindings, binding{
			sourceName:      sourceName,
			object:          object,
			sourceType:      objectType,
			sourceValue:     sourceValue,
			value:           value,
			omitInitializer: callableZero,
		})
	}

	hasPrerequisites := false
	for _, binding := range bindings {
		if len(binding.value.Before()) != 0 {
			hasPrerequisites = true
			break
		}
	}
	var statements []tsgo.Statement
	if hasPrerequisites {
		for index := range bindings {
			binding := &bindings[index]
			statements = append(statements, binding.value.Before()...)
			temporaryName, err := context.Names().Temporary(
				api.TemporaryAssignmentValue,
			)
			if err != nil {
				return nil, nil, err
			}
			statements = append(
				statements,
				context.Factory().VariableStatement(
					nil,
					context.Factory().VariableDeclarationList(
						[]tsgo.VariableDeclaration{
							context.Factory().VariableDeclaration(
								context.Factory().Identifier(temporaryName),
								nil,
								nil,
								binding.value.Value(),
							),
						},
						tsgo.NodeFlagsConst,
					),
				),
			)
			binding.value = api.DirectExpression(
				context.Factory().Identifier(temporaryName),
				binding.value.Requests()...,
			)
		}
	}

	if len(bindings) != 0 && context.IsGotoLocal(bindings[0].object) {
		for _, binding := range bindings {
			target, targetRequests, selected, err := gotoLocalAssignment(
				context,
				children,
				binding,
			)
			if err != nil {
				return nil, nil, err
			}
			if !selected {
				return nil, nil, &api.InvariantError{
					Role:   api.RoleLocalDeclaration,
					Reason: "goto declaration has mixed storage ownership",
				}
			}
			statements = append(statements, target...)
			requests = append(requests, targetRequests...)
		}
		return statements, requests, nil
	}

	declarations := make([]tsgo.VariableDeclaration, 0, len(bindings))
	for _, binding := range bindings {
		declaration, declarationBefore, declarationRequests, err :=
			localVariableDeclaration(context, children, binding)
		if err != nil {
			return nil, nil, err
		}
		statements = append(statements, declarationBefore...)
		declarations = append(declarations, declaration)
		requests = append(requests, declarationRequests...)
	}
	statements = append(
		statements,
		context.Factory().VariableStatement(
			nil,
			context.Factory().VariableDeclarationList(
				declarations,
				tsgo.NodeFlagsLet,
			),
		),
	)
	return statements, requests, nil
}

func omitCallableZeroInitializer(sourceType types.Type) bool {
	_, callableValue := callable.Signature(sourceType)
	_, definedValue := definedtype.ResolveCallable(sourceType)
	return callableValue && !definedValue
}

func localVariableDeclaration(
	context api.Context,
	children api.ChildEmitter,
	binding binding,
) (
	tsgo.VariableDeclaration,
	[]tsgo.Statement,
	[]api.RootRequest,
	error,
) {
	sourceName := binding.sourceName
	object := binding.object
	value := binding.value
	targetName, selected := context.AddressableStorage().Name(context, object)
	var err error
	if !selected {
		targetName, err = context.Names().Declare(object)
	} else {
		value, err = context.AddressableStorage().Cell(
			context,
			children,
			sourceName,
			binding.sourceType,
			value,
		)
	}
	if err != nil {
		return nil, nil, nil, err
	}
	var targetType tsgo.TypeNode
	requests := value.Requests()
	requiresInferenceAnnotation, err :=
		conversionexpression.RequiresInferenceAnnotation(
			context,
			binding.sourceValue,
			binding.sourceType,
		)
	if err != nil {
		return nil, nil, nil, err
	}
	if !selected && (requiresInferenceAnnotation ||
		context.Values().RequiresExplicitType(context, binding.sourceType)) {
		represented, err := children.RepresentedType(
			context.WithRole(api.RoleLocalType),
			sourceName,
			binding.sourceType,
		)
		if err != nil {
			return nil, nil, nil, err
		}
		targetType = represented.Value()
		requests = append(requests, represented.Requests()...)
	}
	var initializer tsgo.Expression = value.Value()
	if binding.omitInitializer && !selected {
		initializer = nil
	}
	return context.Factory().VariableDeclaration(
		context.Factory().Identifier(targetName),
		nil,
		targetType,
		initializer,
	), value.Before(), requests, nil
}

func hasBlankName(names []*ast.Ident) bool {
	for _, name := range names {
		if name != nil && name.Name == "_" {
			return true
		}
	}
	return false
}

func localValue(
	context api.Context,
	children api.ChildEmitter,
	source *ast.ValueSpec,
	sourceName *ast.Ident,
	index int,
	objectType types.Type,
) (api.ExpressionEmission, error) {
	valueContext := context.
		WithRole(api.RoleLocalValue).
		WithExpectedType(objectType)
	if len(source.Values) == 0 {
		return context.Values().Zero(valueContext, sourceName, objectType)
	}
	sourceValue := source.Values[index]
	valueType := context.TypesInfo().TypeOf(sourceValue)
	if valueType == nil || !types.AssignableTo(valueType, objectType) {
		return api.ExpressionEmission{},
			api.Unsupported(
				valueContext,
				api.CategoryExpression,
				sourceValue,
			)
	}
	value, err := children.Expression(valueContext, sourceValue)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return context.Values().Transfer(
		valueContext,
		sourceValue,
		valueType,
		objectType,
		api.ValueTransferCopy,
		value,
	)
}
