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

type callableCorrespondence struct {
	context    api.Context
	owner      types.Object
	profile    *api.GenericCallableProfile
	seen       map[genericTypePair]struct{}
	selections map[*api.GeneratedArtifact]bool
	requests   []api.RootRequest
	leaf       func(*types.Signature, *types.Signature) error
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
	correspondence := callableCorrespondence{
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

func PropagateGenericCallableProfile(
	context api.Context,
	owner types.Object,
	profile *api.GenericCallableProfile,
	declaration *types.Signature,
	instantiated *types.Signature,
) ([]api.RootRequest, error) {
	owner = api.GenericDeclarationOrigin(owner)
	if owner == nil ||
		!profile.Valid() ||
		profile.Owner() != owner ||
		declaration == nil ||
		instantiated == nil ||
		declaration.Params().Len() != instantiated.Params().Len() ||
		declaration.Results().Len() != instantiated.Results().Len() ||
		declaration.Variadic() != instantiated.Variadic() {
		return nil, &api.InvariantError{
			Role:   api.RoleCallArgument,
			Reason: "generic callable profile propagation is invalid",
		}
	}
	correspondence := callableCorrespondence{
		context:    context,
		owner:      owner,
		profile:    profile,
		seen:       make(map[genericTypePair]struct{}),
		selections: make(map[*api.GeneratedArtifact]bool),
	}
	if err := correspondence.signatureMembers(
		declaration,
		instantiated,
	); err != nil {
		return nil, err
	}
	return api.CombineRequests(correspondence.requests), nil
}

func (c *callableCorrespondence) signatureMembers(
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

func (c *callableCorrespondence) pair(
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

func (c *callableCorrespondence) callable(
	declaration *types.Signature,
	instantiated *types.Signature,
) error {
	if c.leaf != nil {
		return c.leaf(declaration, instantiated)
	}
	declarationReference, err :=
		c.context.Names().SourceCallableABI(c.owner, declaration)
	if err != nil {
		return err
	}
	instantiatedReference, err :=
		callable.ABIReference(c.context, instantiated)
	if err != nil {
		return err
	}
	var declarationFacet api.CallableFacet
	if c.profile == nil {
		declarationFacet, err = api.NewCallableABIFacet(
			declarationReference.Artifact(),
		)
	} else {
		declarationFacet, err =
			api.NewGenericProfileCallableABIFacet(
				c.profile,
				declarationReference.Artifact(),
			)
	}
	if err != nil {
		return err
	}
	instantiatedFacet, err :=
		c.context.CallableABIFacet(instantiatedReference)
	if err != nil {
		return err
	}
	declarationObservation, err := c.observe(
		declarationReference,
		declarationFacet,
	)
	if err != nil {
		return err
	}
	instantiatedObservation, err := c.observe(
		instantiatedReference,
		instantiatedFacet,
	)
	if err != nil {
		return err
	}
	if declarationObservation.Cooperative() &&
		!instantiatedObservation.Cooperative() {
		request, err := api.NewCooperativeCallableRequest(
			instantiatedFacet,
		)
		if err != nil {
			return err
		}
		c.requests = append(c.requests, request)
	}
	if c.profile != nil {
		return nil
	}
	if !declarationObservation.Cooperative() &&
		instantiatedObservation.Cooperative() {
		c.selections[declarationReference.Artifact()] = true
	}
	return nil
}

func (c *callableCorrespondence) observe(
	reference api.CallableABIReference,
	facet api.CallableFacet,
) (api.CooperativeCallableObservation, error) {
	if !facet.Valid() {
		return api.CooperativeCallableObservation{}, &api.InvariantError{
			Role:   api.RoleCallArgument,
			Reason: "generic callable observation facet is invalid",
		}
	}
	observation, err := c.context.ObserveCooperativeCallable(facet)
	if err != nil {
		return api.CooperativeCallableObservation{}, err
	}
	c.requests = append(c.requests, reference.Requests()...)
	c.requests = append(c.requests, observation.Requests()...)
	return observation, nil
}

func (c *callableCorrespondence) invalid() error {
	return &api.InvariantError{
		Role:   api.RoleCallArgument,
		Reason: "generic callable correspondence changed structural shape",
	}
}

func JoinInterfaceMethodCallableABIs(
	context api.Context,
	selected api.InterfaceMethodCallableCorrespondence,
) ([]api.RootRequest, error) {
	owner, declaration, instantiated := selected.Parts()
	if owner == nil || declaration == nil || instantiated == nil {
		return nil, &api.InvariantError{
			Role:   context.Role(),
			Reason: "interface-method callable correspondence is invalid",
		}
	}
	var requests []api.RootRequest
	correspondence := callableCorrespondence{
		context: context,
		seen:    make(map[genericTypePair]struct{}),
		leaf: func(
			declaration *types.Signature,
			instantiated *types.Signature,
		) error {
			declarationReference, err :=
				context.Names().SourceCallableABI(
					owner,
					declaration,
				)
			if err != nil {
				return err
			}
			instantiatedReference, err :=
				callable.ABIReference(context, instantiated)
			if err != nil {
				return err
			}
			declarationFacet, err := api.NewCallableABIFacet(
				declarationReference.Artifact(),
			)
			if err != nil {
				return err
			}
			instantiatedFacet, err :=
				context.CallableABIFacet(instantiatedReference)
			if err != nil {
				return err
			}
			declarationObservation, err :=
				context.ObserveCooperativeCallable(
					declarationFacet,
				)
			if err != nil {
				return err
			}
			instantiatedObservation, err :=
				context.ObserveCooperativeCallable(
					instantiatedFacet,
				)
			if err != nil {
				return err
			}
			requests = append(
				requests,
				declarationReference.Requests()...,
			)
			requests = append(
				requests,
				instantiatedReference.Requests()...,
			)
			requests = append(
				requests,
				declarationObservation.Requests()...,
			)
			requests = append(
				requests,
				instantiatedObservation.Requests()...,
			)
			cooperative := declarationObservation.Cooperative() ||
				instantiatedObservation.Cooperative()
			for _, candidate := range []struct {
				facet       api.CallableFacet
				cooperative bool
			}{
				{
					facet: declarationFacet,
					cooperative: declarationObservation.
						Cooperative(),
				},
				{
					facet: instantiatedFacet,
					cooperative: instantiatedObservation.
						Cooperative(),
				},
			} {
				if !cooperative || candidate.cooperative {
					continue
				}
				request, requestErr :=
					api.NewCooperativeCallableRequest(
						candidate.facet,
					)
				if requestErr != nil {
					return requestErr
				}
				requests = append(requests, request)
			}
			return nil
		},
	}
	if err := correspondence.signatureMembers(
		declaration,
		instantiated,
	); err != nil {
		return nil, err
	}
	return api.CombineRequests(requests), nil
}
