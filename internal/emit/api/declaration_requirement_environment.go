package api

import "go/types"

func NewEnvironmentBuiltinRequirement(
	builtin *types.Builtin,
	signature *types.Signature,
) (DeclarationRequirement, error) {
	if builtin == nil ||
		builtin.Pkg() == nil ||
		builtin.Parent() != builtin.Pkg().Scope() ||
		builtin.Parent().Lookup(builtin.Name()) != builtin ||
		!validEnvironmentBuiltinSignature(signature) {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "environment-builtin requirement is invalid",
		}
	}
	return DeclarationRequirement{
		owner:                MustSourceArtifactOwner(builtin),
		kind:                 DeclarationRequirementEnvironmentBuiltin,
		environmentBuiltin:   builtin,
		environmentSignature: signature,
	}, nil
}

func NewEnvironmentBuiltinRequest(
	builtin *types.Builtin,
	signature *types.Signature,
) (RootRequest, error) {
	requirement, err := NewEnvironmentBuiltinRequirement(
		builtin,
		signature,
	)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func (r DeclarationRequirement) EnvironmentBuiltin() (
	*types.Builtin,
	*types.Signature,
	bool,
) {
	if !r.Valid() ||
		r.kind != DeclarationRequirementEnvironmentBuiltin {
		return nil, nil, false
	}
	return r.environmentBuiltin, r.environmentSignature, true
}

func validEnvironmentBuiltinSignature(signature *types.Signature) bool {
	return signature != nil &&
		signature.Recv() == nil &&
		signature.RecvTypeParams().Len() == 0 &&
		signature.TypeParams().Len() == 0 &&
		signature.Params() != nil &&
		signature.Results() != nil
}
