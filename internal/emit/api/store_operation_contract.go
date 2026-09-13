package api

import (
	genericoperation "github.com/tsoniclang/gotots/internal/emit/api/genericoperation"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
	"go/ast"
	"go/token"
	"go/types"
	"slices"
)

type StableAssignmentValues interface {
	AssignStable(
		Context,
		ast.Node,
		types.Type,
		tsgo.Expression,
		ExpressionEmission,
	) (ExpressionEmission, error)
}

func (e StoreTargetEmission) WithStableIdentity() (StoreTargetEmission, error) {
	if e.sourceType == nil || e.copiesValue {
		return StoreTargetEmission{}, &ResultError{Result: "stable-identity store target", Reason: "an addressable aggregate location is required"}
	}
	switch e.sourceType.Underlying().(type) {
	case *types.Array, *types.Struct:
	default:
		return StoreTargetEmission{}, &ResultError{Result: "stable-identity store target", Reason: "source type is not an aggregate"}
	}
	e.stableIdentity = true
	return e, nil
}

type ContainerStorageValues interface {
	ContainerStorageType(
		Context,
		ast.Node,
		types.Type,
	) (TypeEmission, error)
	ContainerStorageZero(
		Context,
		ast.Node,
		types.Type,
	) (ExpressionEmission, error)
	ToContainerStorage(
		Context,
		ast.Node,
		types.Type,
		ExpressionEmission,
	) (ExpressionEmission, error)
	FromContainerStorage(
		Context,
		ast.Node,
		types.Type,
		ExpressionEmission,
	) (ExpressionEmission, error)
}

func NewCanonicalStorageAccessorStoreTargetEmission(
	receiver ExpressionEmission,
	getter string,
	setter string,
	arguments []ExpressionEmission,
	sourceType types.Type,
) (StoreTargetEmission, error) {
	target, err := NewAccessorStoreTargetEmission(
		receiver,
		getter,
		setter,
		arguments,
		sourceType,
	)
	if err != nil {
		return StoreTargetEmission{}, err
	}
	target.storage = StoreTargetStorageCanonical
	return target, nil
}

func NewContainerStorageAccessorStoreTargetEmission(
	receiver ExpressionEmission,
	getter string,
	setter string,
	arguments []ExpressionEmission,
	sourceType types.Type,
) (StoreTargetEmission, error) {
	target, err := NewAccessorStoreTargetEmission(
		receiver,
		getter,
		setter,
		arguments,
		sourceType,
	)
	if err != nil {
		return StoreTargetEmission{}, err
	}
	target.storage = StoreTargetStorageContainer
	return target, nil
}

func NewAccessorStoreTargetEmission(
	receiver ExpressionEmission,
	getter string,
	setter string,
	arguments []ExpressionEmission,
	sourceType types.Type,
) (StoreTargetEmission, error) {
	switch {
	case receiver.Value() == nil:
		return StoreTargetEmission{}, &ResultError{
			Result: "accessor store target",
			Reason: "target receiver is nil",
		}
	case getter == "":
		return StoreTargetEmission{}, &ResultError{
			Result: "accessor store target",
			Reason: "getter member is empty",
		}
	case setter == "":
		return StoreTargetEmission{}, &ResultError{
			Result: "accessor store target",
			Reason: "setter member is empty",
		}
	case sourceType == nil:
		return StoreTargetEmission{}, &ResultError{
			Result: "accessor store target",
			Reason: "source type is nil",
		}
	}
	for _, argument := range arguments {
		if argument.Value() == nil {
			return StoreTargetEmission{}, &ResultError{
				Result: "accessor store target",
				Reason: "accessor argument is nil",
			}
		}
	}
	return StoreTargetEmission{
		accessor:          true,
		accessorReceiver:  receiver,
		getterMember:      getter,
		setterMember:      setter,
		accessorArguments: slices.Clone(arguments),
		sourceType:        sourceType,
	}, nil
}

func NewCopyingAccessorStoreTargetEmission(
	receiver ExpressionEmission,
	getter string,
	setter string,
	arguments []ExpressionEmission,
	sourceType types.Type,
) (StoreTargetEmission, error) {
	target, err := NewAccessorStoreTargetEmission(
		receiver,
		getter,
		setter,
		arguments,
		sourceType,
	)
	if err != nil {
		return StoreTargetEmission{}, err
	}
	target.copiesValue = true
	return target, nil
}

