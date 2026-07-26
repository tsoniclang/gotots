package api

import (
	"fmt"
	"go/types"
	"slices"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type ExpressionEmission struct {
	before   []tsgo.Statement
	value    tsgo.Expression
	requests []PlacementRequest
}

func NewExpressionEmission(
	before []tsgo.Statement,
	value tsgo.Expression,
	requests []PlacementRequest,
) (ExpressionEmission, error) {
	if value == nil {
		return ExpressionEmission{}, &ResultError{
			Result: "expression",
			Reason: "target value is nil",
		}
	}
	return ExpressionEmission{
		before:   slices.Clone(before),
		value:    value,
		requests: slices.Clone(requests),
	}, nil
}

func DirectExpression(
	value tsgo.Expression,
	requests ...PlacementRequest,
) ExpressionEmission {
	if value == nil {
		panic("direct expression target is nil")
	}
	return ExpressionEmission{
		value:    value,
		requests: slices.Clone(requests),
	}
}

func (e ExpressionEmission) Before() []tsgo.Statement {
	return slices.Clone(e.before)
}

func (e ExpressionEmission) Value() tsgo.Expression {
	return e.value
}

func (e ExpressionEmission) Requests() []PlacementRequest {
	return slices.Clone(e.requests)
}

type StoreTargetEmission struct {
	value      tsgo.Expression
	sourceType types.Type
	requests   []PlacementRequest
}

func NewStoreTargetEmission(
	value tsgo.Expression,
	sourceType types.Type,
	requests []PlacementRequest,
) (StoreTargetEmission, error) {
	switch {
	case value == nil:
		return StoreTargetEmission{}, &ResultError{
			Result: "store target",
			Reason: "target value is nil",
		}
	case sourceType == nil:
		return StoreTargetEmission{}, &ResultError{
			Result: "store target",
			Reason: "source type is nil",
		}
	}
	return StoreTargetEmission{
		value:      value,
		sourceType: sourceType,
		requests:   slices.Clone(requests),
	}, nil
}

func (e StoreTargetEmission) Value() tsgo.Expression {
	return e.value
}

func (e StoreTargetEmission) SourceType() types.Type {
	return e.sourceType
}

func (e StoreTargetEmission) Requests() []PlacementRequest {
	return slices.Clone(e.requests)
}

type StatementEmission struct {
	statements []tsgo.Statement
	requests   []PlacementRequest
}

func NewStatementEmission(
	statements []tsgo.Statement,
	requests []PlacementRequest,
) (StatementEmission, error) {
	for _, statement := range statements {
		if statement == nil {
			return StatementEmission{}, &ResultError{
				Result: "statement",
				Reason: "target statement is nil",
			}
		}
	}
	return StatementEmission{
		statements: slices.Clone(statements),
		requests:   slices.Clone(requests),
	}, nil
}

func DirectStatement(
	statement tsgo.Statement,
	requests ...PlacementRequest,
) StatementEmission {
	if statement == nil {
		panic("direct statement target is nil")
	}
	return StatementEmission{
		statements: []tsgo.Statement{statement},
		requests:   slices.Clone(requests),
	}
}

func (e StatementEmission) Statements() []tsgo.Statement {
	return slices.Clone(e.statements)
}

func (e StatementEmission) Requests() []PlacementRequest {
	return slices.Clone(e.requests)
}

type DeclarationEmission struct {
	declarations []tsgo.Statement
	requests     []PlacementRequest
}

func NewDeclarationEmission(
	declarations []tsgo.Statement,
	requests []PlacementRequest,
) (DeclarationEmission, error) {
	if len(declarations) == 0 {
		return DeclarationEmission{}, &ResultError{
			Result: "declaration",
			Reason: "target declarations are empty",
		}
	}
	for _, declaration := range declarations {
		if declaration == nil {
			return DeclarationEmission{}, &ResultError{
				Result: "declaration",
				Reason: "target declaration is nil",
			}
		}
	}
	return DeclarationEmission{
		declarations: slices.Clone(declarations),
		requests:     slices.Clone(requests),
	}, nil
}

func DirectDeclaration(
	declaration tsgo.Statement,
	requests ...PlacementRequest,
) DeclarationEmission {
	if declaration == nil {
		panic("direct declaration target is nil")
	}
	return DeclarationEmission{
		declarations: []tsgo.Statement{declaration},
		requests:     slices.Clone(requests),
	}
}

func (e DeclarationEmission) Declarations() []tsgo.Statement {
	return slices.Clone(e.declarations)
}

func (e DeclarationEmission) Requests() []PlacementRequest {
	return slices.Clone(e.requests)
}

type TypeEmission struct {
	value    tsgo.TypeNode
	requests []PlacementRequest
}

func DirectType(value tsgo.TypeNode, requests ...PlacementRequest) TypeEmission {
	if value == nil {
		panic("direct type target is nil")
	}
	return TypeEmission{value: value, requests: slices.Clone(requests)}
}

func (e TypeEmission) Value() tsgo.TypeNode {
	return e.value
}

func (e TypeEmission) Requests() []PlacementRequest {
	return slices.Clone(e.requests)
}

type BlockEmission struct {
	value    tsgo.Block
	requests []PlacementRequest
}

func DirectBlock(value tsgo.Block, requests ...PlacementRequest) BlockEmission {
	if value == nil {
		panic("direct block target is nil")
	}
	return BlockEmission{value: value, requests: slices.Clone(requests)}
}

func (e BlockEmission) Value() tsgo.Block {
	return e.value
}

func (e BlockEmission) Requests() []PlacementRequest {
	return slices.Clone(e.requests)
}

type ForInitializerEmission struct {
	value    tsgo.ForInitializer
	requests []PlacementRequest
}

func DirectForInitializer(
	value tsgo.ForInitializer,
	requests ...PlacementRequest,
) ForInitializerEmission {
	if value == nil {
		panic("direct for initializer target is nil")
	}
	return ForInitializerEmission{
		value:    value,
		requests: slices.Clone(requests),
	}
}

func (e ForInitializerEmission) Value() tsgo.ForInitializer {
	return e.value
}

func (e ForInitializerEmission) Requests() []PlacementRequest {
	return slices.Clone(e.requests)
}

func CombineRequests(groups ...[]PlacementRequest) []PlacementRequest {
	size := 0
	for _, group := range groups {
		size += len(group)
	}
	result := make([]PlacementRequest, 0, size)
	for _, group := range groups {
		result = append(result, group...)
	}
	return result
}

type ResultError struct {
	Result string
	Reason string
}

func (e *ResultError) Error() string {
	return fmt.Sprintf("create %s emission: %s", e.Result, e.Reason)
}
