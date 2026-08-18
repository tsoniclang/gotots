package interfaceadapter

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	genericinstance "github.com/tsoniclang/gotots/internal/emit/generic/instance"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type emittedHeritageSurface struct {
	qualifier string
	name      string
	arguments []types.Type
}

func contractHeritage(
	context api.Context,
	children api.ChildEmitter,
	contracts []Contract,
) (tsgo.HeritageClause, []api.RootRequest, error) {
	if len(contracts) == 0 {
		return nil, nil, nil
	}
	implements := make(
		[]tsgo.ExpressionWithTypeArguments,
		0,
		len(contracts),
	)
	var requests []api.RootRequest
	var emitted []emittedHeritageSurface
	for _, contract := range contracts {
		if !contract.valid() {
			return nil, nil, &api.GeneratedArtifactShapeError{
				Reason: "interface-adapter heritage contract is invalid",
			}
		}
		if !declarationHeritageSurface(contract.sourceType) {
			continue
		}
		reference, err := context.Names().InterfaceType(contract.sourceType)
		if err != nil {
			return nil, nil, err
		}
		var arguments []tsgo.TypeNode
		var argumentRequests []api.RootRequest
		if named, ok := types.Unalias(contract.sourceType).(*types.Named); ok &&
			named.TypeArgs().Len() != 0 {
			arguments, argumentRequests, err = genericinstance.EmitTypeArguments(
				context,
				children,
				nil,
				named.Origin().Obj(),
				api.TypeArgumentsFromGo(named.TypeArgs()),
			)
			if err != nil {
				return nil, nil, err
			}
		}
		requests = append(requests, reference.Requests()...)
		requests = append(requests, argumentRequests...)
		if heritageSurfaceEmitted(emitted, reference, contract.sourceType) {
			continue
		}
		implements = append(
			implements,
			context.Factory().ExpressionWithTypeArguments(
				reference.Expression(context.Factory()),
				arguments,
			),
		)
		emitted = append(
			emitted,
			heritageSurface(reference, contract.sourceType),
		)
	}
	if len(implements) == 0 {
		return nil, nil, nil
	}
	return context.Factory().HeritageClause(
		tsgo.HeritageClauseTokenKindImplementsKeyword,
		implements,
	), api.CombineRequests(requests), nil
}

func heritageSurfaceEmitted(
	emitted []emittedHeritageSurface,
	reference api.NameReference,
	sourceType types.Type,
) bool {
	selected := heritageSurface(reference, sourceType)
	for _, existing := range emitted {
		if existing.qualifier != selected.qualifier ||
			existing.name != selected.name ||
			len(existing.arguments) != len(selected.arguments) {
			continue
		}
		same := true
		for index, argument := range existing.arguments {
			if !types.Identical(argument, selected.arguments[index]) {
				same = false
				break
			}
		}
		if same {
			return true
		}
	}
	return false
}

func heritageSurface(
	reference api.NameReference,
	sourceType types.Type,
) emittedHeritageSurface {
	qualifier, _ := reference.Qualifier()
	selected := emittedHeritageSurface{
		qualifier: qualifier,
		name:      reference.Name(),
	}
	named, ok := types.Unalias(sourceType).(*types.Named)
	if !ok {
		return selected
	}
	selected.arguments = make([]types.Type, named.TypeArgs().Len())
	for index := range named.TypeArgs().Len() {
		selected.arguments[index] = named.TypeArgs().At(index)
	}
	return selected
}

func declarationHeritageSurface(sourceType types.Type) bool {
	selected := types.Unalias(sourceType)
	if _, ok := selected.Underlying().(*types.Interface); !ok {
		return false
	}
	if _, ok := selected.(*types.Interface); ok {
		return true
	}
	named, ok := selected.(*types.Named)
	if !ok {
		return false
	}
	object := named.Obj()
	if object == nil {
		return false
	}
	if object.Pkg() == nil {
		return object.Parent() == types.Universe
	}
	return object.Parent() == object.Pkg().Scope()
}
