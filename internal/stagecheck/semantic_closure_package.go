package stagecheck

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/semantic"
)

type semanticPackageClosure struct {
	packageID identity.PackageID
	owners    *semanticOwnerCensus
	pkg       semantic.Package
}

func verifySemanticPackageClosure(
	pkg semantic.Package,
	owners *semanticOwnerCensus,
) error {
	closure := semanticPackageClosure{
		packageID: pkg.ID(),
		owners:    owners,
		pkg:       pkg,
	}
	if err := closure.verifyOwners(); err != nil {
		return err
	}
	if err := closure.verifyDefinitions(); err != nil {
		return err
	}
	if err := closure.verifyDeclarations(); err != nil {
		return err
	}
	if err := closure.verifyBindings(); err != nil {
		return err
	}
	if err := closure.verifyTypes(); err != nil {
		return err
	}
	if err := closure.verifyTypeWitnesses(); err != nil {
		return err
	}
	if err := closure.verifyOperations(); err != nil {
		return err
	}
	if err := closure.verifyUnsupported(); err != nil {
		return err
	}
	return closure.verifyResolutions()
}

func (closure semanticPackageClosure) verifyOwners() error {
	memberTargets, err := closure.pkg.MemberTargetCensus()
	if err != nil {
		return err
	}
	if memberTargets.Count() !=
		closure.owners.memberCounts[closure.packageID] ||
		memberTargets.Digest() !=
			closure.owners.memberDigests[closure.packageID] {
		return fmt.Errorf(
			"package %s member-target census disagrees with its manifest",
			closure.packageID,
		)
	}
	if err := closure.pkg.VisitDefinitions(func(
		record semantic.DefinitionSemantics,
	) error {
		owner, present := closure.owners.definitions[record.Definition()]
		if !present || owner != closure.packageID {
			return fmt.Errorf(
				"definition %s has invalid package owner",
				record.Definition(),
			)
		}
		return nil
	}); err != nil {
		return err
	}
	return closure.pkg.VisitDeclarations(func(
		record semantic.Declaration,
	) error {
		owner, present := closure.owners.declarations[record.ID()]
		if !present || owner != closure.packageID {
			return fmt.Errorf(
				"declaration %s has invalid package owner",
				record.ID(),
			)
		}
		return nil
	})
}

func (closure semanticPackageClosure) verifyDefinitions() error {
	return closure.pkg.VisitDefinitions(func(
		record semantic.DefinitionSemantics,
	) error {
		evidence := record.Definition()
		spec := record.Spec()
		for _, declaration := range spec.Declarations {
			if err := closure.requireDeclaration(declaration); err != nil {
				return semanticReferenceError(evidence, err)
			}
		}
		if err := closure.requireType(spec.Signature); err != nil {
			return semanticReferenceError(evidence, err)
		}
		if err := closure.requireBinding(spec.Receiver); err != nil {
			return semanticReferenceError(evidence, err)
		}
		for _, binding := range spec.Bindings {
			if err := closure.requireBinding(binding); err != nil {
				return semanticReferenceError(evidence, err)
			}
		}
		for _, occurrence := range spec.InitializerEntries {
			if err := closure.requireOccurrence(occurrence); err != nil {
				return semanticReferenceError(evidence, err)
			}
		}
		return nil
	})
}

func (closure semanticPackageClosure) verifyDeclarations() error {
	return closure.pkg.VisitDeclarations(func(
		record semantic.Declaration,
	) error {
		if err := closure.requireType(record.Type()); err != nil {
			return semanticReferenceError(record.ID(), err)
		}
		return nil
	})
}

func (closure semanticPackageClosure) verifyBindings() error {
	return closure.pkg.VisitBindings(func(
		record semantic.Binding,
	) error {
		evidence := record.ID()
		if err := closure.requireDefinition(record.Definition()); err != nil {
			return semanticReferenceError(evidence, err)
		}
		if err := closure.requireType(record.Type()); err != nil {
			return semanticReferenceError(evidence, err)
		}
		for _, definition := range record.CapturedBy() {
			if err := closure.requireDefinition(definition); err != nil {
				return semanticReferenceError(evidence, err)
			}
		}
		return nil
	})
}

func (closure semanticPackageClosure) verifyTypes() error {
	return closure.pkg.VisitTypes(func(record semantic.Type) error {
		if err := closure.verifyType(record); err != nil {
			return err
		}
		return nil
	})
}

