package call

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	unsafeoperation "github.com/tsoniclang/gotots/internal/emit/expression/builtin/unsafeoperation"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitUnsafeRuntimeCall(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	kind unsafeoperation.Kind,
	signature *types.Signature,
	arguments []tsgo.Expression,
	before []tsgo.Statement,
	requests []api.RootRequest,
) (api.ExpressionEmission, error) {
	symbol, pointerType, err := unsafeRuntimeSelection(kind, signature)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	logical, storage, typeRequests, err := unsafePointerTypeArguments(
		context,
		children,
		source,
		pointerType,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	reference, err := context.Names().Runtime(symbol, api.ImportPhaseValue)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	var typeArguments []tsgo.TypeNode
	switch kind {
	case unsafeoperation.String, unsafeoperation.StringData:
		typeArguments = []tsgo.TypeNode{logical.Value()}
	case unsafeoperation.Slice, unsafeoperation.SliceData:
		typeArguments = []tsgo.TypeNode{logical.Value(), storage.Value()}
	}
	if kind == unsafeoperation.StringData {
		converter, err := unsafeByteConverter(context)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		arguments = append(arguments, converter)
	}
	return api.NewExpressionEmission(
		before,
		context.Factory().CallExpression(
			context.Factory().Identifier(reference.Name()),
			nil,
			typeArguments,
			arguments,
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			requests,
			typeRequests,
			logical.Requests(),
			storage.Requests(),
			reference.Requests(),
		),
	)
}

func unsafeRuntimeSelection(
	kind unsafeoperation.Kind,
	signature *types.Signature,
) (api.RuntimeSymbol, *types.Pointer, error) {
	if signature == nil || signature.Params() == nil || signature.Results() == nil {
		return api.RuntimeInvalid, nil, &api.InvariantError{
			Reason: "unsafe runtime signature is absent",
		}
	}
	var symbol api.RuntimeSymbol
	var pointerType types.Type
	switch kind {
	case unsafeoperation.String:
		symbol = api.RuntimeUnsafeString
		pointerType = signature.Params().At(0).Type()
	case unsafeoperation.Slice:
		symbol = api.RuntimeUnsafeSlice
		pointerType = signature.Params().At(0).Type()
	case unsafeoperation.StringData:
		symbol = api.RuntimeUnsafeStringData
		pointerType = signature.Results().At(0).Type()
	case unsafeoperation.SliceData:
		symbol = api.RuntimeUnsafeSliceData
		pointerType = signature.Results().At(0).Type()
	default:
		return api.RuntimeInvalid, nil, &api.InvariantError{
			Reason: "unsafe runtime operation is invalid",
		}
	}
	pointer, _, ok := pointertype.Resolve(pointerType)
	if !ok {
		return api.RuntimeInvalid, nil, &api.InvariantError{
			Reason: "unsafe runtime pointer contract is invalid",
		}
	}
	return symbol, pointer, nil
}

func unsafePointerTypeArguments(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	pointer *types.Pointer,
) (api.TypeEmission, api.TypeEmission, []api.RootRequest, error) {
	if pointer == nil {
		return api.TypeEmission{}, api.TypeEmission{}, nil, &api.InvariantError{
			Reason: "unsafe runtime pointer type is nil",
		}
	}
	representation, err := pointertype.Observe(
		context,
		pointer,
		api.PointerRepresentationDemandDynamicLocation,
	)
	if err != nil {
		return api.TypeEmission{}, api.TypeEmission{}, nil, err
	}
	logical, err := children.RepresentedType(
		context.WithRole(api.RoleBuiltinArgument),
		source,
		pointer.Elem(),
	)
	if err != nil {
		return api.TypeEmission{}, api.TypeEmission{}, nil, err
	}
	storage, err := context.ContainerStorage().PointerStorageType(
		context.WithRole(api.RoleStorageType),
		source,
		pointer.Elem(),
		representation,
	)
	if representation.Representation().DirectClass() {
		storage, err = context.Values().StorageType(
			context.WithRole(api.RoleStorageType),
			source,
			pointer.Elem(),
		)
	}
	if err != nil {
		return api.TypeEmission{}, api.TypeEmission{}, nil, err
	}
	return logical, storage, representation.Requests(), nil
}

func unsafeByteConverter(context api.Context) (tsgo.Expression, error) {
	if !context.IntegerRepresentation().Valid() {
		return nil, &api.IntegerRepresentationError{
			Representation: context.IntegerRepresentation(),
		}
	}
	return context.Factory().PropertyAccessExpression(
		context.Factory().Identifier(api.TargetGlobalAnchorName),
		nil,
		context.Factory().Identifier("Number"),
		tsgo.NodeFlagsNone,
	), nil
}
