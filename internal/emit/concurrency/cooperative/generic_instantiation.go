package cooperative

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
)

type genericTypePair struct {
	declaration  types.Type
	instantiated types.Type
}

type genericCallableCorrespondence struct {
	context    api.Context
	owner      types.Object
	seen       map[genericTypePair]struct{}
	selections map[*api.GeneratedArtifact]bool
	requests   []api.RootRequest
}

func CorrespondGenericCallableABIs(
	context api.Context,
	owner types.Object,
	declaration *types.Signature,
	instantiated *types.Signature,
) (
	api.GenericCallableProfileSelection,
	[]api.RootRequest,
	error,
) {
	owner = api.GenericDeclarationOrigin(owner)
	if owner == nil ||
		declaration == nil ||
		instantiated == nil ||
		declaration.Params().Len() != instantiated.Params().Len() ||
		declaration.Results().Len() != instantiated.Results().Len() ||
		declaration.Variadic() != instantiated.Variadic() {
		return api.GenericCallableProfileSelection{}, nil, &api.InvariantError{
			Role:   api.RoleCallArgument,
			Reason: "generic callable correspondence is invalid",
		}
	}
	correspondence := genericCallableCorrespondence{
		context: context,
		owner:   owner,
		seen:    make(map[genericTypePair]struct{}),
		selections: make(
			map[*api.GeneratedArtifact]bool,
		),
	}
	if err := correspondence.signatureMembers(
		declaration,
		instantiated,
	); err != nil {
		return api.GenericCallableProfileSelection{}, nil, err
	}
	selections := make(
		[]api.GenericCallableABISelection,
		0,
		len(correspondence.selections),
	)
	for artifact, cooperative := range correspondence.selections {
		selection, err := api.NewGenericCallableABISelection(
			artifact,
			cooperative,
		)
		if err != nil {
			return api.GenericCallableProfileSelection{}, nil, err
		}
		selections = append(selections, selection)
	}
	profile, err := api.NewGenericCallableProfileSelection(selections)
	if err != nil {
		return api.GenericCallableProfileSelection{}, nil, err
	}
	return profile,
		api.CombineRequests(correspondence.requests),
		nil
}

func (c *genericCallableCorrespondence) signatureMembers(
	declaration *types.Signature,
	instantiated *types.Signature,
) error {
	for index := range declaration.Params().Len() {
		if err := c.pair(
			declaration.Params().At(index).Type(),
			instantiated.Params().At(index).Type(),
		); err != nil {
			return err
		}
	}
	for index := range declaration.Results().Len() {
		if err := c.pair(
			declaration.Results().At(index).Type(),
			instantiated.Results().At(index).Type(),
		); err != nil {
			return err
		}
	}
	return nil
}

