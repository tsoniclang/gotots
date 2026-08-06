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
	accessor          bool
	accessorFunction  bool
	property          bool
	before            []tsgo.Statement
	value             tsgo.Expression
	propertyReceiver  ExpressionEmission
	propertyMember    string
	accessorReceiver  ExpressionEmission
	getterFunction    ExpressionEmission
	setterFunction    ExpressionEmission
	getterMember      string
	setterMember      string
	accessorArguments []ExpressionEmission
	locationCaptured  bool
	copiesValue       bool
	storage           StoreTargetStorage
	stableIdentity    bool
	sourceType        types.Type
	requests          []RootRequest
}

type StoreTargetStorage uint8

const (
	StoreTargetStorageLogical StoreTargetStorage = iota
	StoreTargetStorageCanonical
	StoreTargetStorageContainer
)

func (e StoreTargetEmission) Valid() bool {
	if e.sourceType == nil {
		return false
	}
	if !e.accessor {
		return e.value != nil
	}
	if e.accessorFunction {
		return e.getterFunction.Value() != nil &&
			e.setterFunction.Value() != nil
	}
	return e.accessorReceiver.Value() != nil &&
		e.getterMember != "" &&
		e.setterMember != ""
}

func NewPropertyStoreTargetEmission(
	factory tsgo.Factory,
	receiver ExpressionEmission,
	member string,
	sourceType types.Type,
) (StoreTargetEmission, error) {
	switch {
	case receiver.Value() == nil:
		return StoreTargetEmission{}, &ResultError{
			Result: "property store target",
			Reason: "target receiver is nil",
		}
	case member == "":
		return StoreTargetEmission{}, &ResultError{
			Result: "property store target",
			Reason: "target member is empty",
		}
	case sourceType == nil:
		return StoreTargetEmission{}, &ResultError{
			Result: "property store target",
			Reason: "source type is nil",
		}
	}
	return StoreTargetEmission{
		property: true,
		value: factory.PropertyAccessExpression(
			receiver.Value(),
			nil,
			factory.Identifier(member),
			tsgo.NodeFlagsNone,
		),
		propertyReceiver: receiver,
		propertyMember:   member,
		sourceType:       sourceType,
	}, nil
}

func NewCanonicalStoragePropertyStoreTargetEmission(
	factory tsgo.Factory,
	receiver ExpressionEmission,
	member string,
	sourceType types.Type,
) (StoreTargetEmission, error) {
	target, err := NewPropertyStoreTargetEmission(
		factory,
		receiver,
		member,
		sourceType,
	)
	if err != nil {
		return StoreTargetEmission{}, err
	}
	target.storage = StoreTargetStorageCanonical
	return target, nil
}

func NewStoreTargetEmission(
	value tsgo.Expression,
	sourceType types.Type,
	requests []RootRequest,
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

func NewCanonicalStorageTargetEmission(
	value tsgo.Expression,
	sourceType types.Type,
	requests []RootRequest,
) (StoreTargetEmission, error) {
	target, err := NewStoreTargetEmission(value, sourceType, requests)
	if err != nil {
		return StoreTargetEmission{}, err
	}
	target.storage = StoreTargetStorageCanonical
	return target, nil
}

func (e StoreTargetEmission) IsAccessor() bool {
	return e.accessor
}

func (e StoreTargetEmission) IsProperty() bool {
	return e.property
}

func (e StoreTargetEmission) CopiesValue() bool {
	return e.copiesValue
}

func (e StoreTargetEmission) UsesCanonicalStorage() bool {
	return e.storage == StoreTargetStorageCanonical
}

func (e StoreTargetEmission) UsesContainerStorage() bool {
	return e.storage == StoreTargetStorageContainer
}

func (e StoreTargetEmission) Before() []tsgo.Statement {
	before := slices.Clone(e.before)
	if e.property && !e.locationCaptured {
		before = append(before, e.propertyReceiver.Before()...)
	}
	return before
}

func (e StoreTargetEmission) Value() tsgo.Expression {
	return e.value
}

func (e StoreTargetEmission) AccessorReceiver() ExpressionEmission {
	return e.accessorReceiver
}

func (e StoreTargetEmission) GetterMember() string {
	return e.getterMember
}

func (e StoreTargetEmission) SetterMember() string {
	return e.setterMember
}

func (e StoreTargetEmission) AccessorArguments() []ExpressionEmission {
	return slices.Clone(e.accessorArguments)
}

func (e StoreTargetEmission) SourceType() types.Type {
	return e.sourceType
}

func (e StoreTargetEmission) Requests() []RootRequest {
	requests := slices.Clone(e.requests)
	if e.property {
		requests = CombineRequests(
			requests,
			e.propertyReceiver.Requests(),
		)
	}
	if e.accessor {
		if e.accessorFunction {
			requests = CombineRequests(
				requests,
				e.getterFunction.Requests(),
				e.setterFunction.Requests(),
			)
		} else {
			requests = CombineRequests(
				requests,
				e.accessorReceiver.Requests(),
			)
		}
		for _, argument := range e.accessorArguments {
			requests = CombineRequests(requests, argument.Requests())
		}
	}
	return requests
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
	declarations              []tsgo.Statement
	classOwner                *types.TypeName
	classMembers              []tsgo.ClassElement
	additionalPackageBindings []string
	requests                  []RootRequest
	disposition               DeclarationDisposition
}

type DeclarationDisposition uint8

const (
	DeclarationDispositionInvalid DeclarationDisposition = iota
	DeclarationDispositionMaterialized
	DeclarationDispositionCoverageOnly
	DeclarationDispositionClassMemberContribution
)

func (d DeclarationDisposition) Valid() bool {
	return d == DeclarationDispositionMaterialized ||
		d == DeclarationDispositionCoverageOnly ||
		d == DeclarationDispositionClassMemberContribution
}

// CoverageOnlyDeclarationEmission is the sole declaration result with no
// target statement. It records that the source declaration was visited for
// whole-file coverage but has no requested runtime representation.
func CoverageOnlyDeclarationEmission(
	requests ...RootRequest,
) DeclarationEmission {
	return DeclarationEmission{
		requests:    slices.Clone(requests),
		disposition: DeclarationDispositionCoverageOnly,
	}
}

func ClassMemberContributionEmission(
	owner *types.TypeName,
	members []tsgo.ClassElement,
	requests []RootRequest,
) (DeclarationEmission, error) {
	return ClassMemberAndSupportContributionEmission(
		owner,
		members,
		nil,
		requests,
	)
}

func ClassMemberAndSupportContributionEmission(
	owner *types.TypeName,
	members []tsgo.ClassElement,
	support []tsgo.Statement,
	requests []RootRequest,
) (DeclarationEmission, error) {
	if owner == nil {
		return DeclarationEmission{}, &ResultError{
			Result: "class-member contribution",
			Reason: "target class owner is nil",
		}
	}
	if len(members) == 0 {
		return DeclarationEmission{}, &ResultError{
			Result: "class-member contribution",
			Reason: "target members are empty",
		}
	}
	for _, member := range members {
		if member == nil {
			return DeclarationEmission{}, &ResultError{
				Result: "class-member contribution",
				Reason: "target member is nil",
			}
		}
	}
	for _, declaration := range support {
		if declaration == nil {
			return DeclarationEmission{}, &ResultError{
				Result: "class-member contribution",
				Reason: "support declaration is nil",
			}
		}
	}
	return DeclarationEmission{
		declarations: slices.Clone(support),
		classOwner:   owner,
		classMembers: slices.Clone(members),
		requests:     slices.Clone(requests),
		disposition:  DeclarationDispositionClassMemberContribution,
	}, nil
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
		disposition:  DeclarationDispositionMaterialized,
	}, nil
}

