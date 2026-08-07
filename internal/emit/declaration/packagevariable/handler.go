package packagevariable

import (
	"go/ast"
	"go/constant"
	"go/types"
	"slices"

	"github.com/tsoniclang/gotots/internal/emit/api"
	constantvalue "github.com/tsoniclang/gotots/internal/emit/constant"
	"github.com/tsoniclang/gotots/internal/emit/resulttuple"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const (
	StateClassName = "$PackageState"
	StateValueName = "$state"
)

type StorageEmission struct {
	field                    tsgo.PropertyDeclaration
	initializationStatements []tsgo.Statement
	stateRequests            []api.RootRequest
	assemblyRequests         []api.RootRequest
}

func EmitStorage(
	stateContext api.Context,
	assemblyContext api.Context,
	children api.ChildEmitter,
	source ast.Node,
	variable *types.Var,
	embedded *load.EmbedValue,
) (StorageEmission, error) {
	if variable == nil ||
		variable.IsField() ||
		variable.Pkg() == nil ||
		variable.Pkg() != stateContext.TypesPackage() ||
		variable.Parent() != variable.Pkg().Scope() {
		return StorageEmission{}, &api.InvariantError{
			Role:   stateContext.Role(),
			Reason: "package-state storage has no package variable",
		}
	}
	stateReference, err := stateContext.Names().PackageVariable(variable)
	if err != nil {
		return StorageEmission{}, err
	}
	if len(stateReference.Requests()) != 0 {
		return StorageEmission{}, &api.InvariantError{
			Role:   stateContext.Role(),
			Reason: "package-state field requested an import of its own state",
		}
	}
	targetType, err := children.RepresentedType(
		stateContext.WithRole(api.RolePackageVariableType),
		source,
		variable.Type(),
	)
	if err != nil {
		return StorageEmission{}, err
	}

	assemblyReference, err := assemblyContext.Names().PackageVariable(variable)
	if err != nil {
		return StorageEmission{}, err
	}
	initial, role, err := emitInitialStorageValue(
		assemblyContext,
		source,
		variable,
		embedded,
	)
	if err != nil {
		return StorageEmission{}, err
	}
	target, err := api.NewStoreTargetEmission(
		assemblyReference.Expression(assemblyContext.Factory()),
		variable.Type(),
		assemblyReference.Requests(),
	)
	if err != nil {
		return StorageEmission{}, err
	}
	assigned, err := target.StoreValue(
		assemblyContext.WithRole(role),
		source,
		initial,
	)
	if err != nil {
		return StorageEmission{}, err
	}
	initializationStatements := assigned.Before()
	initializationStatements = append(
		initializationStatements,
		assemblyContext.Factory().ExpressionStatement(assigned.Value()),
	)
	return StorageEmission{
		field: stateContext.Factory().PropertyDeclaration(
			[]tsgo.ModifierLike{stateContext.Factory().DeclareKeyword()},
			stateContext.Factory().Identifier(stateReference.FieldName()),
			nil,
			targetType.Value(),
			nil,
		),
		initializationStatements: initializationStatements,
		stateRequests:            targetType.Requests(),
		assemblyRequests: api.CombineRequests(
			assigned.Requests(),
		),
	}, nil
}

func emitInitialStorageValue(
	context api.Context,
	source ast.Node,
	variable *types.Var,
	embedded *load.EmbedValue,
) (api.ExpressionEmission, api.Role, error) {
	if embedded == nil {
		value, err := context.Values().Zero(
			context.WithRole(api.RolePackageVariableZero),
			source,
			variable.Type(),
		)
		return value, api.RolePackageVariableZero, err
	}
	if embedded.Kind() != load.EmbedString {
		return api.ExpressionEmission{}, api.RolePackageVariableValue,
			api.Unsupported(
				context.WithRole(api.RolePackageVariableValue),
				api.CategoryDeclaration,
				source,
			)
	}
	content, ok := embedded.String()
	if !ok {
		return api.ExpressionEmission{}, api.RolePackageVariableValue,
			&api.InvariantError{
				Role:   context.Role(),
				Reason: "embedded string storage lacks exactly one selected file",
			}
	}
	value, err := constantvalue.EmitValue(
		context.WithRole(api.RolePackageVariableValue),
		source,
		variable.Type(),
		constant.MakeString(content),
	)
	return value, api.RolePackageVariableValue, err
}

func EmitInitializer(
	context api.Context,
	children api.ChildEmitter,
	initializer *types.Initializer,
) (api.StatementEmission, error) {
	if initializer == nil || initializer.Rhs == nil {
		return api.StatementEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "package initializer is nil",
		}
	}
	if len(initializer.Lhs) > 1 {
		return emitMultipleInitializer(context, children, initializer)
	}
	if len(initializer.Lhs) != 1 {
		return api.StatementEmission{},
			api.Unsupported(
				context.WithRole(api.RolePackageVariableValue),
				api.CategoryDeclaration,
				initializer.Rhs,
			)
	}
	variable := initializer.Lhs[0]
	if variable != nil && variable.Name() == "_" {
		return emitBlankInitializer(context, children, initializer, variable)
	}
	if variable == nil ||
		variable.IsField() ||
		variable.Pkg() == nil ||
		variable.Pkg() != context.TypesPackage() ||
		variable.Parent() != variable.Pkg().Scope() {
		return api.StatementEmission{},
			api.Unsupported(
				context.WithRole(api.RolePackageVariableValue),
				api.CategoryDeclaration,
				initializer.Rhs,
			)
	}
	valueType := context.TypesInfo().TypeOf(initializer.Rhs)
	if valueType == nil || !types.AssignableTo(valueType, variable.Type()) {
		return api.StatementEmission{},
			api.Unsupported(
				context.WithRole(api.RolePackageVariableValue),
				api.CategoryExpression,
				initializer.Rhs,
			)
	}
	valueContext := context.
		WithRole(api.RolePackageVariableValue).
		WithExpectedType(variable.Type())
	value, err := children.Expression(valueContext, initializer.Rhs)
	if err != nil {
		return api.StatementEmission{}, err
	}
	value, err = context.Values().Transfer(
		valueContext,
		initializer.Rhs,
		valueType,
		variable.Type(),
		api.ValueTransferCopy,
		value,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	reference, err := context.Names().PackageVariable(variable)
	if err != nil {
		return api.StatementEmission{}, err
	}
	target, err := api.NewStoreTargetEmission(
		reference.Expression(context.Factory()),
		variable.Type(),
		reference.Requests(),
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	assigned, err := target.StoreValue(
		context.WithRole(api.RolePackageVariableValue),
		initializer.Rhs,
		value,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	statements := assigned.Before()
	statements = append(
		statements,
		context.Factory().ExpressionStatement(assigned.Value()),
	)
	return api.NewStatementEmission(
		statements,
		assigned.Requests(),
	)
}

func emitBlankInitializer(
	context api.Context,
	children api.ChildEmitter,
	initializer *types.Initializer,
	variable *types.Var,
) (api.StatementEmission, error) {
	valueType := context.TypesInfo().TypeOf(initializer.Rhs)
	if variable == nil ||
		variable.Pkg() != context.TypesPackage() ||
		valueType == nil ||
		!types.AssignableTo(valueType, variable.Type()) {
		return api.StatementEmission{},
			api.Unsupported(
				context.WithRole(api.RolePackageVariableValue),
				api.CategoryExpression,
				initializer.Rhs,
			)
	}
	value, err := children.Expression(
		context.
			WithRole(api.RolePackageVariableValue).
			WithExpectedType(variable.Type()),
		initializer.Rhs,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	statements := value.Before()
	statements = append(
		statements,
		context.Factory().ExpressionStatement(value.Value()),
	)
	return api.NewStatementEmission(statements, value.Requests())
}

func emitMultipleInitializer(
	context api.Context,
	children api.ChildEmitter,
	initializer *types.Initializer,
) (api.StatementEmission, error) {
	resultType := context.TypesInfo().TypeOf(initializer.Rhs)
	if resultType == nil {
		return api.StatementEmission{},
			api.Unsupported(
				context.WithRole(api.RolePackageVariableValue),
				api.CategoryExpression,
				initializer.Rhs,
			)
	}
	results, ok := types.Unalias(resultType).(*types.Tuple)
	if !ok || results.Len() != len(initializer.Lhs) {
		return api.StatementEmission{},
			api.Unsupported(
				context.WithRole(api.RolePackageVariableValue),
				api.CategoryDeclaration,
				initializer.Rhs,
			)
	}
	for index, variable := range initializer.Lhs {
		if variable == nil ||
			(variable.Name() != "_" &&
				(variable.IsField() ||
					variable.Pkg() == nil ||
					variable.Pkg() != context.TypesPackage() ||
					variable.Parent() != variable.Pkg().Scope() ||
					!types.AssignableTo(
						results.At(index).Type(),
						variable.Type(),
					))) {
			return api.StatementEmission{},
				api.Unsupported(
					context.WithRole(api.RolePackageVariableValue),
					api.CategoryDeclaration,
					initializer.Rhs,
				)
		}
	}
	capture, err := resulttuple.Emit(
		context,
		children,
		initializer.Rhs,
		results,
		api.RolePackageVariableValue,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	statements := capture.Statements()
	requests := capture.Requests()
	for index, variable := range initializer.Lhs {
		if variable.Name() == "_" {
			continue
		}
		reference, err := context.Names().PackageVariable(variable)
		if err != nil {
			return api.StatementEmission{}, err
		}
		element, err := capture.Element(context, index)
		if err != nil {
			return api.StatementEmission{}, err
		}
		value, err := context.Values().Transfer(
			context.WithRole(api.RolePackageVariableValue),
			initializer.Rhs,
			results.At(index).Type(),
			variable.Type(),
			api.ValueTransferCopy,
			api.DirectExpression(element),
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
		target, err := api.NewStoreTargetEmission(
			reference.Expression(context.Factory()),
			variable.Type(),
			reference.Requests(),
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
		assigned, err := target.StoreValue(
			context.WithRole(api.RolePackageVariableValue),
			initializer.Rhs,
			value,
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
		statements = append(statements, assigned.Before()...)
		statements = append(
			statements,
			context.Factory().ExpressionStatement(assigned.Value()),
		)
		requests = append(
			requests,
			assigned.Requests()...,
		)
	}
	return api.NewStatementEmission(statements, requests)
}

func StateDeclarations(
	factory tsgo.Factory,
	fields []tsgo.PropertyDeclaration,
) ([]tsgo.Statement, error) {
	if len(fields) == 0 {
		return nil, &api.InvariantError{
			Role:   api.RoleFileDeclaration,
			Reason: "package state has no fields",
		}
	}
	members := make([]tsgo.ClassElement, 0, len(fields))
	for _, field := range fields {
		if field == nil {
			return nil, &api.InvariantError{
				Role:   api.RolePackageVariableType,
				Reason: "package state has a nil field",
			}
		}
		members = append(members, field)
	}
	class := factory.ClassDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		factory.Identifier(StateClassName),
		nil,
		nil,
		members,
	)
	state := factory.VariableStatement(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				factory.VariableDeclaration(
					factory.Identifier(StateValueName),
					nil,
					nil,
					factory.NewExpression(
						factory.Identifier(StateClassName),
						nil,
						[]tsgo.Expression{},
					),
				),
			},
			tsgo.NodeFlagsConst,
		),
	)
	return []tsgo.Statement{class, state}, nil
}

func (e StorageEmission) Field() tsgo.PropertyDeclaration {
	return e.field
}

func (e StorageEmission) InitializationStatements() []tsgo.Statement {
	return slices.Clone(e.initializationStatements)
}

func (e StorageEmission) StateRequests() []api.RootRequest {
	return slices.Clone(e.stateRequests)
}

func (e StorageEmission) AssemblyRequests() []api.RootRequest {
	return slices.Clone(e.assemblyRequests)
}