func (c *genericCallableCorrespondence) pair(
	declaration types.Type,
	instantiated types.Type,
) error {
	if declaration == nil || instantiated == nil {
		return c.invalid()
	}
	declaration = types.Unalias(declaration)
	instantiated = types.Unalias(instantiated)
	if _, parameter := declaration.(*types.TypeParam); parameter {
		return nil
	}
	pair := genericTypePair{
		declaration:  declaration,
		instantiated: instantiated,
	}
	if _, visited := c.seen[pair]; visited {
		return nil
	}
	c.seen[pair] = struct{}{}

	declarationCallable, declarationIsCallable :=
		callable.Signature(declaration)
	if declarationIsCallable {
		instantiatedCallable, instantiatedIsCallable :=
			callable.Signature(instantiated)
		if !instantiatedIsCallable {
			return c.invalid()
		}
		if err := c.callable(
			declarationCallable,
			instantiatedCallable,
		); err != nil {
			return err
		}
		return c.signatureMembers(
			declarationCallable,
			instantiatedCallable,
		)
	}
	if types.Identical(declaration, instantiated) {
		return nil
	}

	switch declaration := declaration.(type) {
	case *types.Array:
		instantiated, ok := instantiated.(*types.Array)
		if !ok || declaration.Len() != instantiated.Len() {
			return c.invalid()
		}
		return c.pair(declaration.Elem(), instantiated.Elem())
	case *types.Slice:
		instantiated, ok := instantiated.(*types.Slice)
		if !ok {
			return c.invalid()
		}
		return c.pair(declaration.Elem(), instantiated.Elem())
	case *types.Pointer:
		instantiated, ok := instantiated.(*types.Pointer)
		if !ok {
			return c.invalid()
		}
		return c.pair(declaration.Elem(), instantiated.Elem())
	case *types.Map:
		instantiated, ok := instantiated.(*types.Map)
		if !ok {
			return c.invalid()
		}
		if err := c.pair(declaration.Key(), instantiated.Key()); err != nil {
			return err
		}
		return c.pair(declaration.Elem(), instantiated.Elem())
	case *types.Chan:
		instantiated, ok := instantiated.(*types.Chan)
		if !ok || declaration.Dir() != instantiated.Dir() {
			return c.invalid()
		}
		return c.pair(declaration.Elem(), instantiated.Elem())
	case *types.Named:
		instantiated, ok := instantiated.(*types.Named)
		if !ok || declaration.Origin() != instantiated.Origin() {
			return c.invalid()
		}
		return c.pair(
			declaration.Underlying(),
			instantiated.Underlying(),
		)
	case *types.Struct:
		instantiated, ok := instantiated.(*types.Struct)
		if !ok || declaration.NumFields() != instantiated.NumFields() {
			return c.invalid()
		}
		for index := range declaration.NumFields() {
			if err := c.pair(
				declaration.Field(index).Type(),
				instantiated.Field(index).Type(),
			); err != nil {
				return err
			}
		}
		return nil
	case *types.Tuple:
		instantiated, ok := instantiated.(*types.Tuple)
		if !ok || declaration.Len() != instantiated.Len() {
			return c.invalid()
		}
		for index := range declaration.Len() {
			if err := c.pair(
				declaration.At(index).Type(),
				instantiated.At(index).Type(),
			); err != nil {
				return err
			}
		}
		return nil
	case *types.Interface:
		instantiated, ok := instantiated.(*types.Interface)
		if !ok {
			return c.invalid()
		}
		declaration = declaration.Complete()
		instantiated = instantiated.Complete()
		if declaration.NumMethods() != instantiated.NumMethods() {
			return c.invalid()
		}
		for index := range declaration.NumMethods() {
			declarationMethod := declaration.Method(index)
			instantiatedMethod := instantiated.Method(index)
			if declarationMethod.Id() != instantiatedMethod.Id() {
				return c.invalid()
			}
			declarationSignature, declarationOK :=
				declarationMethod.Type().(*types.Signature)
			instantiatedSignature, instantiatedOK :=
				instantiatedMethod.Type().(*types.Signature)
			if !declarationOK || !instantiatedOK {
				return c.invalid()
			}
			if err := c.signatureMembers(
				declarationSignature,
				instantiatedSignature,
			); err != nil {
				return err
			}
		}
		return nil
	case *types.Basic:
		return c.invalid()
	default:
		return nil
	}
}

func (c *genericCallableCorrespondence) callable(
	declaration *types.Signature,
	instantiated *types.Signature,
) error {
	declarationReference, err :=
		c.context.Names().SourceCallableABI(c.owner, declaration)
	if err != nil {
		return err
	}
	instantiatedReference, err :=
		c.context.Names().CallableABI(instantiated)
	if err != nil {
		return err
	}
	declarationObservation, err :=
		c.observe(declarationReference)
	if err != nil {
		return err
	}
	instantiatedObservation, err :=
		c.observe(instantiatedReference)
	if err != nil {
		return err
	}
	if declarationObservation.Cooperative() &&
		!instantiatedObservation.Cooperative() {
		instantiatedFacet, err := api.NewCallableABIFacet(
			instantiatedReference.Artifact(),
		)
		if err != nil {
			return err
		}
		request, err := api.NewCooperativeCallableRequest(
			instantiatedFacet,
		)
		if err != nil {
			return err
		}
		c.requests = append(c.requests, request)
	}
	if !declarationObservation.Cooperative() &&
		instantiatedObservation.Cooperative() {
		c.selections[declarationReference.Artifact()] = true
	}
	return nil
}

func (c *genericCallableCorrespondence) observe(
	reference api.CallableABIReference,
) (api.CooperativeCallableObservation, error) {
	facet, err := api.NewCallableABIFacet(reference.Artifact())
	if err != nil {
		return api.CooperativeCallableObservation{}, err
	}
	observation, err := c.context.ObserveCooperativeCallable(facet)
	if err != nil {
		return api.CooperativeCallableObservation{}, err
	}
	c.requests = append(c.requests, reference.Requests()...)
	c.requests = append(c.requests, observation.Requests()...)
	return observation, nil
}

func (c *genericCallableCorrespondence) invalid() error {
	return &api.InvariantError{
		Role:   api.RoleCallArgument,
		Reason: "generic callable correspondence changed structural shape",
	}
}