func NewDeclarationEmissionWithAdditionalPackageBindings(
	declarations []tsgo.Statement,
	requests []RootRequest,
	bindings []string,
) (DeclarationEmission, error) {
	result, err := NewDeclarationEmission(declarations, requests)
	if err != nil {
		return DeclarationEmission{}, err
	}
	if len(bindings) == 0 {
		return DeclarationEmission{}, &ResultError{
			Result: "declaration",
			Reason: "additional package bindings are empty",
		}
	}
	result.additionalPackageBindings = slices.Clone(bindings)
	slices.Sort(result.additionalPackageBindings)
	for index, binding := range result.additionalPackageBindings {
		if binding == "" {
			return DeclarationEmission{}, &ResultError{
				Result: "declaration",
				Reason: "additional package binding is empty",
			}
		}
		if index != 0 && result.additionalPackageBindings[index-1] == binding {
			return DeclarationEmission{}, &ResultError{
				Result: "declaration",
				Reason: "additional package binding is duplicated",
			}
		}
	}
	return result, nil
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
		disposition:  DeclarationDispositionMaterialized,
	}
}

func (e DeclarationEmission) Declarations() []tsgo.Statement {
	return slices.Clone(e.declarations)
}

func (e DeclarationEmission) Requests() []RootRequest {
	return slices.Clone(e.requests)
}

func (e DeclarationEmission) AdditionalPackageBindings() []string {
	return slices.Clone(e.additionalPackageBindings)
}

func (e DeclarationEmission) ClassMemberContribution() (
	*types.TypeName,
	[]tsgo.ClassElement,
	bool,
) {
	if e.disposition != DeclarationDispositionClassMemberContribution ||
		e.classOwner == nil ||
		len(e.classMembers) == 0 {
		return nil, nil, false
	}
	return e.classOwner, slices.Clone(e.classMembers), true
}

func (e DeclarationEmission) Disposition() DeclarationDisposition {
	return e.disposition
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

func CombineRequests(groups ...[]RootRequest) []RootRequest {
	return combineRootRequests(groups...)
}

type ResultError struct {
	Result string
	Reason string
}

func (e *ResultError) Error() string {
	return fmt.Sprintf("create %s emission: %s", e.Result, e.Reason)
}

type GeneratedArtifactPlacementError struct {
	TypeName string
	Reason   string
}

func (e *GeneratedArtifactPlacementError) Error() string {
	if e.TypeName == "" {
		return "place generated type: " + e.Reason
	}
	return fmt.Sprintf("place generated type containing %q: %s", e.TypeName, e.Reason)
}

type GeneratedArtifactShapeError struct {
	Artifact string
	Reason   string
}

func (e *GeneratedArtifactShapeError) Error() string {
	if e.Artifact == "" {
		return "emit generated type: " + e.Reason
	}
	return fmt.Sprintf("emit generated type %q: %s", e.Artifact, e.Reason)
}
