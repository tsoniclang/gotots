package reflectiontype

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	pointermarker "github.com/tsoniclang/gotots/internal/emit/marker/pointer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

// locationScaffold carries the shared references every generated location
// callback of one walked type needs: the exact adapter guard, the runtime
// box type, the panic owner, and the descriptor thunk return type.
type locationScaffold struct {
	factory        tsgo.Factory
	adapter        api.NameReference
	boxType        api.NameReference
	panicRef       api.NameReference
	descriptorType api.NameReference
	requests       []api.RootRequest
	// payload is the raw represented payload of the walked type's box:
	// the boxed value routed through the defined-type projection when the
	// registered type carries a branded representation.
	payload tsgo.Expression
}

// scaffoldPayload is the raw payload expression of the walked type.
func scaffoldPayload(scaffold *locationScaffold) tsgo.Expression {
	if scaffold.payload != nil {
		return scaffold.payload
	}
	return boxPayload(scaffold.factory)
}

// extendedValueProperties derives the location-model callbacks admitted by
// one non-struct, non-pointer kind beyond its scalar projections.
func extendedValueProperties(
	context api.Context,
	names api.ReflectionNames,
	reflectionType *types.TypeName,
	sourceType types.Type,
	scaffold *locationScaffold,
) ([]tsgo.ObjectLiteralElementLike, error) {
	switch selected := types.Unalias(sourceType).Underlying().(type) {
	case *types.Basic:
		return basicValueProperties(context, scaffold, sourceType, selected)
	case *types.Slice:
		return sliceValueProperties(
			context,
			names,
			reflectionType,
			sourceType,
			selected,
			scaffold,
		)
	case *types.Map:
		return mapValueProperties(
			context,
			sourceType,
			selected,
			scaffold,
		)
	case *types.Interface:
		return interfaceValueProperties(scaffold), nil
	default:
		return nil, nil
	}
}

// locationCallbacks is the exact content of one generated location
// literal: the descriptor thunk target, generated settability evidence,
// the boxing read expression, and the write body.
type locationCallbacks struct {
	descriptor api.NameReference
	settable   bool
	get        tsgo.Expression
	getBlock   tsgo.Block
	set        tsgo.Block
	address    tsgo.Expression
}

func locationLiteral(
	scaffold *locationScaffold,
	callbacks locationCallbacks,
) tsgo.ObjectLiteralExpression {
	factory := scaffold.factory
	var get tsgo.ConciseBody
	if callbacks.getBlock != nil {
		get = callbacks.getBlock
	} else {
		get = factory.ParenthesizedExpression(callbacks.get)
	}
	properties := []tsgo.ObjectLiteralElementLike{
		expressionProperty(factory, "type", arrow(
			factory,
			scaffold.descriptorType,
			callbacks.descriptor.Expression(factory),
		)),
		booleanProperty(factory, "settable", callbacks.settable),
		expressionProperty(factory, "get", factory.ArrowFunction(
			nil,
			nil,
			nil,
			optionalInterfaceBoxType(factory, scaffold.boxType),
			factory.EqualsGreaterThanToken(),
			get,
		)),
		expressionProperty(factory, "set", factory.ArrowFunction(
			nil,
			nil,
			[]tsgo.ParameterDeclaration{factory.ParameterDeclaration(
				nil,
				nil,
				factory.Identifier("value"),
				nil,
				optionalInterfaceBoxType(factory, scaffold.boxType),
				nil,
			)},
			factory.KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindVoidKeyword,
			),
			factory.EqualsGreaterThanToken(),
			callbacks.set,
		)),
	}
	if callbacks.address != nil {
		properties = append(
			properties,
			expressionProperty(factory, "address", callbacks.address),
		)
	}
	return factory.ObjectLiteralExpression(properties, true)
}

