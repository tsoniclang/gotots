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
	requests []RootRequest
}

func NewExpressionEmission(
	before []tsgo.Statement,
	value tsgo.Expression,
	requests []RootRequest,
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
	requests ...RootRequest,
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

func (e ExpressionEmission) Requests() []RootRequest {
	return slices.Clone(e.requests)
}

type StoreTargetEmission struct {
	setter          bool
	before          []tsgo.Statement
	value           tsgo.Expression
	setterReceiver  ExpressionEmission
	setterMember    string
	setterArguments []ExpressionEmission
	sourceType      types.Type
	requests        []RootRequest
}

func NewStoreTargetEmission(
	value tsgo.Expression,
	sourceType types.Type,
	requests []RootRequest,
) (StoreTargetEmission, error) {
	return NewOrderedStoreTargetEmission(nil, value, sourceType, requests)
}

func NewOrderedStoreTargetEmission(
	before []tsgo.Statement,
	value tsgo.Expression,
	sourceType types.Type,
	requests []RootRequest,
) (StoreTargetEmission, error) {
	switch {
	case slices.Contains(before, nil):
		return StoreTargetEmission{}, &ResultError{
			Result: "store target",
			Reason: "prerequisite statement is nil",
		}
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
		before:     slices.Clone(before),
		value:      value,
		sourceType: sourceType,
		requests:   slices.Clone(requests),
	}, nil
}

func NewSetterStoreTargetEmission(
	receiver ExpressionEmission,
	member string,
	arguments []ExpressionEmission,
	sourceType types.Type,
) (StoreTargetEmission, error) {
	switch {
	case receiver.Value() == nil:
		return StoreTargetEmission{}, &ResultError{
			Result: "setter store target",
			Reason: "target receiver is nil",
		}
	case member == "":
		return StoreTargetEmission{}, &ResultError{
			Result: "setter store target",
			Reason: "target member is empty",
		}
	case sourceType == nil:
		return StoreTargetEmission{}, &ResultError{
			Result: "setter store target",
			Reason: "source type is nil",
		}
	}
	for _, argument := range arguments {
		if argument.Value() == nil {
			return StoreTargetEmission{}, &ResultError{
				Result: "setter store target",
				Reason: "setter argument is nil",
			}
		}
	}
	return StoreTargetEmission{
		setter:          true,
		setterReceiver:  receiver,
		setterMember:    member,
		setterArguments: slices.Clone(arguments),
		sourceType:      sourceType,
	}, nil
}

func (e StoreTargetEmission) IsSetter() bool {
	return e.setter
}

func (e StoreTargetEmission) Before() []tsgo.Statement {
	return slices.Clone(e.before)
}

func (e StoreTargetEmission) Value() tsgo.Expression {
	return e.value
}

func (e StoreTargetEmission) SetterReceiver() ExpressionEmission {
	return e.setterReceiver
}

func (e StoreTargetEmission) SetterMember() string {
	return e.setterMember
}

func (e StoreTargetEmission) SetterArguments() []ExpressionEmission {
	return slices.Clone(e.setterArguments)
}

func (e StoreTargetEmission) SourceType() types.Type {
	return e.sourceType
}

func (e StoreTargetEmission) Requests() []RootRequest {
	if e.setter {
		requests := e.setterReceiver.Requests()
		for _, argument := range e.setterArguments {
			requests = append(requests, argument.Requests()...)
		}
		return requests
	}
	return slices.Clone(e.requests)
}

type StatementEmission struct {
	statements []tsgo.Statement
	requests   []RootRequest
}

func NewStatementEmission(
	statements []tsgo.Statement,
	requests []RootRequest,
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
	requests ...RootRequest,
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

func (e StatementEmission) Requests() []RootRequest {
	return slices.Clone(e.requests)
}

type DeclarationEmission struct {
	declarations []tsgo.Statement
	requests     []RootRequest
}

// EmptyDeclarationEmission is a valid declaration emission that contributes no
// target statements. It is the disposition of a source declaration whose runtime
// form is supplied entirely by later demand-driven reconstruction — an untyped
// constant with no projected uses yet — and never a silent skip of a construct
// that owed a direct declaration. The requests, if any, still propagate.
func EmptyDeclarationEmission(requests ...RootRequest) DeclarationEmission {
	return DeclarationEmission{
		requests: slices.Clone(requests),
	}
}

func NewDeclarationEmission(
	declarations []tsgo.Statement,
	requests []RootRequest,
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
	requests ...RootRequest,
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

func (e DeclarationEmission) Requests() []RootRequest {
	return slices.Clone(e.requests)
}

type TypeEmission struct {
	value    tsgo.TypeNode
	requests []RootRequest
}

func DirectType(value tsgo.TypeNode, requests ...RootRequest) TypeEmission {
	if value == nil {
		panic("direct type target is nil")
	}
	return TypeEmission{value: value, requests: slices.Clone(requests)}
}

func (e TypeEmission) Value() tsgo.TypeNode {
	return e.value
}

func (e TypeEmission) Requests() []RootRequest {
	return slices.Clone(e.requests)
}

type BlockEmission struct {
	value    tsgo.Block
	requests []RootRequest
}

func DirectBlock(value tsgo.Block, requests ...RootRequest) BlockEmission {
	if value == nil {
		panic("direct block target is nil")
	}
	return BlockEmission{value: value, requests: slices.Clone(requests)}
}

func (e BlockEmission) Value() tsgo.Block {
	return e.value
}

func (e BlockEmission) Requests() []RootRequest {
	return slices.Clone(e.requests)
}

type ForInitializerEmission struct {
	value    tsgo.ForInitializer
	requests []RootRequest
}

func DirectForInitializer(
	value tsgo.ForInitializer,
	requests ...RootRequest,
) ForInitializerEmission {
	if value == nil {
		panic("direct for initializer target is nil")
	}
	return ForInitializerEmission{
		value:    value,
		requests: slices.Clone(requests),
	}
}

func ExpressionForInitializer(
	value tsgo.Expression,
	requests ...RootRequest,
) (ForInitializerEmission, error) {
	target, ok := value.(tsgo.ForInitializer)
	if !ok {
		return ForInitializerEmission{}, &ResultError{
			Result: "for initializer",
			Reason: "target expression is not admitted by the TS-Go contract",
		}
	}
	return DirectForInitializer(target, requests...), nil
}

func (e ForInitializerEmission) Value() tsgo.ForInitializer {
	return e.value
}

func (e ForInitializerEmission) Requests() []RootRequest {
	return slices.Clone(e.requests)
}

func CombineRequests(groups ...[]RootRequest) []RootRequest {
	size := 0
	for _, group := range groups {
		size += len(group)
	}
	result := make([]RootRequest, 0, size)
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
