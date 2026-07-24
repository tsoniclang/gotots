package stagecheck

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/semantic"
	"github.com/tsoniclang/gotots/internal/language/structure"
)

func (verifier *checkerSemanticVerifier) verifyOperationControl(
	reference semanticOccurrenceRef,
	occurrence structure.OccurrenceRef,
	node ast.Node,
	operation semantic.Operation,
) error {
	branch, branchNode := node.(*ast.BranchStmt)
	if !branchNode {
		if !operation.ControlTarget().IsZero() ||
			!operation.Label().IsZero() {
			return fmt.Errorf(
				"non-branch operation %s carries control evidence",
				operation.ID(),
			)
		}
		return nil
	}
	label, err := verifier.independentBranchLabel(
		reference, occurrence, branch,
	)
	if err != nil {
		return fmt.Errorf(
			"operation %s label: %w", operation.ID(), err,
		)
	}
	targetOccurrence, err := verifier.independentBranchTarget(
		reference, occurrence, branch, label,
	)
	if err != nil {
		return fmt.Errorf(
			"operation %s control target: %w",
			operation.ID(), err,
		)
	}
	var target identity.OperationID
	if !targetOccurrence.IsZero() {
		resolution, present := verifier.resolution(targetOccurrence)
		if !present ||
			resolution.Kind() != semantic.ResolutionOperation {
			return fmt.Errorf(
				"target occurrence %s has no operation resolution",
				targetOccurrence,
			)
		}
		target = resolution.Operation()
	}
	if operation.Label() != label ||
		operation.ControlTarget() != target {
		return fmt.Errorf(
			"operation %s control differs: semantic=%s/%s checker=%s/%s",
			operation.ID(),
			operation.Label(),
			operation.ControlTarget(),
			label,
			target,
		)
	}
	return nil
}

func (verifier *checkerSemanticVerifier) independentBranchLabel(
	reference semanticOccurrenceRef,
	occurrence structure.OccurrenceRef,
	branch *ast.BranchStmt,
) (identity.SemanticBindingID, error) {
	if branch.Label == nil {
		return identity.SemanticBindingID{}, nil
	}
	var labelOccurrence identity.OccurrenceID
	for _, childReference := range verifier.childReferences(
		reference,
	) {
		child := verifier.expected.occurrenceRecord(childReference)
		if child.Role() == catalog.RoleLabelReference {
			if !labelOccurrence.IsZero() {
				return identity.SemanticBindingID{},
					fmt.Errorf("branch has two label occurrences")
			}
			labelOccurrence = child.ID()
		}
	}
	if labelOccurrence.IsZero() {
		return identity.SemanticBindingID{},
			fmt.Errorf("branch label has no occurrence")
	}
	resolution, present := verifier.resolution(labelOccurrence)
	if !present || resolution.Kind() != semantic.ResolutionBinding {
		return identity.SemanticBindingID{},
			fmt.Errorf("branch label does not resolve to a binding")
	}
	object, present := verifier.view.UseOf(branch.Label)
	if !present {
		return identity.SemanticBindingID{},
			fmt.Errorf("branch label has no checker object")
	}
	if err := verifier.verifyObjectReference(
		mustBindingReference(resolution.Binding()),
		object,
	); err != nil {
		return identity.SemanticBindingID{}, err
	}
	return resolution.Binding(), nil
}

func (verifier *checkerSemanticVerifier) independentBranchTarget(
	reference semanticOccurrenceRef,
	occurrence structure.OccurrenceRef,
	branch *ast.BranchStmt,
	label identity.SemanticBindingID,
) (identity.OccurrenceID, error) {
	if !label.IsZero() {
		record := verifier.bindings[label]
		if record == nil || record.source.IsZero() {
			return identity.OccurrenceID{},
				fmt.Errorf("label %s has no declaration anchor", label)
		}
		labelReference := verifier.expected.occurrences.reference(
			record.source,
		)
		labeledReference := verifier.expected.parentReference(
			labelReference,
		)
		labeled := verifier.expected.occurrenceRecord(
			labeledReference,
		)
		if labeled.Kind() != catalog.KindLabeledStmt {
			return identity.OccurrenceID{},
				fmt.Errorf("label %s is not owned by a labeled statement", label)
		}
		if branch.Tok != token.BREAK &&
			branch.Tok != token.CONTINUE {
			return labeled.ID(), nil
		}
		for _, childReference := range verifier.childReferences(
			labeledReference,
		) {
			child := verifier.expected.occurrenceRecord(childReference)
			if child.Role() == catalog.RoleLabeledStatement {
				return child.ID(), nil
			}
		}
		return identity.OccurrenceID{},
			fmt.Errorf("labeled control target has no statement")
	}
	switch branch.Tok {
	case token.BREAK:
		return verifier.nearestControlAncestor(
			reference,
			catalog.KindForStmt,
			catalog.KindRangeStmt,
			catalog.KindSwitchStmt,
			catalog.KindTypeSwitchStmt,
			catalog.KindSelectStmt,
		), nil
	case token.CONTINUE:
		return verifier.nearestControlAncestor(
			reference,
			catalog.KindForStmt,
			catalog.KindRangeStmt,
		), nil
	case token.FALLTHROUGH:
		return verifier.nextCaseOccurrence(reference), nil
	default:
		return identity.OccurrenceID{}, nil
	}
}

func (verifier *checkerSemanticVerifier) nearestControlAncestor(
	reference semanticOccurrenceRef,
	kinds ...catalog.Kind,
) identity.OccurrenceID {
	for parentReference := verifier.expected.parentReference(reference); parentReference.valid(); parentReference = verifier.expected.parentReference(parentReference) {
		record := verifier.expected.occurrenceRecord(parentReference)
		for _, kind := range kinds {
			if record.Kind() == kind {
				return record.ID()
			}
		}
	}
	return identity.OccurrenceID{}
}

func (verifier *checkerSemanticVerifier) nextCaseOccurrence(
	reference semanticOccurrenceRef,
) identity.OccurrenceID {
	caseReference := verifier.nearestControlAncestorReference(
		reference, catalog.KindCaseClause,
	)
	if !caseReference.valid() {
		return identity.OccurrenceID{}
	}
	current := verifier.expected.occurrenceRecord(caseReference)
	parentReference := verifier.expected.parentReference(caseReference)
	var next identity.OccurrenceID
	for _, siblingReference := range verifier.childReferences(
		parentReference,
	) {
		sibling := verifier.expected.occurrenceRecord(
			siblingReference,
		)
		if sibling.Kind() != catalog.KindCaseClause ||
			sibling.Ordinal() <= current.Ordinal() {
			continue
		}
		if next.IsZero() ||
			sibling.Ordinal() <
				verifier.expected.occurrence(next).Ordinal() {
			next = sibling.ID()
		}
	}
	return next
}

func (
	verifier *checkerSemanticVerifier,
) nearestControlAncestorReference(
	reference semanticOccurrenceRef,
	kind catalog.Kind,
) semanticOccurrenceRef {
	for parentReference := verifier.expected.parentReference(reference); parentReference.valid(); parentReference = verifier.expected.parentReference(parentReference) {
		if verifier.expected.occurrenceRecord(parentReference).Kind() == kind {
			return parentReference
		}
	}
	return 0
}
