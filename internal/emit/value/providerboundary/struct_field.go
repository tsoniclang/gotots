package providerboundary

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func StructField(
	context api.Context,
	ownerType types.Type,
	field *types.Var,
) (gostdlib.ProviderStructField, bool, error) {
	owner, ok := types.Unalias(ownerType).(*types.Named)
	if !ok || owner.Obj() == nil {
		return gostdlib.ProviderStructField{}, false, nil
	}
	return context.Names().ProviderStructField(owner.Origin().Obj(), field)
}

func ReadStructField(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	ownerType types.Type,
	field *types.Var,
	receiver api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	selected, providerOwned, err := StructField(context, ownerType, field)
	if err != nil || !providerOwned {
		return api.ExpressionEmission{}, providerOwned, err
	}
	raw, err := api.NewExpressionEmission(
		receiver.Before(),
		context.Factory().PropertyAccessExpression(
			receiver.Value(),
			nil,
			context.Factory().Identifier(selected.Member()),
			tsgo.NodeFlagsNone,
		),
		receiver.Requests(),
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	converted, _, err := FromProviderValue(
		context.WithRole(api.RoleStructField),
		children,
		nil,
		"",
		field.Type(),
		raw,
	)
	return converted, true, err
}

func ToProviderStructField(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	ownerType types.Type,
	field *types.Var,
	value api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	_, providerOwned, err := StructField(context, ownerType, field)
	if err != nil || !providerOwned {
		return value, providerOwned, err
	}
	converted, _, err := ToProviderValue(
		context.WithRole(api.RoleStructAssignField),
		children,
		nil,
		"",
		field.Type(),
		value,
	)
	return converted, true, err
}

func StructFieldStoreTarget(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	ownerType types.Type,
	field *types.Var,
	receiver api.ExpressionEmission,
) (api.StoreTargetEmission, bool, error) {
	selected, providerOwned, err := StructField(context, ownerType, field)
	if err != nil || !providerOwned {
		return api.StoreTargetEmission{}, providerOwned, err
	}
	ownerName := "$providerOwner"
	productName := "$productValue"
	raw := api.DirectExpression(context.Factory().PropertyAccessExpression(
		context.Factory().Identifier(ownerName),
		nil,
		context.Factory().Identifier(selected.Member()),
		tsgo.NodeFlagsNone,
	))
	read, readChanged, err := FromProviderValue(
		context.WithRole(api.RoleStructField),
		children,
		nil,
		"",
		field.Type(),
		raw,
	)
	if err != nil {
		return api.StoreTargetEmission{}, true, err
	}
	write, writeChanged, err := ToProviderValue(
		context.WithRole(api.RoleStructAssignField),
		children,
		nil,
		"",
		field.Type(),
		api.DirectExpression(context.Factory().Identifier(productName)),
	)
	if err != nil {
		return api.StoreTargetEmission{}, true, err
	}
	if !readChanged && !writeChanged {
		target, directErr := api.NewPropertyStoreTargetEmission(
			context.Factory(),
			receiver,
			selected.Member(),
			field.Type(),
		)
		return target, true, directErr
	}
	providerContext, err := providerRepresentationContext(context, nil)
	if err != nil {
		return api.StoreTargetEmission{}, true, err
	}
	productField, err := children.RepresentedType(
		context.WithRole(api.RoleStructField),
		source,
		field.Type(),
	)
	if err != nil {
		return api.StoreTargetEmission{}, true, err
	}
	providerOwner, err := children.RepresentedType(
		providerContext.WithRole(api.RoleStructField),
		source,
		ownerType,
	)
	if err != nil {
		return api.StoreTargetEmission{}, true, err
	}
	getter := conversionArrow(
		context,
		ownerName,
		providerOwner.Value(),
		productField.Value(),
		read,
	)
	setterBody := append(write.Before(), context.Factory().ExpressionStatement(
		context.Factory().BinaryExpression(
			nil,
			context.Factory().PropertyAccessExpression(
				context.Factory().Identifier(ownerName),
				nil,
				context.Factory().Identifier(selected.Member()),
				tsgo.NodeFlagsNone,
			),
			nil,
			context.Factory().BinaryOperatorToken(tsgo.BinaryOperatorEqualsToken),
			write.Value(),
		),
	))
	setterBody = append(
		setterBody,
		context.Factory().ReturnStatement(context.Factory().Identifier(productName)),
	)
	setter := context.Factory().ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			context.Factory().ParameterDeclaration(
				nil,
				nil,
				context.Factory().Identifier(ownerName),
				nil,
				providerOwner.Value(),
				nil,
			),
			context.Factory().ParameterDeclaration(
				nil,
				nil,
				context.Factory().Identifier(productName),
				nil,
				productField.Value(),
				nil,
			),
		},
		productField.Value(),
		context.Factory().EqualsGreaterThanToken(),
		context.Factory().Block(setterBody, true),
	)
	requests := api.CombineRequests(
		providerOwner.Requests(),
		productField.Requests(),
		read.Requests(),
		write.Requests(),
	)
	target, err := api.NewFunctionStoreTargetEmission(
		api.DirectExpression(getter, requests...),
		api.DirectExpression(setter, requests...),
		[]api.ExpressionEmission{receiver},
		field.Type(),
	)
	return target, true, err
}

