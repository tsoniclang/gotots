package semantic

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
)

func (pkg Package) VisitDefinitions(
	visit func(DefinitionSemantics) error,
) error {
	if visit == nil {
		return fmt.Errorf("semantic definition visitor is absent")
	}
	identities := newPackageIdentityProjection(pkg.identities)
	for index := range pkg.definitions.records {
		record, err := pkg.definitions.record(
			identities, pkg.authorities, index,
		)
		if err != nil {
			return err
		}
		if err := visit(record); err != nil {
			return err
		}
	}
	return nil
}

func (pkg Package) VisitResolutions(
	visit func(OccurrenceResolution) error,
) error {
	if visit == nil {
		return fmt.Errorf("semantic resolution visitor is absent")
	}
	identities := newPackageIdentityProjection(pkg.identities)
	for index := range pkg.resolutions.records {
		record, err := pkg.resolutions.record(
			identities, index,
		)
		if err != nil {
			return err
		}
		if err := visit(record); err != nil {
			return err
		}
	}
	return nil
}

func (pkg Package) VisitDeclarations(
	visit func(Declaration) error,
) error {
	if visit == nil {
		return fmt.Errorf("semantic declaration visitor is absent")
	}
	identities := newPackageIdentityProjection(pkg.identities)
	for index := range pkg.declarations.records {
		record, err := pkg.declarations.record(
			identities, pkg.authorities, index,
		)
		if err != nil {
			return err
		}
		if err := visit(record); err != nil {
			return err
		}
	}
	return nil
}

func (pkg Package) VisitBindings(visit func(Binding) error) error {
	if visit == nil {
		return fmt.Errorf("semantic binding visitor is absent")
	}
	identities := newPackageIdentityProjection(pkg.identities)
	for index := range pkg.bindings.records {
		record, err := pkg.bindings.record(
			identities, pkg.authorities, index,
		)
		if err != nil {
			return err
		}
		if err := visit(record); err != nil {
			return err
		}
	}
	return nil
}

func (pkg Package) VisitTypes(visit func(Type) error) error {
	if visit == nil {
		return fmt.Errorf("semantic type visitor is absent")
	}
	identities := newPackageIdentityProjection(pkg.identities)
	for index := range pkg.types.records {
		record, err := pkg.types.record(identities, index)
		if err != nil {
			return err
		}
		if err := visit(record); err != nil {
			return err
		}
	}
	return nil
}

func (pkg Package) VisitTypeWitnesses(
	visit func(TypeWitness) error,
) error {
	if visit == nil {
		return fmt.Errorf("semantic type-witness visitor is absent")
	}
	identities := newPackageIdentityProjection(pkg.identities)
	for index := range pkg.witnesses.records {
		record, err := pkg.witnesses.record(
			identities, pkg.authorities, index,
		)
		if err != nil {
			return err
		}
		if err := visit(record); err != nil {
			return err
		}
	}
	return nil
}

func (pkg Package) VisitOperations(visit func(Operation) error) error {
	if visit == nil {
		return fmt.Errorf("semantic operation visitor is absent")
	}
	identities := newPackageIdentityProjection(pkg.identities)
	for index := range pkg.operations.records {
		record, err := pkg.operations.operation(
			identities, index,
		)
		if err != nil {
			return err
		}
		if err := visit(record); err != nil {
			return err
		}
	}
	return nil
}

func (pkg Package) VisitUnsupported(
	visit func(Unsupported) error,
) error {
	if visit == nil {
		return fmt.Errorf("semantic unsupported visitor is absent")
	}
	identities := newPackageIdentityProjection(pkg.identities)
	for index := range pkg.unsupported.records {
		record, err := pkg.unsupported.record(
			identities, pkg.authorities, index,
		)
		if err != nil {
			return err
		}
		if err := visit(record); err != nil {
			return err
		}
	}
	return nil
}

func (pkg Package) Definition(
	id identity.DefinitionID,
) (DefinitionSemantics, bool) {
	index, present := pkg.definitionIndex(id)
	if !present {
		return DefinitionSemantics{}, false
	}
	identities := newPackageIdentityProjection(pkg.identities)
	record, err := pkg.definitions.record(
		identities, pkg.authorities, index,
	)
	return record, err == nil
}

func (pkg Package) Resolution(
	id identity.OccurrenceID,
) (OccurrenceResolution, bool) {
	index, present := pkg.resolutionIndex(id)
	if !present {
		return OccurrenceResolution{}, false
	}
	identities := newPackageIdentityProjection(pkg.identities)
	record, err := pkg.resolutions.record(identities, index)
	return record, err == nil
}

func (pkg Package) Declaration(
	id identity.SemanticDeclarationID,
) (Declaration, bool) {
	index, present := pkg.declarationIndex(id)
	if !present {
		return Declaration{}, false
	}
	identities := newPackageIdentityProjection(pkg.identities)
	record, err := pkg.declarations.record(
		identities, pkg.authorities, index,
	)
	return record, err == nil
}

func (pkg Package) Binding(
	id identity.SemanticBindingID,
) (Binding, bool) {
	index, present := pkg.bindingIndex(id)
	if !present {
		return Binding{}, false
	}
	identities := newPackageIdentityProjection(pkg.identities)
	record, err := pkg.bindings.record(
		identities, pkg.authorities, index,
	)
	return record, err == nil
}

func (pkg Package) Type(
	id identity.SemanticTypeID,
) (Type, bool) {
	index, present := pkg.typeIndex(id)
	if !present {
		return Type{}, false
	}
	identities := newPackageIdentityProjection(pkg.identities)
	record, err := pkg.types.record(identities, index)
	return record, err == nil
}

func (pkg Package) TypeWitness(
	id identity.SemanticTypeID,
) (TypeWitness, bool) {
	index, present := pkg.typeWitnessIndex(id)
	if !present {
		return TypeWitness{}, false
	}
	identities := newPackageIdentityProjection(pkg.identities)
	record, err := pkg.witnesses.record(
		identities, pkg.authorities, index,
	)
	return record, err == nil
}

func (pkg Package) Operation(
	id identity.OperationID,
) (Operation, bool) {
	index, present := pkg.operationIndex(id)
	if !present {
		return Operation{}, false
	}
	identities := newPackageIdentityProjection(pkg.identities)
	record, err := pkg.operations.operation(identities, index)
	return record, err == nil
}

func (pkg Package) UnsupportedRecord(
	id identity.UnsupportedID,
) (Unsupported, bool) {
	index, present := pkg.unsupportedIndex(id)
	if !present {
		return Unsupported{}, false
	}
	identities := newPackageIdentityProjection(pkg.identities)
	record, err := pkg.unsupported.record(
		identities, pkg.authorities, index,
	)
	return record, err == nil
}
