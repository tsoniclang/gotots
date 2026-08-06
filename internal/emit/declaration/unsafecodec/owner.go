package unsafecodec

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type builder struct {
	context  api.Context
	children api.ChildEmitter
	factory  tsgo.Factory
	names    api.UnsafeCodecNames
	requests []api.RootRequest
}

func Build(
	context api.Context,
	children api.ChildEmitter,
	artifact *api.GeneratedArtifact,
) ([]tsgo.Statement, []api.RootRequest, error) {
	sourceType, ok := artifact.UnsafeCodecType()
	if !ok || sourceType == nil || api.ContainsGenericTypeParameter(sourceType) {
		return nil, nil, &api.GeneratedArtifactShapeError{
			Artifact: artifact.TargetName(),
			Reason:   "unsafe-codec source is invalid",
		}
	}
	names, ok := context.Names().(api.UnsafeCodecNames)
	if !ok {
		return nil, nil, &api.ContextError{
			Reason: "unsafe-codec names are unavailable",
		}
	}
	target := &builder{
		context:  context,
		children: children,
		factory:  context.Factory(),
		names:    names,
	}
	if pointer, pointerOK := pointerSource(sourceType); pointerOK {
		selection, err := pointertype.Observe(
			context,
			pointer,
			api.PointerRepresentationDemandDynamicLocation,
		)
		if err != nil {
			return nil, nil, err
		}
		target.addRequests(selection.Requests())
	}
	storage, err := target.storageType(sourceType)
	if err != nil {
		return nil, nil, err
	}
	read, write, err := target.operations(sourceType, storage)
	if err != nil {
		return nil, nil, err
	}
	runtime, err := context.Names().Runtime(
		api.RuntimeUnsafeCodec,
		api.ImportPhaseValue,
	)
	if err != nil {
		return nil, nil, err
	}
	target.addRequests(runtime.Requests())
	size := context.TypesSizes().Sizeof(sourceType)
	if size < 0 {
		return nil, nil, &api.GeneratedArtifactShapeError{
			Artifact: artifact.TargetName(),
			Reason:   "unsafe-codec source has invalid size",
		}
	}
	codecType := target.factory.TypeReferenceNode(
		runtime.EntityName(target.factory),
		[]tsgo.TypeNode{storage},
	)
	initializer := target.factory.NewExpression(
		runtime.Expression(target.factory),
		[]tsgo.TypeNode{storage},
		[]tsgo.Expression{
			target.number(size),
			read,
			write,
		},
	)
	statement := target.factory.VariableStatement(
		[]tsgo.ModifierLike{target.factory.ExportKeyword()},
		target.factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{target.factory.VariableDeclaration(
				target.id(artifact.TargetName()),
				nil,
				codecType,
				initializer,
			)},
			tsgo.NodeFlagsConst,
		),
	)
	return []tsgo.Statement{statement}, target.requests, nil
}

func (b *builder) storageType(sourceType types.Type) (tsgo.TypeNode, error) {
	selection, err := pointertype.Observe(
		b.context,
		types.NewPointer(sourceType),
		api.PointerRepresentationDemandDynamicLocation,
	)
	if err != nil {
		return nil, err
	}
	target, err := b.context.ContainerStorage().PointerStorageType(
		b.context.WithRole(api.RoleStorageType), nil, sourceType, selection,
	)
	if err != nil {
		return nil, err
	}
	b.addRequests(selection.Requests(), target.Requests())
	return target.Value(), nil
}

func (b *builder) representedType(sourceType types.Type) (tsgo.TypeNode, error) {
	target, err := b.children.RepresentedType(
		b.context.WithRole(api.RoleStorageType),
		nil,
		sourceType,
	)
	if err != nil {
		return nil, err
	}
	b.addRequests(target.Requests())
	return target.Value(), nil
}

func (b *builder) codec(sourceType types.Type) (tsgo.Expression, error) {
	reference, err := b.names.UnsafeCodec(sourceType)
	if err != nil {
		return nil, err
	}
	b.addRequests(reference.Requests())
	return reference.Expression(b.factory), nil
}

func (b *builder) runtime(
	symbol api.RuntimeSymbol,
	phase api.ImportPhase,
) (api.NameReference, error) {
	reference, err := b.context.Names().Runtime(symbol, phase)
	if err != nil {
		return api.NameReference{}, err
	}
	b.addRequests(reference.Requests())
	return reference, nil
}

func (b *builder) addRequests(groups ...[]api.RootRequest) {
	b.requests = api.CombineRequests(append([][]api.RootRequest{b.requests}, groups...)...)
}

func pointerSource(sourceType types.Type) (*types.Pointer, bool) {
	if pointer, ok := types.Unalias(sourceType).(*types.Pointer); ok {
		return pointer, true
	}
	named, ok := types.Unalias(sourceType).(*types.Named)
	if !ok {
		return nil, false
	}
	pointer, ok := named.Underlying().(*types.Pointer)
	return pointer, ok
}
