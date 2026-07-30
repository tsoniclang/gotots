package api

import (
	"go/ast"
	"go/types"
	"slices"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
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

type ContainerStorageValues interface {
	ContainerStorageType(
		Context,
		ast.Node,
		types.Type,
	) (TypeEmission, error)
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
	PointerStorageType(
		Context,
		ast.Node,
		types.Type,
		PointerRepresentationObservation,
	) (TypeEmission, error)
	ToPointerStorage(
		Context,
		ast.Node,
		types.Type,
		PointerRepresentationObservation,
		ExpressionEmission,
	) (ExpressionEmission, error)
	FromPointerStorage(
		Context,
		ast.Node,
		types.Type,
		PointerRepresentationObservation,
		ExpressionEmission,
	) (ExpressionEmission, error)
}

func NewStableIdentityStoreTargetEmission(
	location ExpressionEmission,
	sourceType types.Type,
) (StoreTargetEmission, error) {
	if location.Value() == nil || sourceType == nil {
		return StoreTargetEmission{}, &ResultError{
			Result: "stable-identity store target",
			Reason: "location or source type is invalid",
		}
	}
	return StoreTargetEmission{
		before:         location.Before(),
		value:          location.Value(),
		stableIdentity: true,
		sourceType:     sourceType,
		requests:       location.Requests(),
	}, nil
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

func (e StoreTargetEmission) StoreValue(
	context Context,
	source ast.Node,
	value ExpressionEmission,
) (ExpressionEmission, error) {
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
	if e.stableIdentity {
		values := context.StableAssignments()
		if values == nil {
			return ExpressionEmission{}, &ResultError{
				Result: "stable-identity store",
				Reason: "stable-assignment service is unavailable",
			}
		}
		assigned, err := values.AssignStable(
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
