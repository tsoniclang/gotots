package semantic

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
)

func validatePackage(pkg Package) error {
	definitions := map[identity.DefinitionID]bool{}
	for _, record := range pkg.definitions {
		if record.Package() != pkg.id ||
			definitions[record.Definition()] {
			return fmt.Errorf(
				"semantic package %s has invalid definition %s",
				pkg.id, record.Definition(),
			)
		}
		definitions[record.Definition()] = true
	}
	declarations := map[identity.SemanticDeclarationID]bool{}
	for _, record := range pkg.declarations {
		if record.ID().IsZero() ||
			record.Package() != pkg.id ||
			declarations[record.ID()] {
			return fmt.Errorf(
				"semantic package %s has duplicate declaration %s",
				pkg.id, record.ID(),
			)
		}
		declarations[record.ID()] = true
	}
	for _, record := range pkg.definitions {
		for _, declaration := range record.Spec().Declarations {
			if !declarations[declaration] {
				return fmt.Errorf(
					"definition %s references absent semantic declaration %s",
					record.Definition(), declaration,
				)
			}
		}
	}
	bindings := map[identity.SemanticBindingID]bool{}
	for _, record := range pkg.bindings {
		if record.Package() != pkg.id {
			return fmt.Errorf(
				"semantic package %s does not own binding %s",
				pkg.id, record.ID(),
			)
		}
		if !record.Definition().IsZero() &&
			!definitions[record.Definition()] {
			return fmt.Errorf(
				"semantic binding %s references absent definition %s",
				record.ID(), record.Definition(),
			)
		}
		if bindings[record.ID()] {
			return fmt.Errorf(
				"semantic package %s duplicates binding %s",
				pkg.id, record.ID(),
			)
		}
		bindings[record.ID()] = true
	}
	types := map[identity.SemanticTypeID]string{}
	for _, record := range pkg.types {
		if existing, duplicate := types[record.ID()]; duplicate {
			if existing != record.Canonical() {
				return fmt.Errorf(
					"semantic package %s has type collision %s",
					pkg.id, record.ID(),
				)
			}
			return fmt.Errorf(
				"semantic package %s duplicates type %s",
				pkg.id, record.ID(),
			)
		}
		types[record.ID()] = record.Canonical()
	}
	witnesses := map[identity.SemanticTypeID]bool{}
	for _, record := range pkg.typeWitnesses {
		if record.Package() != pkg.id ||
			types[record.Type()] == "" ||
			witnesses[record.Type()] {
			return fmt.Errorf(
				"semantic package %s has invalid type witness %s",
				pkg.id, record.Type(),
			)
		}
		witnesses[record.Type()] = true
	}
	if len(witnesses) != len(types) {
		return fmt.Errorf(
			"semantic package %s has %d types and %d witnesses",
			pkg.id, len(types), len(witnesses),
		)
	}
	operations := map[identity.OperationID]bool{}
	for _, record := range pkg.operations {
		if !definitions[record.Definition()] {
			return fmt.Errorf(
				"semantic operation %s references absent definition %s",
				record.ID(), record.Definition(),
			)
		}
		if operations[record.ID()] {
			return fmt.Errorf(
				"semantic package %s duplicates operation %s",
				pkg.id, record.ID(),
			)
		}
		operations[record.ID()] = true
	}
	unsupported := map[identity.UnsupportedID]bool{}
	for _, record := range pkg.unsupported {
		if !definitions[record.ID().Definition()] ||
			unsupported[record.ID()] {
			return fmt.Errorf(
				"semantic package %s has invalid unsupported record %s",
				pkg.id, record.ID(),
			)
		}
		unsupported[record.ID()] = true
	}
	resolutions := map[identity.OccurrenceID]bool{}
	for _, record := range pkg.resolutions {
		if resolutions[record.Occurrence()] {
			return fmt.Errorf(
				"semantic package %s has duplicate resolution %s",
				pkg.id, record.Occurrence(),
			)
		}
		resolutions[record.Occurrence()] = true
		switch record.Kind() {
		case ResolutionStructuralOnly:
			structural := record.Structural()
			if declarationIsPackageOwned(
				structural.Declaration(), pkg.id,
			) && !declarations[structural.Declaration()] {
				return fmt.Errorf(
					"resolution %s has absent structural declaration coverage",
					record.Occurrence(),
				)
			}
			if !structural.Type().IsZero() &&
				types[structural.Type()] == "" {
				return fmt.Errorf(
					"resolution %s has absent structural type coverage",
					record.Occurrence(),
				)
			}
		case ResolutionDefinitionComponent:
			if !definitions[record.Definition()] {
				return fmt.Errorf(
					"resolution %s names absent definition",
					record.Occurrence(),
				)
			}
		case ResolutionDeclaration:
			if declarationIsPackageOwned(
				record.Declaration(), pkg.id,
			) && !declarations[record.Declaration()] {
				return fmt.Errorf(
					"resolution %s names absent declaration",
					record.Occurrence(),
				)
			}
		case ResolutionBinding:
			if !bindings[record.Binding()] {
				return fmt.Errorf(
					"resolution %s names absent binding",
					record.Occurrence(),
				)
			}
		case ResolutionType:
			if types[record.Type()] == "" {
				return fmt.Errorf(
					"resolution %s names absent type",
					record.Occurrence(),
				)
			}
		case ResolutionOperation:
			if !operations[record.Operation()] {
				return fmt.Errorf(
					"resolution %s names absent operation",
					record.Occurrence(),
				)
			}
		case ResolutionUnsupported:
			if !unsupported[record.Unsupported()] {
				return fmt.Errorf(
					"resolution %s names absent unsupported record",
					record.Occurrence(),
				)
			}
		}
	}
	if err := validateTypeRecords(pkg.types, types); err != nil {
		return err
	}
	if err := validateTypeClosure([]Package{pkg}, types); err != nil {
		return err
	}
	return validatePackageReachability(pkg)
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
