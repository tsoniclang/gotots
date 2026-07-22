package analyze

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
)

// Contract is the closed declaration contract a parent retains at an
// implementation edge — the typed obligation preserved when a child body is
// excised. It is the "contract owner" column of the region/reference model: a
// non-full child still preserves its parent's callable, initializer, or
// obligation contract at the exact edge, so stopping at a FuncLit, bodyless
// declaration, or package ValueSpec never silently removes the contract.
type Contract uint8

const (
	// ContractInvalid is the zero value; an unclassified contract fails.
	ContractInvalid Contract = iota
	// ContractDeclarationSignature: a function/method declaration or bodyless
	// declaration — the parent retains the declaration and its signature.
	ContractDeclarationSignature
	// ContractCallableSignature: a function literal — the parent retains the
	// function-value operation and its callable signature at the edge.
	ContractCallableSignature
	// ContractInitializer: a package ValueSpec with values — the parent retains
	// the names/type declaration and the ordered initializer at the edge.
	ContractInitializer
	// ContractCatalogOwner: implicit executable work — the owning catalog/type
	// operation retains it.
	ContractCatalogOwner

	numContracts
)

var contractNames = [numContracts]string{
	ContractDeclarationSignature: "declaration-signature",
	ContractCallableSignature:    "callable-signature",
	ContractInitializer:          "initializer",
	ContractCatalogOwner:         "catalog-owner",
}

// Valid reports whether c names a contract.
func (c Contract) Valid() bool { return c > ContractInvalid && c < numContracts }

// String renders c for reports and digests.
func (c Contract) String() string {
	if c.Valid() {
		return contractNames[c]
	}
	return fmt.Sprintf("analyze.Contract(%d)", uint8(c))
}

// ContractForKind is the single owner of the kind→contract map. Every unit kind
// has exactly one declaration contract; an unknown kind fails closed.
func ContractForKind(kind identity.UnitKind) (Contract, error) {
	switch kind {
	case identity.UnitFuncBody, identity.UnitBodylessDecl:
		return ContractDeclarationSignature, nil
	case identity.UnitFuncLitBody:
		return ContractCallableSignature, nil
	case identity.UnitVarInitializer:
		return ContractInitializer, nil
	case identity.UnitImplicitExecutable:
		return ContractCatalogOwner, nil
	default:
		return ContractInvalid, fmt.Errorf("no declaration contract for unit kind %s", kind)
	}
}
