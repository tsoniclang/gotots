package api

import "go/types"

type DeclarationDemandResolver interface {
	NamedStructOperationSelected(
		*types.TypeName,
		NamedStructOperation,
	) (bool, error)
	AnonymousStructDemandSelected(
		*GeneratedArtifact,
		AnonymousStructDemand,
	) (bool, error)
}

func (c Context) WithDeclarationDemandResolver(
	resolver DeclarationDemandResolver,
) Context {
	if resolver == nil {
		panic("declaration demand resolver is nil")
	}
	c.declarationDemandResolver = resolver
	return c
}

func (c Context) ResolveNamedStructOperation(
	owner *types.TypeName,
	operation NamedStructOperation,
) (bool, error) {
	if owner == nil || owner.Pkg() == nil || owner.Parent() == nil ||
		!operation.Valid() {
		return false, &ContextError{
			Reason: "named-struct operation selection is invalid",
		}
	}
	if owner.Parent() != owner.Pkg().Scope() {
		for _, requirement := range c.lexicalTypeRequirements[owner] {
			selectedOwner, selectedOperation, ok :=
				requirement.NamedStructOperation()
			if ok && selectedOwner == owner && selectedOperation == operation {
				return true, nil
			}
		}
		return false, nil
	}
	if c.declarationDemandResolver == nil {
		return false, &ContextError{
			Reason: "declaration demand resolver is unavailable",
		}
	}
	return c.declarationDemandResolver.NamedStructOperationSelected(
		owner,
		operation,
	)
}

func (c Context) ResolveAnonymousStructDemand(
	artifact *GeneratedArtifact,
	demand AnonymousStructDemand,
) (bool, error) {
	if !artifact.Valid() ||
		artifact.Kind() != GeneratedArtifactAnonymousStruct ||
		!demand.Valid() {
		return false, &ContextError{
			Reason: "anonymous-struct demand selection is invalid",
		}
	}
	if c.declarationDemandResolver == nil {
		return false, &ContextError{
			Reason: "declaration demand resolver is unavailable",
		}
	}
	return c.declarationDemandResolver.AnonymousStructDemandSelected(
		artifact,
		demand,
	)
}