func NewFunctionStoreTargetEmission(
	getter ExpressionEmission,
	setter ExpressionEmission,
	arguments []ExpressionEmission,
	sourceType types.Type,
) (StoreTargetEmission, error) {
	switch {
	case getter.Value() == nil:
		return StoreTargetEmission{}, &ResultError{
			Result: "function store target",
			Reason: "getter function is nil",
		}
	case setter.Value() == nil:
		return StoreTargetEmission{}, &ResultError{
			Result: "function store target",
			Reason: "setter function is nil",
		}
	case len(getter.Before()) != 0 || len(setter.Before()) != 0:
		return StoreTargetEmission{}, &ResultError{
			Result: "function store target",
			Reason: "accessor function has evaluation prerequisites",
		}
	case sourceType == nil:
		return StoreTargetEmission{}, &ResultError{
			Result: "function store target",
			Reason: "source type is nil",
		}
	}
	for _, argument := range arguments {
		if argument.Value() == nil {
			return StoreTargetEmission{}, &ResultError{
				Result: "function store target",
				Reason: "accessor argument is nil",
			}
		}
	}
	return StoreTargetEmission{
		accessor:          true,
		accessorFunction:  true,
		getterFunction:    getter,
		setterFunction:    setter,
		accessorArguments: slices.Clone(arguments),
		sourceType:        sourceType,
	}, nil
}

func (e StoreTargetEmission) ReadValue(
	context Context,
	source ast.Node,
) (ExpressionEmission, error) {
	var value ExpressionEmission
	if e.accessor {
		target, err := e.accessorRead(context)
		if err != nil {
			return ExpressionEmission{}, err
		}
		value = target
	} else {
		var err error
		value, err = NewExpressionEmission(
			e.Before(),
			e.value,
			e.Requests(),
		)
		if err != nil {
			return ExpressionEmission{}, err
		}
	}
	switch e.storage {
	case StoreTargetStorageLogical:
		if !e.stableIdentity {
			return value, nil
		}
		return context.Values().Transfer(
			context,
			source,
			e.sourceType,
			e.sourceType,
			ValueTransferCopy,
			value,
		)
	case StoreTargetStorageCanonical:
		return context.Values().FromStorage(
			context,
			source,
			e.sourceType,
			value,
		)
	case StoreTargetStorageContainer:
		return context.ContainerStorage().FromContainerStorage(
			context,
			source,
			e.sourceType,
			value,
		)
	}
	return ExpressionEmission{}, &ResultError{
		Result: "store target read",
		Reason: "storage disposition is invalid",
	}
}

func (e StoreTargetEmission) MutableValue(
	context Context,
	source ast.Node,
) (ExpressionEmission, error) {
	var value ExpressionEmission
	if e.accessor {
		target, err := e.accessorRead(context)
		if err != nil {
			return ExpressionEmission{}, err
		}
		value = target
	} else {
		var err error
		value, err = NewExpressionEmission(
			e.Before(),
			e.value,
			e.Requests(),
		)
		if err != nil {
			return ExpressionEmission{}, err
		}
	}
	switch e.storage {
	case StoreTargetStorageLogical:
		return value, nil
	case StoreTargetStorageCanonical:
		return context.Values().FromStorage(
			context,
			source,
			e.sourceType,
			value,
		)
	case StoreTargetStorageContainer:
		return context.ContainerStorage().FromContainerStorage(
			context,
			source,
			e.sourceType,
			value,
		)
	}
	return ExpressionEmission{}, &ResultError{
		Result: "mutable store target",
		Reason: "storage disposition is invalid",
	}
}

func (e StoreTargetEmission) storeGenericValue(context Context, source ast.Node, value ExpressionEmission) (ExpressionEmission, error) {
	values := context.StableAssignments()
	if values == nil {
		return ExpressionEmission{}, &ResultError{Result: "generic store", Reason: "stable-assignment service is unavailable"}
	}
	captured := e
	before := e.Before()
	requests := e.Requests()
	captured.before = nil
	captured.requests = nil
	if !e.locationCaptured {
		var err error
		captured, before, requests, err = e.PrepareLocation(context)
		if err != nil {
			return ExpressionEmission{}, err
		}
	}
	current, err := captured.MutableValue(context, source)
	if err != nil {
		return ExpressionEmission{}, err
	}
	updated, err := values.AssignStable(context, source, e.sourceType, current.Value(), value)
	if err != nil {
		return ExpressionEmission{}, err
	}
	stored, err := captured.storeConcreteValue(context, source, updated)
	if err != nil {
		return ExpressionEmission{}, err
	}
	before = append(before, current.Before()...)
	before = append(before, stored.Before()...)
	return NewExpressionEmission(before, stored.Value(), CombineRequests(requests, current.Requests(), stored.Requests()))
}

func (e StoreTargetEmission) StoreValue(
	context Context,
	source ast.Node,
	value ExpressionEmission,
) (ExpressionEmission, error) {
	if _, generic := GenericTypeParameter(e.sourceType); generic && !e.copiesValue {
		return e.storeGenericValue(context, source, value)
	}
	return e.storeConcreteValue(context, source, value)
}