func reflectedStoreTargetAddress(
	context api.Context,
	elementType types.Type,
	target api.StoreTargetEmission,
) (api.ExpressionEmission, error) {
	if target.IsAccessor() {
		return api.ExpressionEmission{}, &api.GeneratedArtifactShapeError{
			Artifact: elementType.String(),
			Reason:   "reflected address target is accessor-backed",
		}
	}
	var typeArguments []api.TypeEmission
	if target.UsesCanonicalStorage() || explicitAddressMarkerType(elementType) {
		storageType, err := context.Values().StorageType(
			context.WithRole(api.RoleStorageType),
			nil,
			elementType,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		typeArguments = []api.TypeEmission{storageType}
	}
	pointer, err := pointermarker.Operation(
		context,
		tsoniccore.SymbolAddressOf,
		typeArguments,
		[]api.ExpressionEmission{api.DirectExpression(
			target.Value(),
			target.Requests()...,
		)},
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	pointer, err = api.NewExpressionEmission(
		append(target.Before(), pointer.Before()...),
		pointer.Value(),
		pointer.Requests(),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if target.UsesCanonicalStorage() {
		pointer, err = context.Values().ProjectStoragePointer(
			context.WithRole(api.RoleStructField),
			nil,
			elementType,
			pointer,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	return boxedReflectionAddress(
		context,
		elementType,
		pointer,
	)
}

func explicitAddressMarkerType(sourceType types.Type) bool {
	switch selected := types.Unalias(sourceType).Underlying().(type) {
	case *types.Pointer, *types.Interface, *types.Signature, *types.Chan:
		return true
	case *types.Basic:
		return selected.Kind() == types.UnsafePointer
	default:
		return false
	}
}

func boxedReflectionAddress(
	context api.Context,
	elementType types.Type,
	pointer api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	pointerType := types.NewPointer(elementType)
	adapter, err := context.Names().InterfaceAdapter(pointerType, nil)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		pointer.Before(),
		context.Factory().NewExpression(
			adapter.Expression(context.Factory()),
			nil,
			[]tsgo.Expression{pointer.Value()},
		),
		api.CombineRequests(
			pointer.Requests(),
			adapter.Requests(),
		),
	)
}

func addressCallback(
	factory tsgo.Factory,
	parameters []tsgo.ParameterDeclaration,
	address api.ExpressionEmission,
) tsgo.ArrowFunction {
	if len(address.Before()) == 0 {
		return factory.ArrowFunction(
			nil,
			nil,
			parameters,
			nil,
			factory.EqualsGreaterThanToken(),
			factory.ParenthesizedExpression(address.Value()),
		)
	}
	statements := append([]tsgo.Statement(nil), address.Before()...)
	statements = append(statements, factory.ReturnStatement(address.Value()))
	return factory.ArrowFunction(
		nil,
		nil,
		parameters,
		nil,
		factory.EqualsGreaterThanToken(),
		factory.Block(statements, true),
	)
}

// guardedProjection wraps one projection of the walked type's payload in
// the exact generated adapter guard, panicking on a foreign box.
func guardedProjection(
	scaffold *locationScaffold,
	operation string,
	projection tsgo.Expression,
) tsgo.Expression {
	return scaffold.factory.ConditionalExpression(
		adapterGuard(scaffold.factory, scaffold.adapter, "box"),
		scaffold.factory.QuestionToken(),
		projection,
		scaffold.factory.ColonToken(),
		runtimePanic(
			scaffold,
			"reflect: "+operation+" received a foreign interface box",
		),
	)
}

// guardedForeignPayload projects the payload of one written box through
// the target member type's exact adapter guard.
func guardedForeignPayload(
	scaffold *locationScaffold,
	adapter api.NameReference,
	operation string,
) tsgo.Expression {
	return guardedForeignOperand(scaffold, adapter, "value", operation)
}

// guardedForeignOperand projects the payload of one named box operand
// through the exact adapter guard of its expected type.
func guardedForeignOperand(
	scaffold *locationScaffold,
	adapter api.NameReference,
	operand string,
	operation string,
) tsgo.Expression {
	factory := scaffold.factory
	return factory.ConditionalExpression(
		adapterGuard(factory, adapter, operand),
		factory.QuestionToken(),
		memberAccess(factory, operand, "$go$value"),
		factory.ColonToken(),
		runtimePanic(
			scaffold,
			"reflect: "+operation+" received a foreign interface box",
		),
	)
}

func foreignBoxGuardStatement(
	scaffold *locationScaffold,
	operation string,
) tsgo.Statement {
	factory := scaffold.factory
	return factory.IfStatement(
		factory.PrefixUnaryExpression(
			tsgo.PrefixUnaryExpressionOperatorKindExclamationToken,
			adapterGuard(factory, scaffold.adapter, "box"),
		),
		factory.Block([]tsgo.Statement{
			factory.ReturnStatement(runtimePanic(
				scaffold,
				"reflect: "+operation+
					" received a foreign interface box",
			)),
		}, true),
		nil,
	)
}

func runtimePanic(
	scaffold *locationScaffold,
	message string,
) tsgo.Expression {
	factory := scaffold.factory
	return factory.CallExpression(
		factory.PropertyAccessExpression(
			scaffold.panicRef.Expression(factory),
			nil,
			factory.Identifier("raiseRuntime"),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		[]tsgo.Expression{factory.StringLiteral(
			message,
			tsgo.TokenFlagsNone,
		)},
		tsgo.NodeFlagsNone,
	)
}

func adapterGuard(
	factory tsgo.Factory,
	adapter api.NameReference,
	operand string,
) tsgo.Expression {
	return factory.CallExpression(
		factory.PropertyAccessExpression(
			adapter.Expression(factory),
			nil,
			factory.Identifier("$is"),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		[]tsgo.Expression{factory.Identifier(operand)},
		tsgo.NodeFlagsNone,
	)
}

func boxPayload(factory tsgo.Factory) tsgo.Expression {
	return memberAccess(factory, "box", "$go$value")
}

func memberAccess(
	factory tsgo.Factory,
	target string,
	member string,
) tsgo.Expression {
	return factory.PropertyAccessExpression(
		factory.Identifier(target),
		nil,
		factory.Identifier(member),
		tsgo.NodeFlagsNone,
	)
}

func boxParameter(scaffold *locationScaffold) tsgo.ParameterDeclaration {
	return scaffold.factory.ParameterDeclaration(
		nil,
		nil,
		scaffold.factory.Identifier("box"),
		nil,
		scaffold.factory.TypeReferenceNode(
			scaffold.boxType.EntityName(scaffold.factory),
			nil,
		),
		nil,
	)
}

func constStatement(
	factory tsgo.Factory,
	name string,
	initializer tsgo.Expression,
) tsgo.Statement {
	return factory.VariableStatement(
		nil,
		factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				factory.VariableDeclaration(
					factory.Identifier(name),
					nil,
					nil,
					initializer,
				),
			},
			tsgo.NodeFlagsConst,
		),
	)
}
