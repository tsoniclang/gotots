package representation

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	channeltype "github.com/tsoniclang/gotots/internal/emit/type/channel"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
	"github.com/tsoniclang/gotots/internal/emit/value/maprepresentation"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (owner Owner) PointerRepresentation(
	context api.Context,
	pointer *types.Pointer,
	carrierDemand bool,
) (api.PointerRepresentationObservation, error) {
	if pointer == nil {
		return api.PointerRepresentationObservation{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "pointer representation source is nil",
		}
	}
	baseline, err := api.DefaultPointerRepresentationForType(pointer)
	if err != nil {
		return api.PointerRepresentationObservation{}, err
	}
	if baseline == api.PointerRepresentationCarrierLogical &&
		!owner.RequiresStorageProjection(context, pointer.Elem()) {
		return api.NewPointerRepresentationObservation(
			api.PointerRepresentationCarrierLogical,
		)
	}
	names := context.PointerRepresentationNames()
	if names == nil {
		return api.PointerRepresentationObservation{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "pointer representation name owner is unavailable",
		}
	}
	reference, err := names.PointerRepresentation(pointer)
	if err != nil {
		return api.PointerRepresentationObservation{}, err
	}
	resolver, ok := owner.children.(api.PointerRepresentationResolver)
	if !ok {
		return api.PointerRepresentationObservation{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "pointer representation resolver is unavailable",
		}
	}
	observation, err := resolver.ObservePointerRepresentation(
		context.ArtifactOwner(),
		reference.Artifact(),
		carrierDemand,
	)
	if err != nil {
		return api.PointerRepresentationObservation{}, err
	}
	return api.NewPointerRepresentationObservation(
		observation.Representation(),
		api.CombineRequests(
			reference.Requests(),
			observation.Requests(),
		)...,
	)
}

func primitive(
	context api.Context,
	sourceType types.Type,
) (api.PrimitiveAlias, bool) {
	return basictype.PrimitiveAlias(context.TypesSizes(), sourceType)
}

func callableValue(sourceType types.Type) bool {
	_, ok := callable.Signature(sourceType)
	return ok
}

func pointerValue(sourceType types.Type) bool {
	_, _, ok := pointertype.Resolve(sourceType)
	return ok
}

func unsafePointerValue(sourceType types.Type) bool {
	return basictype.SupportsUnsafePointer(sourceType)
}

func channelValue(sourceType types.Type) bool {
	_, ok := channeltype.Resolve(sourceType)
	return ok
}

func mapValue(context api.Context, sourceType types.Type) bool {
	_, ok := maprepresentation.Source(context, sourceType)
	return ok
}

func namedStruct(
	sourceType types.Type,
) (*types.TypeName, *types.Struct, bool) {
	if sourceType == nil {
		return nil, nil, false
	}
	named, ok := types.Unalias(sourceType).(*types.Named)
	if !ok ||
		named.Obj() == nil ||
		(named.TypeParams().Len() != 0 &&
			named.TypeArgs().Len() != named.TypeParams().Len()) {
		return nil, nil, false
	}
	structType, ok := named.Underlying().(*types.Struct)
	if !ok {
		return nil, nil, false
	}
	return named.Origin().Obj(), structType, true
}

func staticStructOperationCall(
	context api.Context,
	className string,
	operation api.NamedStructOperation,
	arguments []tsgo.Expression,
) (tsgo.CallExpression, error) {
	memberName, err := api.NamedStructOperationMemberName(operation)
	if err != nil {
		return nil, err
	}
	return context.Factory().CallExpression(
		context.Factory().PropertyAccessExpression(
			context.Factory().Identifier(className),
			nil,
			context.Factory().Identifier(memberName),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	), nil
}
