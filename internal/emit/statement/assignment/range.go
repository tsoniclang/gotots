package assignment

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type RangeIterationValue struct {
	sourceType types.Type
	emission   api.ExpressionEmission
	present    bool
	fresh      bool
}

func NewFreshRangeIterationValue(
	sourceType types.Type,
	emission api.ExpressionEmission,
) (RangeIterationValue, error) {
	value, err := NewRangeIterationValue(sourceType, emission)
	if err != nil {
		return RangeIterationValue{}, err
	}
	value.fresh = true
	return value, nil
}

func NewRangeIterationValue(
	sourceType types.Type,
	emission api.ExpressionEmission,
) (RangeIterationValue, error) {
	if sourceType == nil || emission.Value() == nil {
		return RangeIterationValue{}, &api.InvariantError{
			Role:   api.RoleRangeValue,
			Reason: "range iteration value is invalid",
		}
	}
	return RangeIterationValue{
		sourceType: sourceType,
		emission:   emission,
		present:    true,
	}, nil
}

type rangeBinding struct {
	source      ast.Expr
	object      *types.Var
	name        string
	declaration bool
	storage     bool
	target      api.StoreTargetEmission
	value       api.ExpressionEmission
	fresh       bool
}

func EmitRangeIteration(
	context api.Context,
	children api.ChildEmitter,
	source *ast.RangeStmt,
	key RangeIterationValue,
	value RangeIterationValue,
) (api.StatementEmission, error) {
	if source == nil ||
		(source.Tok != token.DEFINE && source.Tok != token.ASSIGN) {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	bindings, err := rangeBindings(
		context,
		children,
		source,
		key,
		value,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	var statements []tsgo.Statement
	var requests []api.RootRequest
	for index := range bindings {
		binding := &bindings[index]
		copied := binding.value
		if !binding.fresh {
			var err error
			copied, err = context.Values().Copy(
				context.WithRole(api.RoleRangeValue),
				nil,
				binding.valueType(),
				binding.value,
			)
			if err != nil {
				return api.StatementEmission{}, err
			}
		}
		name, err := context.Names().Temporary(api.TemporaryRangeValue)
		if err != nil {
			return api.StatementEmission{}, err
		}
		statements = append(statements, copied.Before()...)
		statements = append(
			statements,
			rangeVariable(
				context,
				tsgo.NodeFlagsConst,
				name,
				nil,
				copied.Value(),
			),
		)
		requests = append(requests, copied.Requests()...)
		binding.value = api.DirectExpression(
			context.Factory().Identifier(name),
		)
	}
	for index := range bindings {
		binding := &bindings[index]
		if binding.declaration {
			declaration, declarationRequests, err := declareRangeBinding(
				context,
				children,
				binding,
			)
			if err != nil {
				return api.StatementEmission{}, err
			}
			statements = append(statements, declaration)
			requests = append(requests, declarationRequests...)
			continue
		}
		prepared, before, targetRequests, err :=
			binding.target.PrepareLocation(context)
		if err != nil {
			return api.StatementEmission{}, err
		}
		binding.target = prepared
		statements = append(statements, before...)
		requests = append(requests, targetRequests...)
	}
	for _, binding := range bindings {
		if binding.declaration {
			continue
		}
		stored, err := binding.target.StoreValue(
			context.WithRole(api.RoleAssignmentTarget),
			binding.source,
			binding.value,
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
		statements = append(statements, stored.Before()...)
		statements = append(
			statements,
			context.Factory().ExpressionStatement(stored.Value()),
		)
		requests = append(requests, stored.Requests()...)
	}
	return api.NewStatementEmission(statements, requests)
}

func rangeBindings(
	context api.Context,
	children api.ChildEmitter,
	source *ast.RangeStmt,
	key RangeIterationValue,
	value RangeIterationValue,
) ([]rangeBinding, error) {
	sourceExpressions := []ast.Expr{source.Key, source.Value}
	values := []RangeIterationValue{key, value}
	var bindings []rangeBinding
	for index, expression := range sourceExpressions {
		if expression == nil {
			if values[index].present {
				return nil, &api.InvariantError{
					Role:   api.RoleRangeValue,
					Reason: "range value exists without a target",
				}
			}
			continue
		}
		if identifier, ok := expression.(*ast.Ident); ok &&
			identifier.Name == "_" {
			continue
		}
		selected := values[index]
		if !selected.present {
			return nil, &api.InvariantError{
				Role:   api.RoleRangeValue,
				Reason: "range target has no iteration value",
			}
		}
		if source.Tok == token.DEFINE {
			identifier, ok := expression.(*ast.Ident)
			if !ok {
				return nil,
					api.Unsupported(context, api.CategoryStatement, source)
			}
			object, ok := context.TypesInfo().Defs[identifier].(*types.Var)
			if !ok || !types.AssignableTo(selected.sourceType, object.Type()) {
				return nil,
					api.Unsupported(context, api.CategoryStatement, identifier)
			}
			name, storage := context.AddressableStorage().Name(context, object)
			var err error
			if !storage {
				name, err = context.Names().Declare(object)
			}
			if err != nil {
				return nil, err
			}
			bindings = append(bindings, rangeBinding{
				source:      identifier,
				object:      object,
				name:        name,
				declaration: true,
				storage:     storage,
				value:       selected.emission,
				fresh:       selected.fresh,
			})
			continue
		}
		target, err := children.StoreTarget(
			context.WithRole(api.RoleAssignmentTarget),
			expression,
		)
		if err != nil {
			return nil, err
		}
		if !types.AssignableTo(selected.sourceType, target.SourceType()) {
			return nil,
				api.Unsupported(context, api.CategoryStatement, expression)
		}
		bindings = append(bindings, rangeBinding{
			source: expression,
			target: target,
			value:  selected.emission,
			fresh:  selected.fresh,
		})
	}
	return bindings, nil
}

func (b rangeBinding) valueType() types.Type {
	if b.declaration {
		return b.object.Type()
	}
	return b.target.SourceType()
}

func declareRangeBinding(
	context api.Context,
	children api.ChildEmitter,
	binding *rangeBinding,
) (tsgo.Statement, []api.RootRequest, error) {
	value := binding.value
	if binding.storage {
		var err error
		value, err = context.AddressableStorage().Cell(
			context,
			children,
			binding.source,
			binding.object.Type(),
			value,
		)
		if err != nil {
			return nil, nil, err
		}
	}
	var targetType tsgo.TypeNode
	var requests []api.RootRequest
	if !binding.storage &&
		context.Values().RequiresExplicitType(context, binding.object.Type()) {
		represented, err := children.RepresentedType(
			context.WithRole(api.RoleLocalType),
			binding.source,
			binding.object.Type(),
		)
		if err != nil {
			return nil, nil, err
		}
		targetType = represented.Value()
		requests = append(requests, represented.Requests()...)
	}
	requests = append(requests, value.Requests()...)
	return rangeVariable(
		context,
		tsgo.NodeFlagsLet,
		binding.name,
		targetType,
		value.Value(),
	), requests, nil
}

func rangeVariable(
	context api.Context,
	flags tsgo.NodeFlags,
	name string,
	targetType tsgo.TypeNode,
	value tsgo.Expression,
) tsgo.VariableStatement {
	return context.Factory().VariableStatement(
		nil,
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				context.Factory().VariableDeclaration(
					context.Factory().Identifier(name),
					nil,
					targetType,
					value,
				),
			},
			flags,
		),
	)
}
