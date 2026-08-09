package providerboundary

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	pointermarker "github.com/tsoniclang/gotots/internal/emit/marker/pointer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func fromProviderPointer(
	context api.Context,
	children api.ChildEmitter,
	owner *types.Named,
	ownerBridge string,
	profile []gostdlib.ProviderCallableProfileInterface,
	sourceType types.Type,
	source api.ExpressionEmission,
	leafPolicy providerBoundaryLeafPolicy,
) (api.ExpressionEmission, bool, bool, error) {
	pointer, ok := types.Unalias(sourceType).(*types.Pointer)
	if !ok || pointer.Elem() == nil {
		return api.ExpressionEmission{}, false, false, nil
	}
	element := pointer.Elem()
	temporary, err := context.Names().Temporary(api.TemporaryConversionOperand)
	if err != nil {
		return api.ExpressionEmission{}, true, false, err
	}
	raw := context.Factory().Identifier(temporary)
	rawValue := tsgo.Expression(raw)
	if !directProviderPointerObject(element) {
		rawValue = context.Factory().PropertyAccessExpression(
			raw,
			nil,
			context.Factory().Identifier("value"),
			tsgo.NodeFlagsNone,
		)
	}
	read, _, err := fromProviderValueWithPolicy(
		context,
		children,
		owner,
		ownerBridge,
		profile,
		element,
		api.DirectExpression(rawValue),
		leafPolicy,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, false, err
	}
	targetElement, err := children.RepresentedType(
		context.WithRole(api.RoleResultType),
		nil,
		element,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, false, err
	}
	getter := context.Factory().ArrowFunction(
		nil,
		nil,
		nil,
		targetElement.Value(),
		context.Factory().EqualsGreaterThanToken(),
		context.Factory().Block(
			append(read.Before(), context.Factory().ReturnStatement(read.Value())),
			true,
		),
	)
	parameter := "$go$providerPointerValue"
	write, _, err := toProviderValueWithPolicy(
		context,
		children,
		owner,
		ownerBridge,
		profile,
		element,
		api.DirectExpression(context.Factory().Identifier(parameter)),
		leafPolicy,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, false, err
	}
	writeBody := write.Before()
	writeRequests := write.Requests()
	if directProviderPointerObject(element) {
		assignments := context.StableAssignments()
		if assignments == nil {
			return api.ExpressionEmission{}, true, false, boundaryInvariant(
				context,
				"provider named-pointer result requires stable-assignment semantics",
			)
		}
		assigned, assignErr := assignments.AssignStable(
			context,
			nil,
			element,
			raw,
			write,
		)
		if assignErr != nil {
			return api.ExpressionEmission{}, true, false, assignErr
		}
		writeBody = append(
			assigned.Before(),
			context.Factory().ExpressionStatement(assigned.Value()),
		)
		writeRequests = assigned.Requests()
	} else {
		writeBody = append(writeBody, context.Factory().ExpressionStatement(
			context.Factory().BinaryExpression(
				nil,
				context.Factory().PropertyAccessExpression(
					raw,
					nil,
					context.Factory().Identifier("value"),
					tsgo.NodeFlagsNone,
				),
				nil,
				context.Factory().BinaryOperatorToken(tsgo.BinaryOperatorEqualsToken),
				write.Value(),
			),
		))
	}
	setter := context.Factory().ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{context.Factory().ParameterDeclaration(
			nil,
			nil,
			context.Factory().Identifier(parameter),
			nil,
			targetElement.Value(),
			nil,
		)},
		context.Factory().KeywordTypeNode(tsgo.KeywordTypeSyntaxKindVoidKeyword),
		context.Factory().EqualsGreaterThanToken(),
		context.Factory().Block(writeBody, true),
	)
	bound, err := pointermarker.Operation(
		context,
		tsoniccore.SymbolBindPointer,
		[]api.TypeEmission{targetElement},
		[]api.ExpressionEmission{
			api.DirectExpression(raw),
			api.DirectExpression(getter, read.Requests()...),
			api.DirectExpression(setter, writeRequests...),
		},
	)
	if err != nil {
		return api.ExpressionEmission{}, true, false, err
	}
	before := append(source.Before(), captureStatement(
		context.Factory(),
		temporary,
		source.Value(),
	))
	result, err := api.NewExpressionEmission(
		before,
		context.Factory().ConditionalExpression(
			isUndefined(context.Factory(), temporary),
			context.Factory().QuestionToken(),
			context.Factory().Identifier("undefined"),
			context.Factory().ColonToken(),
			bound.Value(),
		),
		api.CombineRequests(source.Requests(), bound.Requests()),
	)
	return result, true, true, err
}

func toProviderPointer(
	context api.Context,
	children api.ChildEmitter,
	owner *types.Named,
	ownerBridge string,
	profile []gostdlib.ProviderCallableProfileInterface,
	sourceType types.Type,
	source api.ExpressionEmission,
	leafPolicy providerBoundaryLeafPolicy,
) (api.ExpressionEmission, bool, bool, error) {
	pointer, ok := types.Unalias(sourceType).(*types.Pointer)
	if !ok || pointer.Elem() == nil {
		return api.ExpressionEmission{}, false, false, nil
	}
	element := pointer.Elem()
	if !directProviderPointerObject(element) {
		return api.ExpressionEmission{}, true, false, boundaryInvariant(
			context,
			"provider pointer input requires an exact external-location transport contract",
		)
	}
	temporary, err := context.Names().Temporary(api.TemporaryConversionOperand)
	if err != nil {
		return api.ExpressionEmission{}, true, false, err
	}
	targetElement, err := children.RepresentedType(
		context.WithRole(api.RoleParameterType),
		nil,
		element,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, false, err
	}
	loaded, err := pointermarker.Operation(
		context,
		tsoniccore.SymbolLoadPointer,
		[]api.TypeEmission{targetElement},
		[]api.ExpressionEmission{api.DirectExpression(
			context.Factory().Identifier(temporary),
		)},
	)
	if err != nil {
		return api.ExpressionEmission{}, true, false, err
	}
	provider, _, err := toProviderValueWithPolicy(
		context,
		children,
		owner,
		ownerBridge,
		profile,
		element,
		loaded,
		leafPolicy,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, false, err
	}
	if len(provider.Before()) != 0 {
		return api.ExpressionEmission{}, true, false, boundaryInvariant(
			context,
			"provider named-pointer input conversion requires conditional statements",
		)
	}
	before := append(source.Before(), captureStatement(
		context.Factory(),
		temporary,
		source.Value(),
	))
	result, err := api.NewExpressionEmission(
		before,
		context.Factory().ConditionalExpression(
			isUndefined(context.Factory(), temporary),
			context.Factory().QuestionToken(),
			context.Factory().Identifier("undefined"),
			context.Factory().ColonToken(),
			provider.Value(),
		),
		api.CombineRequests(source.Requests(), provider.Requests()),
	)
	return result, true, true, err
}

func directProviderPointerObject(source types.Type) bool {
	named, ok := types.Unalias(source).(*types.Named)
	if !ok || named.Obj() == nil {
		return false
	}
	_, ok = named.Underlying().(*types.Struct)
	return ok
}
