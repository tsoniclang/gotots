package semantic

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
)

type normalizedPackageValidator struct {
	pkg           Package
	index         normalizedPackageIndex
	packageRef    packageRef
	authorityUses []bool
}

func validateNormalizedPackageSemantics(pkg Package) error {
	index, err := newNormalizedPackageIndex(pkg)
	if err != nil {
		return err
	}
	packageReference := pkg.identities.packageReference(pkg.id)
	if packageReference == 0 {
		return fmt.Errorf(
			"semantic package identity is absent from normalized storage",
		)
	}
	validator := normalizedPackageValidator{
		pkg: pkg, index: index, packageRef: packageReference,
		authorityUses: make([]bool, len(pkg.authorities.records)+1),
	}
	if err := validator.validateDefinitions(); err != nil {
		return err
	}
	if err := validator.validateDeclarations(); err != nil {
		return err
	}
	if err := validator.validateBindings(); err != nil {
		return err
	}
	if err := validator.validateTypes(); err != nil {
		return err
	}
	if err := validator.validateTypeWitnesses(); err != nil {
		return err
	}
	if err := validator.validateOperations(); err != nil {
		return err
	}
	if err := validator.validateUnsupported(); err != nil {
		return err
	}
	if err := validator.validateResolutions(); err != nil {
		return err
	}
	if err := validator.validateAuthorityConservation(); err != nil {
		return err
	}
	return validateNormalizedPackageReachability(pkg, index)
}

func (validator *normalizedPackageValidator) requireAuthority(
	reference authorityRef,
) error {
	if reference == 0 ||
		uint64(reference) >= uint64(len(validator.authorityUses)) {
		return fmt.Errorf(
			"authority reference %d is invalid", reference,
		)
	}
	if authority, present := validator.pkg.authorities.authority(
		reference,
	); !present || !authority.Valid() {
		return fmt.Errorf(
			"authority reference %d is invalid", reference,
		)
	}
	validator.authorityUses[reference] = true
	return nil
}

func (validator normalizedPackageValidator) validateAuthorityConservation() error {
	for index := 1; index < len(validator.authorityUses); index++ {
		if !validator.authorityUses[index] {
			return fmt.Errorf(
				"semantic authority %d has no record owner", index,
			)
		}
	}
	return nil
}

func (validator normalizedPackageValidator) requireDefinition(
	reference definitionRef,
) error {
	if reference == 0 {
		return nil
	}
	if !referenceInSet(validator.index.definitions, reference) {
		return fmt.Errorf(
			"absent definition %s",
			validator.pkg.identities.definition(reference),
		)
	}
	return nil
}

func (validator normalizedPackageValidator) requireDefinitionIdentity(
	reference definitionRef,
) error {
	if reference == 0 {
		return nil
	}
	if uint64(reference) >
		uint64(len(validator.pkg.identities.definitions)) {
		return fmt.Errorf(
			"definition identity reference %d is invalid", reference,
		)
	}
	return nil
}

func (validator normalizedPackageValidator) requireOccurrence(
	reference occurrenceRef,
) error {
	if reference == 0 {
		return nil
	}
	if !referenceInSet(validator.index.resolutions, reference) {
		return fmt.Errorf(
			"absent occurrence resolution %s",
			validator.pkg.identities.occurrence(reference),
		)
	}
	return nil
}

func (validator normalizedPackageValidator) requireBinding(
	reference bindingRef,
) error {
	if reference == 0 {
		return nil
	}
	if !referenceInSet(validator.index.bindings, reference) {
		return fmt.Errorf(
			"absent binding %s",
			validator.pkg.identities.binding(reference),
		)
	}
	return nil
}

func (validator normalizedPackageValidator) requireType(
	reference typeRef,
) error {
	if reference == 0 {
		return nil
	}
	if !referenceInSet(validator.index.types, reference) {
		return fmt.Errorf(
			"absent type %s",
			validator.pkg.identities.typeID(reference),
		)
	}
	return nil
}

func (validator normalizedPackageValidator) requireOperation(
	reference operationRef,
) error {
	if reference == 0 {
		return nil
	}
	if !referenceInSet(validator.index.operations, reference) {
		return fmt.Errorf(
			"absent operation %s",
			validator.pkg.identities.operation(reference),
		)
	}
	return nil
}

func (validator normalizedPackageValidator) requireUnsupported(
	reference unsupportedRef,
) error {
	if reference == 0 {
		return nil
	}
	if !referenceInSet(validator.index.unsupported, reference) {
		return fmt.Errorf(
			"absent unsupported record %s",
			validator.pkg.identities.unsupportedID(reference),
		)
	}
	return nil
}

func (validator normalizedPackageValidator) requireDeclaration(
	reference declarationRef,
) error {
	if reference == 0 {
		return nil
	}
	id := validator.pkg.identities.declaration(reference)
	if id.IsZero() {
		return fmt.Errorf(
			"declaration identity reference %d is invalid", reference,
		)
	}
	if id.Form() == identity.SemanticDeclarationMember {
		if _, present := validator.pkg.ResolveDeclarationTarget(id); !present {
			return fmt.Errorf(
				"absent semantic member target %s", id,
			)
		}
		return nil
	}
	if declarationIsPackageOwned(id, validator.pkg.id) &&
		!referenceInSet(validator.index.declarations, reference) {
		return fmt.Errorf(
			"absent standalone declaration %s", id,
		)
	}
	return nil
}

func (validator normalizedPackageValidator) requireOwnedDeclaration(
	reference declarationRef,
) error {
	if reference == 0 {
		return nil
	}
	id := validator.pkg.identities.declaration(reference)
	if id.IsZero() {
		return fmt.Errorf(
			"declaration identity reference %d is invalid", reference,
		)
	}
	if id.Form() == identity.SemanticDeclarationMember {
		if _, present := validator.pkg.ResolveDeclarationTarget(id); !present {
			return fmt.Errorf(
				"absent semantic member target %s", id,
			)
		}
		return nil
	}
	if !referenceInSet(validator.index.declarations, reference) {
		return fmt.Errorf(
			"absent owned declaration %s", id,
		)
	}
	return nil
}

func declarationIsPackageOwned(
	declaration identity.SemanticDeclarationID,
	pkg identity.PackageID,
) bool {
	switch declaration.Form() {
	case identity.SemanticDeclarationPackageObject:
		return declaration.Package() == pkg
	case identity.SemanticDeclarationOccurrence:
		return true
	default:
		return false
	}
}

func storedRelation[Value any](
	values []Value,
	start uint64,
	count uint64,
) ([]Value, bool) {
	if start > uint64(len(values)) ||
		count > uint64(len(values))-start {
		return nil, false
	}
	return values[start : start+count], true
}