func ProjectStructFieldPointer(
	context api.Context,
	providerContext api.Context,
	children api.ChildEmitter,
	source ast.Node,
	fieldType types.Type,
	pointer api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	fromProvider, fromChanged, err := structFieldStorageConversion(
		context,
		providerContext,
		children,
		source,
		fieldType,
		"$providerField",
		true,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	toProvider, toChanged, err := structFieldStorageConversion(
		context,
		providerContext,
		children,
		source,
		fieldType,
		"$productField",
		false,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	storageChanged := context.Values().RequiresStorageProjection(context, fieldType) ||
		providerContext.Values().RequiresStorageProjection(providerContext, fieldType)
	if !fromChanged && !toChanged && !storageChanged {
		return pointer, nil
	}
	productLogical, err := children.RepresentedType(context, source, fieldType)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	providerLogical, err := children.RepresentedType(
		providerContext,
		source,
		fieldType,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	productStorage, err := context.Values().StorageType(context, source, fieldType)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	providerStorage, err := providerContext.Values().StorageType(
		providerContext,
		source,
		fieldType,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	runtime, err := context.Names().Runtime(
		api.RuntimePointerProjection,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		pointer.Before(),
		context.Factory().CallExpression(
			context.Factory().Identifier(runtime.Name()),
			nil,
			[]tsgo.TypeNode{
				providerLogical.Value(),
				providerStorage.Value(),
				productLogical.Value(),
				productStorage.Value(),
			},
			[]tsgo.Expression{
				pointer.Value(),
				conversionArrow(
					context,
					"$providerField",
					providerStorage.Value(),
					productStorage.Value(),
					fromProvider,
				),
				conversionArrow(
					context,
					"$productField",
					productStorage.Value(),
					providerStorage.Value(),
					toProvider,
				),
			},
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			pointer.Requests(),
			productLogical.Requests(),
			providerLogical.Requests(),
			productStorage.Requests(),
			providerStorage.Requests(),
			fromProvider.Requests(),
			toProvider.Requests(),
			runtime.Requests(),
		),
	)
}

func structFieldStorageConversion(
	context api.Context,
	providerContext api.Context,
	children api.ChildEmitter,
	source ast.Node,
	fieldType types.Type,
	parameter string,
	fromProvider bool,
) (api.ExpressionEmission, bool, error) {
	sourceContext := context
	targetContext := providerContext
	if fromProvider {
		sourceContext = providerContext
		targetContext = context
	}
	logical, err := sourceContext.Values().FromStorage(
		sourceContext,
		source,
		fieldType,
		api.DirectExpression(context.Factory().Identifier(parameter)),
	)
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	changed := false
	if fromProvider {
		logical, changed, err = FromProviderValue(
			context,
			children,
			nil,
			"",
			fieldType,
			logical,
		)
	} else {
		logical, changed, err = ToProviderValue(
			context,
			children,
			nil,
			"",
			fieldType,
			logical,
		)
	}
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	stored, err := targetContext.Values().ToStorage(
		targetContext,
		source,
		fieldType,
		logical,
	)
	return stored, changed, err
}
