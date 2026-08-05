package reflectiontype

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

// locationScaffold carries the shared references every generated location
// callback of one walked type needs: the exact adapter guard, the runtime
// box type, the panic owner, and the descriptor thunk return type.
type locationScaffold struct {
	factory    tsgo.Factory
	adapter    api.NameReference
	boxType    api.NameReference
	panicRef   api.NameReference
	targetType api.NameReference
	requests   []api.RootRequest
}

// extendedValueProperties derives the location-model callbacks admitted by
// one type's kind beyond the scalar projections: zero evidence and string
// boxing for basic scalars, aliasing element navigation for pointers to
// directly represented structs, and field navigation for structs. A struct
// participating in the value model with a field kind outside the supported
// scalar set fails at emission, never silently at runtime.
func extendedValueProperties(
	context api.Context,
	names api.ReflectionNames,
	reflectionType *types.TypeName,
	sourceType types.Type,
	scaffold *locationScaffold,
) ([]tsgo.ObjectLiteralElementLike, error) {
	switch selected := types.Unalias(sourceType).Underlying().(type) {
	case *types.Basic:
		return basicValueProperties(context, scaffold, selected)
	case *types.Pointer:
		switch types.Unalias(selected.Elem()).Underlying().(type) {
		case *types.Struct:
			return pointerValueProperties(
				context,
				names,
				reflectionType,
				selected.Elem(),
				scaffold,
			)
		case *types.Slice, *types.Basic, *types.Map:
			return pointerCellValueProperties(
				context,
				names,
				reflectionType,
				selected.Elem(),
				scaffold,
			)
		default:
			return nil, nil
		}
	case *types.Struct:
		return structValueProperties(
			context,
			names,
			reflectionType,
			selected,
			scaffold,
		)
	case *types.Slice:
		return sliceValueProperties(
			context,
			names,
			reflectionType,
			selected,
			scaffold,
		)
	case *types.Map:
		return mapValueProperties(
			context,
			names,
			reflectionType,
			selected,
			scaffold,
		)
	default:
		return nil, nil
	}
}

// supportedValueStruct admits one struct into the value location model
// exactly when every field is a plain basic scalar. Aggregate field kinds
// join with their own construct slices; until then the demand fails closed
// at emission.
func supportedValueStruct(structLike types.Type) (*types.Struct, error) {
	structType, ok := types.Unalias(structLike).Underlying().(*types.Struct)
	if !ok {
		return nil, &api.GeneratedArtifactShapeError{
			Artifact: structLike.String(),
			Reason:   "reflection value location target is not a struct",
		}
	}
	for index := range structType.NumFields() {
		fieldType := structType.Field(index).Type()
		basic, basicOK := types.Unalias(fieldType).(*types.Basic)
		if !basicOK || basic.Info()&(types.IsBoolean|types.IsString|
			types.IsInteger|types.IsFloat) == 0 {
			return nil, &api.GeneratedArtifactShapeError{
				Artifact: structType.String(),
				Reason: "reflection value struct field type " +
					fieldType.String() +
					" is outside the supported scalar location model",
			}
		}
	}
	return structType, nil
}

// locationCallbacks is the exact content of one generated location
// literal: the descriptor thunk target, generated settability evidence,
// the boxing read expression, and the write body.
type locationCallbacks struct {
	descriptor api.NameReference
	settable   bool
	get        tsgo.Expression
	set        tsgo.Block
}

func locationLiteral(
	scaffold *locationScaffold,
	callbacks locationCallbacks,
) tsgo.ObjectLiteralExpression {
	factory := scaffold.factory
	return factory.ObjectLiteralExpression(
		[]tsgo.ObjectLiteralElementLike{
			expressionProperty(factory, "type", arrow(
				factory,
				scaffold.targetType,
				callbacks.descriptor.Expression(factory),
			)),
			booleanProperty(factory, "settable", callbacks.settable),
			expressionProperty(factory, "get", factory.ArrowFunction(
				nil,
				nil,
				nil,
				factory.TypeReferenceNode(
					scaffold.boxType.EntityName(factory),
					nil,
				),
				factory.EqualsGreaterThanToken(),
				factory.ParenthesizedExpression(callbacks.get),
			)),
			expressionProperty(factory, "set", factory.ArrowFunction(
				nil,
				nil,
				[]tsgo.ParameterDeclaration{factory.ParameterDeclaration(
					nil,
					nil,
					factory.Identifier("value"),
					nil,
					factory.TypeReferenceNode(
						scaffold.boxType.EntityName(factory),
						nil,
					),
					nil,
				)},
				factory.KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindVoidKeyword,
				),
				factory.EqualsGreaterThanToken(),
				callbacks.set,
			)),
		},
		true,
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
