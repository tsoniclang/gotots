package stagecheck

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/semantic"
)

type semanticPackageClosure struct {
	packageID    identity.PackageID
	owners       *semanticOwnerCensus
	definitions  []semantic.DefinitionSemantics
	resolutions  []semantic.OccurrenceResolution
	declarations []semantic.Declaration
	bindings     []semantic.Binding
	types        []semantic.Type
	witnesses    []semantic.TypeWitness
	operations   []semantic.Operation
	unsupported  []semantic.Unsupported
}

func verifySemanticPackageClosure(
	pkg semantic.Package,
	owners *semanticOwnerCensus,
) error {
	closure := semanticPackageClosure{
		packageID:    pkg.ID(),
		owners:       owners,
		definitions:  pkg.Definitions(),
		resolutions:  pkg.Resolutions(),
		declarations: pkg.Declarations(),
		bindings:     pkg.Bindings(),
		types:        pkg.Types(),
		witnesses:    pkg.TypeWitnesses(),
		operations:   pkg.Operations(),
		unsupported:  pkg.Unsupported(),
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
	for _, record := range closure.definitions {
		owner, present := closure.owners.definitions[record.Definition()]
		if !present || owner != closure.packageID {
			return fmt.Errorf(
				"definition %s has invalid package owner",
				record.Definition(),
			)
		}
	}
	for _, record := range closure.declarations {
		owner, present := closure.owners.declarations[record.ID()]
		if !present || owner != closure.packageID {
			return fmt.Errorf(
				"declaration %s has invalid package owner",
				record.ID(),
			)
		}
	}
	return nil
}

func (closure semanticPackageClosure) verifyDefinitions() error {
	for _, record := range closure.definitions {
		evidence := record.Definition().String()
		spec := record.Spec()
		for _, declaration := range spec.Declarations {
			if err := closure.requireDeclaration(
				evidence, declaration,
			); err != nil {
				return err
			}
		}
		if err := closure.requireType(
			evidence, spec.Signature,
		); err != nil {
			return err
		}
		if err := closure.requireBinding(
			evidence, spec.Receiver,
		); err != nil {
			return err
		}
		for _, binding := range spec.Bindings {
			if err := closure.requireBinding(
				evidence, binding,
			); err != nil {
				return err
			}
		}
		for _, occurrence := range spec.InitializerEntries {
			if err := closure.requireOccurrence(
				evidence, occurrence,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func (closure semanticPackageClosure) verifyDeclarations() error {
	for _, record := range closure.declarations {
		if err := closure.requireType(
			record.ID().String(), record.Type(),
		); err != nil {
			return err
		}
	}
	return nil
}

func (closure semanticPackageClosure) verifyBindings() error {
	for _, record := range closure.bindings {
		evidence := record.ID().String()
		if err := closure.requireDefinition(
			evidence, record.Definition(),
		); err != nil {
			return err
		}
		if err := closure.requireType(
			evidence, record.Type(),
		); err != nil {
			return err
		}
		for _, definition := range record.CapturedBy() {
			if err := closure.requireDefinition(
				evidence, definition,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func (closure semanticPackageClosure) verifyTypes() error {
	for _, record := range closure.types {
		if err := closure.verifyType(record); err != nil {
			return err
		}
	}
	return nil
}

func (closure semanticPackageClosure) verifyType(
	record semantic.Type,
) error {
	spec := record.Spec()
	evidence := record.ID().String()
	if err := closure.requireDeclaration(
		evidence, spec.Declaration,
	); err != nil {
		return err
	}
	if err := closure.requireDeclaration(
		evidence, spec.Parameter.Declaration(),
	); err != nil {
		return err
	}
	if err := closure.requireDefinition(
		evidence, spec.Parameter.Definition(),
	); err != nil {
		return err
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
		if err := closure.requireType(
			evidence, typeID,
		); err != nil {
			return err
		}
	}
	for _, field := range spec.Fields {
		if err := closure.requireType(
			evidence, field.Type,
		); err != nil {
			return err
		}
	}
	for _, method := range spec.Methods {
		if err := closure.requireType(
			evidence, method.Signature,
		); err != nil {
			return err
		}
	}
	for _, term := range spec.Terms {
		if err := closure.requireType(
			evidence, term.Type,
		); err != nil {
			return err
		}
	}
	return nil
}

func (closure semanticPackageClosure) verifyTypeWitnesses() error {
	for _, witness := range closure.witnesses {
		if witness.Package() != closure.packageID {
			return fmt.Errorf(
				"type witness %s has invalid package owner",
				witness.Type(),
			)
		}
		if err := closure.requireType(
			witness.Type().String(), witness.Type(),
		); err != nil {
			return err
		}
	}
	return nil
}

func (closure semanticPackageClosure) verifyOperations() error {
	for _, record := range closure.operations {
		evidence := record.ID().String()
		spec := record.Spec()
		if err := closure.requireDefinition(
			evidence, record.Definition(),
		); err != nil {
			return err
		}
		if err := closure.requireType(
			evidence, spec.ResultType,
		); err != nil {
			return err
		}
		if err := closure.requireType(
			evidence, spec.ExpectedType,
		); err != nil {
			return err
		}
		if err := closure.requireObject(
			evidence, spec.Object,
		); err != nil {
			return err
		}
		if !spec.Selection.IsZero() {
			if err := closure.requireType(
				evidence, spec.Selection.Receiver(),
			); err != nil {
				return err
			}
			if err := closure.requireDeclaration(
				evidence, spec.Selection.Object(),
			); err != nil {
				return err
			}
		}
		if !spec.Instance.IsZero() {
			if err := closure.requireObject(
				evidence, spec.Instance.Target(),
			); err != nil {
				return err
			}
			for _, typeID := range spec.Instance.Types() {
				if err := closure.requireType(
					evidence, typeID,
				); err != nil {
					return err
				}
			}
			if err := closure.requireType(
				evidence, spec.Instance.Signature(),
			); err != nil {
				return err
			}
		}
		for _, occurrence := range spec.Operands {
			if err := closure.requireOccurrence(
				evidence, occurrence,
			); err != nil {
				return err
			}
		}
		for _, definition := range spec.Definitions {
			if err := closure.requireDefinition(
				evidence, definition,
			); err != nil {
				return err
			}
		}
		for _, implicit := range spec.Implicit {
			if err := closure.requireOccurrence(
				evidence, implicit.Site(),
			); err != nil {
				return err
			}
			if err := closure.requireType(
				evidence, implicit.Source(),
			); err != nil {
				return err
			}
			if err := closure.requireType(
				evidence, implicit.Target(),
			); err != nil {
				return err
			}
		}
		if err := closure.requireOperation(
			evidence, spec.ControlTarget,
		); err != nil {
			return err
		}
		if err := closure.requireBinding(
			evidence, spec.Label,
		); err != nil {
			return err
		}
	}
	return nil
}

func (closure semanticPackageClosure) verifyUnsupported() error {
	for _, record := range closure.unsupported {
		if err := closure.requireDefinition(
			record.ID().String(),
			record.ID().Definition(),
		); err != nil {
			return err
		}
	}
	return nil
}

func (closure semanticPackageClosure) verifyResolutions() error {
	for _, record := range closure.resolutions {
		evidence := record.Occurrence().String()
		switch record.Kind() {
		case semantic.ResolutionStructuralOnly:
			if err := closure.requireDeclaration(
				evidence,
				record.Structural().Declaration(),
			); err != nil {
				return err
			}
			if err := closure.requireType(
				evidence, record.Structural().Type(),
			); err != nil {
				return err
			}
		case semantic.ResolutionDefinitionComponent:
			if err := closure.requireDefinition(
				evidence, record.Definition(),
			); err != nil {
				return err
			}
		case semantic.ResolutionDeclaration:
			if err := closure.requireDeclaration(
				evidence, record.Declaration(),
			); err != nil {
				return err
			}
		case semantic.ResolutionBinding:
			if err := closure.requireBinding(
				evidence, record.Binding(),
			); err != nil {
				return err
			}
		case semantic.ResolutionType:
			if err := closure.requireType(
				evidence, record.Type(),
			); err != nil {
				return err
			}
		case semantic.ResolutionOperation:
			if err := closure.requireOperation(
				evidence, record.Operation(),
			); err != nil {
				return err
			}
		case semantic.ResolutionUnsupported:
			if err := closure.requireUnsupported(
				evidence, record.Unsupported(),
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func (closure semanticPackageClosure) requireObject(
	evidence string,
	reference semantic.ObjectReference,
) error {
	switch reference.Kind() {
	case semantic.ObjectReferenceDeclaration:
		return closure.requireDeclaration(
			evidence, reference.Declaration(),
		)
	case semantic.ObjectReferenceBinding:
		return closure.requireBinding(
			evidence, reference.Binding(),
		)
	default:
		return nil
	}
}

func (closure semanticPackageClosure) requireDefinition(
	evidence string,
	id identity.DefinitionID,
) error {
	if id.IsZero() {
		return nil
	}
	if _, present := closure.owners.definitions[id]; !present {
		return absentSemanticTarget(evidence, id.String())
	}
	return nil
}

func (closure semanticPackageClosure) requireDeclaration(
	evidence string,
	id identity.SemanticDeclarationID,
) error {
	if id.IsZero() {
		return nil
	}
	if _, present := closure.owners.declarations[id]; !present {
		return absentSemanticTarget(evidence, id.String())
	}
	return nil
}

func (closure semanticPackageClosure) requireBinding(
	evidence string,
	id identity.SemanticBindingID,
) error {
	if id.IsZero() {
		return nil
	}
	if !semanticBindingPresent(closure.bindings, id) {
		return absentSemanticTarget(evidence, id.String())
	}
	return nil
}

func (closure semanticPackageClosure) requireType(
	evidence string,
	id identity.SemanticTypeID,
) error {
	if id.IsZero() {
		return nil
	}
	if !semanticTypePresent(closure.types, id) {
		return absentSemanticTarget(evidence, id.String())
	}
	return nil
}

func (closure semanticPackageClosure) requireOperation(
	evidence string,
	id identity.OperationID,
) error {
	if id.IsZero() {
		return nil
	}
	if !semanticOperationPresent(closure.operations, id) {
		return absentSemanticTarget(evidence, id.String())
	}
	return nil
}

func (closure semanticPackageClosure) requireUnsupported(
	evidence string,
	id identity.UnsupportedID,
) error {
	if id.IsZero() {
		return nil
	}
	if !semanticUnsupportedPresent(closure.unsupported, id) {
		return absentSemanticTarget(evidence, id.String())
	}
	return nil
}

func (closure semanticPackageClosure) requireOccurrence(
	evidence string,
	id identity.OccurrenceID,
) error {
	if id.IsZero() {
		return nil
	}
	if !semanticResolutionPresent(closure.resolutions, id) {
		return absentSemanticTarget(evidence, id.String())
	}
	return nil
}

func absentSemanticTarget(evidence, target string) error {
	return fmt.Errorf(
		"%s references absent semantic target %s",
		evidence, target,
	)
}