func (closure semanticPackageClosure) verifyType(
	record semantic.Type,
) error {
	spec := record.Spec()
	evidence := record.ID()
	if err := closure.requireDeclaration(spec.Declaration); err != nil {
		return semanticReferenceError(evidence, err)
	}
	if err := closure.requireDeclaration(
		spec.Parameter.Declaration(),
	); err != nil {
		return semanticReferenceError(evidence, err)
	}
	if err := closure.requireDefinition(
		spec.Parameter.Definition(),
	); err != nil {
		return semanticReferenceError(evidence, err)
	}
	typeIDs := []identity.SemanticTypeID{
		spec.Underlying,
		spec.Target,
		spec.Constraint,
		spec.Element,
		spec.Key,
		spec.Signature.Receiver,
	}
	typeIDs = append(typeIDs, spec.Arguments...)
	typeIDs = append(
		typeIDs, spec.Signature.ReceiverTypeParameters...,
	)
	typeIDs = append(
		typeIDs, spec.Signature.TypeParameters...,
	)
	typeIDs = append(typeIDs, spec.Signature.Parameters...)
	typeIDs = append(typeIDs, spec.Signature.Results...)
	typeIDs = append(typeIDs, spec.Embeddeds...)
	typeIDs = append(typeIDs, spec.Elements...)
	for _, typeID := range typeIDs {
		if err := closure.requireType(typeID); err != nil {
			return semanticReferenceError(evidence, err)
		}
	}
	for _, field := range spec.Fields {
		if err := closure.requireType(field.Type); err != nil {
			return semanticReferenceError(evidence, err)
		}
	}
	for _, method := range spec.Methods {
		if err := closure.requireType(method.Signature); err != nil {
			return semanticReferenceError(evidence, err)
		}
	}
	for _, term := range spec.Terms {
		if err := closure.requireType(term.Type); err != nil {
			return semanticReferenceError(evidence, err)
		}
	}
	return nil
}

func (closure semanticPackageClosure) verifyTypeWitnesses() error {
	return closure.pkg.VisitTypeWitnesses(func(
		witness semantic.TypeWitness,
	) error {
		if witness.Package() != closure.packageID {
			return fmt.Errorf(
				"type witness %s has invalid package owner",
				witness.Type(),
			)
		}
		if err := closure.requireType(witness.Type()); err != nil {
			return semanticReferenceError(witness.Type(), err)
		}
		return nil
	})
}

func (closure semanticPackageClosure) verifyOperations() error {
	return closure.pkg.VisitOperations(func(
		record semantic.Operation,
	) error {
		evidence := record.ID()
		spec := record.Spec()
		if err := closure.requireDefinition(record.Definition()); err != nil {
			return semanticReferenceError(evidence, err)
		}
		if err := closure.requireType(spec.ResultType); err != nil {
			return semanticReferenceError(evidence, err)
		}
		if err := closure.requireType(spec.ExpectedType); err != nil {
			return semanticReferenceError(evidence, err)
		}
		if err := closure.requireObject(spec.Object); err != nil {
			return semanticReferenceError(evidence, err)
		}
		if !spec.Selection.IsZero() {
			if err := closure.requireType(
				spec.Selection.Receiver(),
			); err != nil {
				return semanticReferenceError(evidence, err)
			}
			if err := closure.requireDeclaration(
				spec.Selection.Object(),
			); err != nil {
				return semanticReferenceError(evidence, err)
			}
		}
		if !spec.Instance.IsZero() {
			if err := closure.requireObject(
				spec.Instance.Target(),
			); err != nil {
				return semanticReferenceError(evidence, err)
			}
			for _, typeID := range spec.Instance.Types() {
				if err := closure.requireType(typeID); err != nil {
					return semanticReferenceError(evidence, err)
				}
			}
			if err := closure.requireType(
				spec.Instance.Signature(),
			); err != nil {
				return semanticReferenceError(evidence, err)
			}
		}
		for _, occurrence := range spec.Operands {
			if err := closure.requireOccurrence(occurrence); err != nil {
				return semanticReferenceError(evidence, err)
			}
		}
		for _, definition := range spec.Definitions {
			if err := closure.requireDefinition(definition); err != nil {
				return semanticReferenceError(evidence, err)
			}
		}
		for _, implicit := range spec.Implicit {
			if err := closure.requireOccurrence(
				implicit.Site(),
			); err != nil {
				return semanticReferenceError(evidence, err)
			}
			if err := closure.requireType(
				implicit.Source(),
			); err != nil {
				return semanticReferenceError(evidence, err)
			}
			if err := closure.requireType(
				implicit.Target(),
			); err != nil {
				return semanticReferenceError(evidence, err)
			}
		}
		if err := closure.requireOperation(
			spec.ControlTarget,
		); err != nil {
			return semanticReferenceError(evidence, err)
		}
		if err := closure.requireBinding(
			spec.Label,
		); err != nil {
			return semanticReferenceError(evidence, err)
		}
		return nil
	})
}

