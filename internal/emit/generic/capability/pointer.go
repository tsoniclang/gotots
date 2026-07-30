package capability

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	pointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func buildPointerCapability(
	context api.Context,
	children api.ChildEmitter,
	artifact *api.GeneratedArtifact,
	modifiers []tsgo.ModifierLike,
	signature *types.Signature,
	selection api.GenericOperationSelection,
) (tsgo.Statement, []api.RootRequest, bool, error) {
	element, ok := api.GenericPointerOperationElement(selection, signature)
	if !ok {
		return nil, nil, false, nil
	}
	if context.IsCooperative() {
		return nil, nil, true, invariant(
			context,
			"pointer capability cannot be cooperative",
		)
	}
	logical, err := children.RepresentedType(
		context.WithRole(api.RoleParameterType),
		nil,
		element,
	)
	if err != nil {
		return nil, nil, true, err
	}
	storage, err := context.Values().StorageType(
		context.WithRole(api.RoleStorageType),
		nil,
		element,
	)
	if err != nil {
		return nil, nil, true, err
	}
	pointerSource := types.NewPointer(element)
	pointer, err := pointertype.EmitRepresented(
		context.WithRole(api.RoleParameterType),
		children,
		nil,
		pointerSource,
	)
	if err != nil {
		return nil, nil, true, err
	}
	nonNilPointer, err := pointertype.EmitNonNilRepresented(
		context.WithRole(api.RoleResultType),
		children,
		nil,
		pointerSource,
	)
	if err != nil {
		return nil, nil, true, err
	}
	runtime, err := context.Names().Runtime(
		api.RuntimePointer,
		api.ImportPhaseValue,
	)
	if err != nil {
		return nil, nil, true, err
	}
	parameters, resultType, body, operationRequests, err :=
		pointerCapabilityBody(
			context,
			element,
			selection.Operation(),
			logical.Value(),
			storage.Value(),
			pointer.Value(),
			nonNilPointer.Value(),
			runtime,
		)
	if err != nil {
		return nil, nil, true, err
	}
	return context.Factory().FunctionDeclaration(
			modifiers,
			nil,
			context.Factory().Identifier(artifact.TargetName()),
			nil,
			parameters,
			resultType,
			context.Factory().Block(body, true),
		), api.CombineRequests(
			logical.Requests(),
			storage.Requests(),
			pointer.Requests(),
			nonNilPointer.Requests(),
			runtime.Requests(),
			operationRequests,
		), true, nil
}

func pointerCapabilityBody(
	context api.Context,
	element types.Type,
	operation api.GenericOperation,
	logicalType tsgo.TypeNode,
	storageType tsgo.TypeNode,
	pointerType tsgo.TypeNode,
	nonNilPointerType tsgo.TypeNode,
	runtime api.NameReference,
) (
	[]tsgo.ParameterDeclaration,
	tsgo.TypeNode,
	[]tsgo.Statement,
	[]api.RootRequest,
	error,
) {
	first := context.Factory().Identifier("$0")
	second := context.Factory().Identifier("$1")
	switch operation {
	case api.GenericOperationPointerCell:
		stored, err := context.Values().ToStorage(
			context.WithRole(api.RoleFunctionBody),
			nil,
			element,
			api.DirectExpression(first),
		)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		body := append(stored.Before(), context.Factory().ReturnStatement(
			pointerruntime.Cell(
				context.Factory(),
				runtime.Name(),
				logicalType,
				storageType,
				stored.Value(),
			),
		))
		return []tsgo.ParameterDeclaration{
				operationParameter(context, "$0", logicalType),
			},
			nonNilPointerType,
			body,
			stored.Requests(),
			nil
	case api.GenericOperationPointerLoad:
		stored := api.DirectExpression(pointerruntime.CellValue(
			context.Factory(),
			runtime.Name(),
			logicalType,
			storageType,
			first,
		))
		value, err := context.Values().FromStorage(
			context.WithRole(api.RoleFunctionBody),
			nil,
			element,
			stored,
		)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		body := append(value.Before(), context.Factory().ReturnStatement(
			value.Value(),
		))
		return []tsgo.ParameterDeclaration{
				operationParameter(context, "$0", pointerType),
			},
			logicalType,
			body,
			value.Requests(),
			nil
	case api.GenericOperationPointerStore:
		stored, err := context.Values().ToStorage(
			context.WithRole(api.RoleFunctionBody),
			nil,
			element,
			api.DirectExpression(second),
		)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		target := context.Factory().PropertyAccessExpression(
			pointerruntime.Dereference(
				context.Factory(),
				runtime.Name(),
				logicalType,
				storageType,
				first,
			),
			nil,
			context.Factory().Identifier(pointerruntime.CellValueName),
			tsgo.NodeFlagsNone,
		)
		body := append(stored.Before(), context.Factory().ExpressionStatement(
			context.Factory().BinaryExpression(
				nil,
				target,
				nil,
				context.Factory().BinaryOperatorToken(
					tsgo.BinaryOperatorEqualsToken,
				),
				stored.Value(),
			),
		))
		return []tsgo.ParameterDeclaration{
				operationParameter(context, "$0", pointerType),
				operationParameter(context, "$1", logicalType),
			},
			context.Factory().KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindVoidKeyword,
			),
			body,
			stored.Requests(),
			nil
	default:
		return nil, nil, nil, nil, invariant(
			context,
			"pointer capability operation is invalid",
		)
	}
}

func operationParameter(
	context api.Context,
	name string,
	targetType tsgo.TypeNode,
) tsgo.ParameterDeclaration {
	return context.Factory().ParameterDeclaration(
		nil,
		nil,
		context.Factory().Identifier(name),
		nil,
		targetType,
		nil,
	)
}