func (e StoreTargetEmission) storeConcreteValue(context Context, source ast.Node, value ExpressionEmission) (ExpressionEmission, error) {
	if e.stableIdentity {
		values := context.StableAssignments()
		if values == nil {
			return ExpressionEmission{}, &ResultError{Result: "stable-identity store", Reason: "stable-assignment service is unavailable"}
		}
		location, err := e.MutableValue(context, source)
		if err != nil {
			return ExpressionEmission{}, err
		}
		assigned, err := values.AssignStable(context, source, e.sourceType, location.Value(), value)
		if err != nil {
			return ExpressionEmission{}, err
		}
		return NewExpressionEmission(append(location.Before(), assigned.Before()...), assigned.Value(), CombineRequests(location.Requests(), assigned.Requests()))
	}
	switch e.storage {
	case StoreTargetStorageCanonical:
		var err error
		value, err = context.Values().ToStorage(
			context,
			source,
			e.sourceType,
			value,
		)
		if err != nil {
			return ExpressionEmission{}, err
		}
	case StoreTargetStorageContainer:
		var err error
		value, err = context.ContainerStorage().ToContainerStorage(
			context,
			source,
			e.sourceType,
			value,
		)
		if err != nil {
			return ExpressionEmission{}, err
		}
	}
	if e.accessor {
		return e.AccessorStore(context, value)
	}
	if e.storage != StoreTargetStorageLogical {
		return NewExpressionEmission(
			append(
				e.Before(),
				value.Before()...,
			),
			context.Factory().BinaryExpression(
				nil,
				e.value,
				nil,
				context.Factory().BinaryOperatorToken(
					tsgo.BinaryOperatorEqualsToken,
				),
				value.Value(),
			),
			CombineRequests(e.Requests(), value.Requests()),
		)
	}
	assigned, err := context.Values().Assign(
		context,
		source,
		e.sourceType,
		e.value,
		value,
	)
	if err != nil {
		return ExpressionEmission{}, err
	}
	return NewExpressionEmission(
		append(e.Before(), assigned.Before()...),
		assigned.Value(),
		CombineRequests(e.Requests(), assigned.Requests()),
	)
}

type GenericOperation = genericoperation.GenericOperation

const (
	GenericOperationInvalid                  = genericoperation.GenericOperationInvalid
	GenericOperationZero                     = genericoperation.GenericOperationZero
	GenericOperationCopy                     = genericoperation.GenericOperationCopy
	GenericOperationEqual                    = genericoperation.GenericOperationEqual
	GenericOperationHash                     = genericoperation.GenericOperationHash
	GenericOperationUnaryPlus                = genericoperation.GenericOperationUnaryPlus
	GenericOperationUnaryMinus               = genericoperation.GenericOperationUnaryMinus
	GenericOperationUnaryNot                 = genericoperation.GenericOperationUnaryNot
	GenericOperationUnaryXor                 = genericoperation.GenericOperationUnaryXor
	GenericOperationBinaryAdd                = genericoperation.GenericOperationBinaryAdd
	GenericOperationBinarySubtract           = genericoperation.GenericOperationBinarySubtract
	GenericOperationBinaryMultiply           = genericoperation.GenericOperationBinaryMultiply
	GenericOperationBinaryDivide             = genericoperation.GenericOperationBinaryDivide
	GenericOperationBinaryRemainder          = genericoperation.GenericOperationBinaryRemainder
	GenericOperationBinaryAnd                = genericoperation.GenericOperationBinaryAnd
	GenericOperationBinaryOr                 = genericoperation.GenericOperationBinaryOr
	GenericOperationBinaryXor                = genericoperation.GenericOperationBinaryXor
	GenericOperationBinaryAndNot             = genericoperation.GenericOperationBinaryAndNot
	GenericOperationBinaryShiftLeft          = genericoperation.GenericOperationBinaryShiftLeft
	GenericOperationBinaryShiftRight         = genericoperation.GenericOperationBinaryShiftRight
	GenericOperationBinaryEqual              = genericoperation.GenericOperationBinaryEqual
	GenericOperationBinaryNotEqual           = genericoperation.GenericOperationBinaryNotEqual
	GenericOperationBinaryLess               = genericoperation.GenericOperationBinaryLess
	GenericOperationBinaryLessEqual          = genericoperation.GenericOperationBinaryLessEqual
	GenericOperationBinaryGreater            = genericoperation.GenericOperationBinaryGreater
	GenericOperationBinaryGreaterEqual       = genericoperation.GenericOperationBinaryGreaterEqual
	GenericOperationLength                   = genericoperation.GenericOperationLength
	GenericOperationCapacity                 = genericoperation.GenericOperationCapacity
	GenericOperationConvert                  = genericoperation.GenericOperationConvert
	GenericOperationIndex                    = genericoperation.GenericOperationIndex
	GenericOperationConstraintMethod         = genericoperation.GenericOperationConstraintMethod
	GenericOperationMapConstruct             = genericoperation.GenericOperationMapConstruct
	GenericOperationInterfaceAdapt           = genericoperation.GenericOperationInterfaceAdapt
	GenericOperationInterfaceAssert          = genericoperation.GenericOperationInterfaceAssert
	GenericOperationInterfaceAssertOK        = genericoperation.GenericOperationInterfaceAssertOK
	GenericOperationClear                    = genericoperation.GenericOperationClear
	GenericOperationNilEqual                 = genericoperation.GenericOperationNilEqual
	GenericOperationToStorage                = genericoperation.GenericOperationToStorage
	GenericOperationFromStorage              = genericoperation.GenericOperationFromStorage
	GenericOperationToContainerStorage       = genericoperation.GenericOperationToContainerStorage
	GenericOperationFromContainerStorage     = genericoperation.GenericOperationFromContainerStorage
	GenericOperationIndexAddress             = genericoperation.GenericOperationIndexAddress
	GenericOperationSlice                    = genericoperation.GenericOperationSlice
	GenericOperationSliceFull                = genericoperation.GenericOperationSliceFull
	GenericOperationDeferredCallableRegistry = genericoperation.GenericOperationDeferredCallableRegistry
	GenericOperationAppendSpread             = genericoperation.GenericOperationAppendSpread
	GenericOperationReflectionType           = genericoperation.GenericOperationReflectionType
	GenericOperationReflectionValue          = genericoperation.GenericOperationReflectionValue
	GenericOperationAssign                   = genericoperation.GenericOperationAssign
)