func (closure semanticPackageClosure) verifyUnsupported() error {
	return closure.pkg.VisitUnsupported(func(
		record semantic.Unsupported,
	) error {
		if err := closure.requireDefinition(
			record.ID().Definition(),
		); err != nil {
			return semanticReferenceError(record.ID(), err)
		}
		return nil
	})
}

func (closure semanticPackageClosure) verifyResolutions() error {
	return closure.pkg.VisitResolutions(func(
		record semantic.OccurrenceResolution,
	) error {
		evidence := record.Occurrence()
		switch record.Kind() {
		case semantic.ResolutionStructuralOnly:
			if err := closure.requireDeclaration(
				record.Structural().Declaration(),
			); err != nil {
				return semanticReferenceError(evidence, err)
			}
			if err := closure.requireType(
				record.Structural().Type(),
			); err != nil {
				return semanticReferenceError(evidence, err)
			}
		case semantic.ResolutionDefinitionComponent:
			if err := closure.requireDefinition(
				record.Definition(),
			); err != nil {
				return semanticReferenceError(evidence, err)
			}
		case semantic.ResolutionDeclaration:
			if err := closure.requireDeclaration(
				record.Declaration(),
			); err != nil {
				return semanticReferenceError(evidence, err)
			}
		case semantic.ResolutionBinding:
			if err := closure.requireBinding(
				record.Binding(),
			); err != nil {
				return semanticReferenceError(evidence, err)
			}
		case semantic.ResolutionType:
			if err := closure.requireType(
				record.Type(),
			); err != nil {
				return semanticReferenceError(evidence, err)
			}
		case semantic.ResolutionOperation:
			if err := closure.requireOperation(
				record.Operation(),
			); err != nil {
				return semanticReferenceError(evidence, err)
			}
		case semantic.ResolutionUnsupported:
			if err := closure.requireUnsupported(
				record.Unsupported(),
			); err != nil {
				return semanticReferenceError(evidence, err)
			}
		}
		return nil
	})
}

func (closure semanticPackageClosure) requireObject(
	reference semantic.ObjectReference,
) error {
	switch reference.Kind() {
	case semantic.ObjectReferenceDeclaration:
		return closure.requireDeclaration(reference.Declaration())
	case semantic.ObjectReferenceBinding:
		return closure.requireBinding(reference.Binding())
	default:
		return nil
	}
}

func (closure semanticPackageClosure) requireDefinition(
	id identity.DefinitionID,
) error {
	if id.IsZero() {
		return nil
	}
	if _, present := closure.owners.definitions[id]; !present {
		return absentSemanticTarget(id)
	}
	return nil
}

func (closure semanticPackageClosure) requireDeclaration(
	id identity.SemanticDeclarationID,
) error {
	if id.IsZero() {
		return nil
	}
	if id.Form() == identity.SemanticDeclarationMember {
		if _, present := closure.pkg.ResolveDeclarationTarget(id); !present {
			return absentSemanticTarget(id)
		}
		return nil
	}
	if _, present := closure.owners.declarations[id]; !present {
		return absentSemanticTarget(id)
	}
	return nil
}

func (closure semanticPackageClosure) requireBinding(
	id identity.SemanticBindingID,
) error {
	if id.IsZero() {
		return nil
	}
	if !closure.pkg.HasBinding(id) {
		return absentSemanticTarget(id)
	}
	return nil
}

func (closure semanticPackageClosure) requireType(
	id identity.SemanticTypeID,
) error {
	if id.IsZero() {
		return nil
	}
	if !closure.pkg.HasType(id) {
		return absentSemanticTarget(id)
	}
	return nil
}

func (closure semanticPackageClosure) requireOperation(
	id identity.OperationID,
) error {
	if id.IsZero() {
		return nil
	}
	if !closure.pkg.HasOperation(id) {
		return absentSemanticTarget(id)
	}
	return nil
}

func (closure semanticPackageClosure) requireUnsupported(
	id identity.UnsupportedID,
) error {
	if id.IsZero() {
		return nil
	}
	if !closure.pkg.HasUnsupported(id) {
		return absentSemanticTarget(id)
	}
	return nil
}

func (closure semanticPackageClosure) requireOccurrence(
	id identity.OccurrenceID,
) error {
	if id.IsZero() {
		return nil
	}
	if !closure.pkg.HasResolution(id) {
		return absentSemanticTarget(id)
	}
	return nil
}

func semanticReferenceError[Evidence fmt.Stringer](
	evidence Evidence,
	err error,
) error {
	return fmt.Errorf("%s references %w", evidence, err)
}

func absentSemanticTarget[Target fmt.Stringer](target Target) error {
	return fmt.Errorf("absent semantic target %s", target)
}
