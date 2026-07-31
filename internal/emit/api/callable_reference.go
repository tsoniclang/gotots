package api

import (
	"go/types"
	"slices"
	"strings"
)

type CallableABIReference struct {
	artifact    *GeneratedArtifact
	sourceOwner types.Object
	requests    []RootRequest
}

func NewCallableABIReference(
	artifact *GeneratedArtifact,
	requests ...RootRequest,
) (CallableABIReference, error) {
	_, ok := artifact.CallableABI()
	if !ok {
		return CallableABIReference{}, &RootRequestError{
			Reason: "callable ABI reference is invalid",
		}
	}
	if err := validateReferenceRequests(requests); err != nil {
		return CallableABIReference{}, &RootRequestError{
			Reason: "callable ABI reference request is invalid",
		}
	}
	return CallableABIReference{
		artifact: artifact,
		requests: slices.Clone(requests),
	}, nil
}

func NewSourceCallableABIReference(
	sourceOwner types.Object,
	artifact *GeneratedArtifact,
	requests ...RootRequest,
) (CallableABIReference, error) {
	sourceOwner = GenericDeclarationOrigin(sourceOwner)
	reference, err := NewCallableABIReference(artifact, requests...)
	if err != nil {
		return CallableABIReference{}, err
	}
	if sourceOwner == nil || sourceOwner.Pkg() == nil {
		return CallableABIReference{}, &RootRequestError{
			Reason: "source callable ABI reference owner is invalid",
		}
	}
	reference.sourceOwner = sourceOwner
	return reference, nil
}

func (r CallableABIReference) Artifact() *GeneratedArtifact {
	return r.artifact
}

func (r CallableABIReference) SourceOwner() (types.Object, bool) {
	return r.sourceOwner, r.sourceOwner != nil
}

func (r CallableABIReference) Requests() []RootRequest {
	return slices.Clone(r.requests)
}

type InterfaceMethodCallableCorrespondence struct {
	owner        *types.TypeName
	declaration  *types.Signature
	instantiated *types.Signature
}

func NewInterfaceMethodCallableCorrespondence(
	owner *types.TypeName,
	declaration *types.Signature,
	instantiated *types.Signature,
) (InterfaceMethodCallableCorrespondence, error) {
	origin, _ := GenericDeclarationOrigin(owner).(*types.TypeName)
	validSignatures := declaration != nil &&
		instantiated != nil &&
		declaration.Recv() == nil &&
		instantiated.Recv() == nil &&
		declaration.Params().Len() == instantiated.Params().Len() &&
		declaration.Results().Len() == instantiated.Results().Len() &&
		declaration.Variadic() == instantiated.Variadic() &&
		!types.Identical(declaration, instantiated)
	if origin == nil ||
		len(GenericDeclarationParameters(origin)) == 0 ||
		!validSignatures {
		return InterfaceMethodCallableCorrespondence{}, &NameError{
			Reason: "interface-method callable correspondence is invalid",
		}
	}
	return InterfaceMethodCallableCorrespondence{
		owner:        origin,
		declaration:  declaration,
		instantiated: instantiated,
	}, nil
}

func (c InterfaceMethodCallableCorrespondence) Parts() (
	*types.TypeName,
	*types.Signature,
	*types.Signature,
) {
	return c.owner, c.declaration, c.instantiated
}

type InterfaceMethodCallableReference struct {
	artifacts      []*GeneratedArtifact
	correspondence []InterfaceMethodCallableCorrespondence
	requests       []RootRequest
}

func NewInterfaceMethodCallableReference(
	artifacts []*GeneratedArtifact,
	correspondence []InterfaceMethodCallableCorrespondence,
	requests ...RootRequest,
) (InterfaceMethodCallableReference, error) {
	if len(artifacts) == 0 {
		return InterfaceMethodCallableReference{}, &NameError{
			Reason: "interface-method callable identities are absent",
		}
	}
	artifacts = slices.Clone(artifacts)
	slices.SortFunc(
		artifacts,
		func(left *GeneratedArtifact, right *GeneratedArtifact) int {
			if left == nil || right == nil {
				switch {
				case left == right:
					return 0
				case left == nil:
					return -1
				default:
					return 1
				}
			}
			return strings.Compare(left.ArtifactKey(), right.ArtifactKey())
		},
	)
	var previous *GeneratedArtifact
	for _, callable := range artifacts {
		if callable == nil ||
			callable.Kind() != GeneratedArtifactInterfaceMethodCallable ||
			!callable.Valid() ||
			callable == previous {
			return InterfaceMethodCallableReference{}, &NameError{
				Reason: "interface-method callable identities are invalid",
			}
		}
		previous = callable
	}
	correspondence = slices.Clone(correspondence)
	for _, selected := range correspondence {
		owner, declaration, instantiated := selected.Parts()
		if owner == nil || declaration == nil || instantiated == nil {
			return InterfaceMethodCallableReference{}, &NameError{
				Reason: "interface-method callable correspondences are invalid",
			}
		}
	}
	if err := validateReferenceRequests(requests); err != nil {
		return InterfaceMethodCallableReference{}, &RootRequestError{
			Reason: "interface-method reference request is invalid",
		}
	}
	return InterfaceMethodCallableReference{
		artifacts:      artifacts,
		correspondence: correspondence,
		requests:       slices.Clone(requests),
	}, nil
}

func (r InterfaceMethodCallableReference) Artifacts() []*GeneratedArtifact {
	return slices.Clone(r.artifacts)
}

func (r InterfaceMethodCallableReference) Correspondences() []InterfaceMethodCallableCorrespondence {
	return slices.Clone(r.correspondence)
}

func (r InterfaceMethodCallableReference) Requests() []RootRequest {
	return slices.Clone(r.requests)
}