type GenericOperationSelection = genericoperation.GenericOperationSelection

type GenericStorageDirection = genericoperation.GenericStorageDirection

const (
	GenericStorageDirectionInvalid = genericoperation.GenericStorageDirectionInvalid
	GenericStorageDirectionTo      = genericoperation.GenericStorageDirectionTo
	GenericStorageDirectionFrom    = genericoperation.GenericStorageDirectionFrom
)

type GenericRepresentationFacet = genericoperation.GenericRepresentationFacet

const (
	GenericRepresentationInvalid          = genericoperation.GenericRepresentationInvalid
	GenericRepresentationStorage          = genericoperation.GenericRepresentationStorage
	GenericRepresentationPointer          = genericoperation.GenericRepresentationPointer
	GenericRepresentationContainerStorage = genericoperation.GenericRepresentationContainerStorage
)

func GenericTypeParameter(sourceType types.Type) (*types.TypeParam, bool) {
	return genericoperation.GenericTypeParameter(sourceType)
}

func UnaryGenericOperation(source token.Token) (GenericOperation, bool) {
	return genericoperation.UnaryGenericOperation(source)
}

func BinaryGenericOperation(source token.Token) (GenericOperation, bool) {
	return genericoperation.BinaryGenericOperation(source)
}

func SelectGenericOperation(
	operation GenericOperation,
) (GenericOperationSelection, error) {
	selection, err := genericoperation.SelectGenericOperation(operation)
	if selectionError, ok := err.(*genericoperation.Error); ok {
		return GenericOperationSelection{}, &InvariantError{
			Role:   RoleCallCallee,
			Reason: selectionError.Reason,
		}
	}
	return selection, err
}

func SelectGenericConstraintMethod(
	method *types.Func,
) (GenericOperationSelection, error) {
	selection, err := genericoperation.SelectGenericConstraintMethod(method)
	if selectionError, ok := err.(*genericoperation.Error); ok {
		return GenericOperationSelection{}, &InvariantError{
			Role:   RoleCallCallee,
			Reason: selectionError.Reason,
		}
	}
	return selection, err
}

func GenericStorageOperationType(
	selection GenericOperationSelection,
	signature *types.Signature,
) (
	types.Type,
	GenericRepresentationFacet,
	GenericStorageDirection,
	bool,
) {
	return genericoperation.GenericStorageOperationType(selection, signature)
}

func GenericIndexAddressOperation(
	selection GenericOperationSelection,
	signature *types.Signature,
) (types.Type, types.Type, types.Type, bool) {
	return genericoperation.GenericIndexAddressOperation(selection, signature)
}

func GenericRepresentationFacetOrder() []GenericRepresentationFacet {
	return genericoperation.GenericRepresentationFacetOrder()
}
