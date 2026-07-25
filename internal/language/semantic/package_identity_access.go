package semantic

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
)

// VisitDeclarationIdentities visits the canonical declaration dictionary
// owned by this normalized package.
func (pkg Package) VisitDeclarationIdentities(
	visit func(identity.SemanticDeclarationID) error,
) error {
	if visit == nil {
		return fmt.Errorf("semantic declaration-identity visitor is absent")
	}
	identities := newPackageIdentityProjection(pkg.identities)
	for index := range pkg.identities.declarations {
		id := identities.declaration(declarationRef(index + 1))
		if id.IsZero() {
			return fmt.Errorf(
				"semantic declaration identity %d is invalid",
				index+1,
			)
		}
		if err := visit(id); err != nil {
			return err
		}
	}
	return nil
}
