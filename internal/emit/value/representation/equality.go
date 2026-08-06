package representation

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	genericoperation "github.com/tsoniclang/gotots/internal/emit/generic/operation"
	pointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	interfacetype "github.com/tsoniclang/gotots/internal/emit/type/interfacevalue"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
	arrayvalue "github.com/tsoniclang/gotots/internal/emit/value/array"
	complexvalue "github.com/tsoniclang/gotots/internal/emit/value/complex"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (owner Owner) Equal(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	left tsgo.Expression,
	right tsgo.Expression,
) (api.ExpressionEmission, error) {
	if api.ContainsGenericTypeParameter(sourceType) {
		return genericoperation.Call(
			context,
			source,
			api.GenericOperationEqual,
			[]types.Type{sourceType, sourceType},
			[]types.Type{types.Typ[types.Bool]},
			[]api.ExpressionEmission{
				api.DirectExpression(left),
				api.DirectExpression(right),
			},
		)
	}
	if _, ok := interfacetype.Resolve(sourceType); ok {
		reference, err := context.Names().Runtime(
			api.RuntimeInterfaceEqual,
			api.ImportPhaseValue,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return api.DirectExpression(
			context.Factory().CallExpression(
				reference.Expression(context.Factory()),
				nil,
				nil,
				[]tsgo.Expression{left, right},
				tsgo.NodeFlagsNone,
			),
			reference.Requests()...,
		), nil
	}
	if panicNilRuntimeValue(context, sourceType) {
		return panicNilEqual(context), nil
	}
	if unsafePointerValue(sourceType) {
		return api.DirectExpression(context.Factory().BinaryExpression(
			nil,
			left,
			nil,
			context.Factory().BinaryOperatorToken(
				tsgo.BinaryOperatorEqualsEqualsEqualsToken,
			),
			right,
		)), nil
	}
	if defined, ok := definedtype.Resolve(sourceType); ok {
		operationContext, err := defined.OperationContext(context)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		leftValue, err := defined.Project(
			context.WithRole(api.RoleDefinedValue),
			api.DirectExpression(left),
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		rightValue, err := defined.Project(
			context.WithRole(api.RoleDefinedValue),
			api.DirectExpression(right),
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		result, err := (Owner{}).Equal(
			operationContext.WithRole(api.RoleDefinedValue),
			source,
			defined.Underlying(),
			leftValue.Value(),
			rightValue.Value(),
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		before := append(leftValue.Before(), rightValue.Before()...)
		before = append(before, result.Before()...)
		return api.NewExpressionEmission(
			before,
			result.Value(),
			api.CombineRequests(
				leftValue.Requests(),
				rightValue.Requests(),
				result.Requests(),
			),
		)
	}
	if carrier, ok := complexvalue.Describe(sourceType); ok {
		symbol, valid := complexvalue.EqualSymbol(carrier)
		if !valid {
			return api.ExpressionEmission{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "complex equality has no runtime symbol",
			}
		}
		return complexvalue.Call(
			context,
			symbol,
			[]tsgo.Expression{left, right},
		)
	}
	if array, ok := arrayvalue.Resolve(context, sourceType); ok {
		return array.Equal(context, source, left, right)
	}
	if structType, ok := isAnonymousStruct(sourceType); ok {
		return anonymousStructEqual(context, source, structType, left, right)
	}
	if _, ok := primitive(context, sourceType); ok {
		return api.DirectExpression(context.Factory().BinaryExpression(
			nil,
			left,
			nil,
			context.Factory().BinaryOperatorToken(
				tsgo.BinaryOperatorEqualsEqualsEqualsToken,
			),
			right,
		)), nil
	}
	if pointerValue(sourceType) {
		pointer, _, _ := pointertype.Resolve(sourceType)
		representation, err := owner.PointerRepresentation(
			context,
			pointer,
			api.PointerRepresentationDemandNone,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		if representation.Representation().DirectClass() {
			return owner.directClassPointerEqual(
				context,
				source,
				pointer.Elem(),
				left,
				right,
				representation.UsesStorageIdentity(),
				representation.Requests(),
			)
		}
		reference, err := context.Names().Runtime(
			api.RuntimePointer,
			api.ImportPhaseValue,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return api.DirectExpression(
			context.Factory().CallExpression(
				context.Factory().PropertyAccessExpression(
					reference.Expression(context.Factory()),
					nil,
					context.Factory().Identifier(pointerruntime.EqualName),
					tsgo.NodeFlagsNone,
				),
				nil,
				nil,
				[]tsgo.Expression{left, right},
				tsgo.NodeFlagsNone,
			),
			api.CombineRequests(
				reference.Requests(),
				representation.Requests(),
			)...,
		), nil
	}
	if channelValue(sourceType) {
		return api.DirectExpression(
			context.Factory().BinaryExpression(
				nil,
				left,
				nil,
				context.Factory().BinaryOperatorToken(
					tsgo.BinaryOperatorEqualsEqualsEqualsToken,
				),
				right,
			),
		), nil
	}
	if callableValue(sourceType) {
		return api.DirectExpression(
			context.Factory().BinaryExpression(
				nil,
				left,
				nil,
				context.Factory().BinaryOperatorToken(
					tsgo.BinaryOperatorEqualsEqualsEqualsToken,
				),
				right,
			),
		), nil
	}
	_, _, ok := namedStruct(sourceType)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	return owner.namedStructOperation(
		context,
		source,
		sourceType,
		api.NamedStructOperationEqual,
		[]tsgo.Expression{left, right},
	)
}

func (owner Owner) directClassPointerEqual(
	context api.Context,
	source ast.Node,
	element types.Type,
	left tsgo.Expression,
	right tsgo.Expression,
	requiresStorageIdentity bool,
	requests []api.RootRequest,
) (api.ExpressionEmission, error) {
	usesStorageIdentity := false
	if requiresStorageIdentity {
		var err error
		usesStorageIdentity, err = owner.directPointerUsesStorageIdentity(
			context,
			element,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	if !usesStorageIdentity {
		return api.DirectExpression(
			context.Factory().BinaryExpression(
				nil,
				left,
				nil,
				context.Factory().BinaryOperatorToken(
					tsgo.BinaryOperatorEqualsEqualsEqualsToken,
				),
				right,
			),
			requests...,
		), nil
	}
	before := make([]tsgo.Statement, 0, 2)
	var err error
	left, before, err = captureEqualityValue(context, left, before)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	right, before, err = captureEqualityValue(context, right, before)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	leftStorage, err := owner.ToStorage(
		context.WithRole(api.RoleBinaryLeft),
		source,
		element,
		api.DirectExpression(left),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	rightStorage, err := owner.ToStorage(
		context.WithRole(api.RoleBinaryRight),
		source,
		element,
		api.DirectExpression(right),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if len(leftStorage.Before()) != 0 || len(rightStorage.Before()) != 0 {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "direct pointer equality produced deferred storage work",
		}
	}
	undefined := context.Factory().Identifier("undefined")
	strict := func(left tsgo.Expression, operator tsgo.BinaryOperator, right tsgo.Expression) tsgo.Expression {
		return context.Factory().BinaryExpression(
			nil,
			left,
			nil,
			context.Factory().BinaryOperatorToken(operator),
			right,
		)
	}
	identity := strict(left, tsgo.BinaryOperatorEqualsEqualsEqualsToken, right)
	sameStorage := strict(
		leftStorage.Value(),
		tsgo.BinaryOperatorEqualsEqualsEqualsToken,
		rightStorage.Value(),
	)
	nonNil := strict(left, tsgo.BinaryOperatorExclamationEqualsEqualsToken, undefined)
	nonNil = strict(
		nonNil,
		tsgo.BinaryOperatorAmpersandAmpersandToken,
		strict(right, tsgo.BinaryOperatorExclamationEqualsEqualsToken, undefined),
	)
	nonNil = strict(
		nonNil,
		tsgo.BinaryOperatorAmpersandAmpersandToken,
		sameStorage,
	)
	return api.NewExpressionEmission(
		before,
		strict(identity, tsgo.BinaryOperatorBarBarToken, nonNil),
		api.CombineRequests(
			leftStorage.Requests(),
			rightStorage.Requests(),
			requests,
		),
	)
}

func captureEqualityValue(
	context api.Context,
	value tsgo.Expression,
	before []tsgo.Statement,
) (tsgo.Expression, []tsgo.Statement, error) {
	if value.Kind() == tsgo.SyntaxKindIdentifier {
		return value, before, nil
	}
	name, err := context.Names().Temporary(api.TemporaryEqualityOperand)
	if err != nil {
		return nil, nil, err
	}
	before = append(before, context.Factory().VariableStatement(
		nil,
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{context.Factory().VariableDeclaration(
				context.Factory().Identifier(name),
				nil,
				nil,
				value,
			)},
			tsgo.NodeFlagsConst,
		),
	))
	return context.Factory().Identifier(name), before, nil
}
