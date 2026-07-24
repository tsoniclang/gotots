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
	occurrence structure.OccurrenceRef,
	node ast.Node,
	operation semantic.Operation,
) error {
	branch, branchNode := node.(*ast.BranchStmt)
	spec := operation.Spec()
	if !branchNode {
		if !spec.ControlTarget.IsZero() || !spec.Label.IsZero() {
			return fmt.Errorf(
				"non-branch operation %s carries control evidence",
				operation.ID(),
			)
		}
		return nil
	}
	label, err := verifier.independentBranchLabel(
		occurrence, branch,
	)
	if err != nil {
		return fmt.Errorf(
			"operation %s label: %w", operation.ID(), err,
		)
	}
	targetOccurrence, err := verifier.independentBranchTarget(
		occurrence, branch, label,
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
	if spec.Label != label || spec.ControlTarget != target {
		return fmt.Errorf(
			"operation %s control differs: semantic=%s/%s checker=%s/%s",
			operation.ID(),
			spec.Label,
			spec.ControlTarget,
			label,
			target,
		)
	}
	return nil
}

func (verifier *checkerSemanticVerifier) independentBranchLabel(
	occurrence structure.OccurrenceRef,
	branch *ast.BranchStmt,
) (identity.SemanticBindingID, error) {
	if branch.Label == nil {
		return identity.SemanticBindingID{}, nil
	}
	var labelOccurrence identity.OccurrenceID
	for _, childID := range verifier.children[occurrence.ID()] {
		child := verifier.expected.occurrence(childID)
		if child.Role() == catalog.RoleLabelReference {
			if !labelOccurrence.IsZero() {
				return identity.SemanticBindingID{},
					fmt.Errorf("branch has two label occurrences")
			}
			labelOccurrence = childID
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
		labelOccurrence := verifier.expected.occurrence(record.source)
		labeled := verifier.expected.occurrence(labelOccurrence.Parent())
		if labeled.Kind() != catalog.KindLabeledStmt {
			return identity.OccurrenceID{},
				fmt.Errorf("label %s is not owned by a labeled statement", label)
		}
		if branch.Tok != token.BREAK &&
			branch.Tok != token.CONTINUE {
			return labeled.ID(), nil
		}
		for _, childID := range verifier.children[labeled.ID()] {
			child := verifier.expected.occurrence(childID)
			if child.Role() == catalog.RoleLabeledStatement {
				return childID, nil
			}
		}
		return identity.OccurrenceID{},
			fmt.Errorf("labeled control target has no statement")
	}
	switch branch.Tok {
	case token.BREAK:
		return verifier.nearestControlAncestor(
			occurrence,
			catalog.KindForStmt,
			catalog.KindRangeStmt,
			catalog.KindSwitchStmt,
			catalog.KindTypeSwitchStmt,
			catalog.KindSelectStmt,
		), nil
	case token.CONTINUE:
		return verifier.nearestControlAncestor(
			occurrence,
			catalog.KindForStmt,
			catalog.KindRangeStmt,
		), nil
	case token.FALLTHROUGH:
		return verifier.nextCaseOccurrence(occurrence), nil
	default:
		return identity.OccurrenceID{}, nil
	}
}

func (verifier *checkerSemanticVerifier) nearestControlAncestor(
	occurrence structure.OccurrenceRef,
	kinds ...catalog.Kind,
) identity.OccurrenceID {
	for parent := occurrence.Parent(); !parent.IsZero(); {
		record := verifier.expected.occurrence(parent)
		for _, kind := range kinds {
			if record.Kind() == kind {
				return parent
			}
		}
		parent = record.Parent()
	}
	return identity.OccurrenceID{}
}

func (verifier *checkerSemanticVerifier) nextCaseOccurrence(
	occurrence structure.OccurrenceRef,
) identity.OccurrenceID {
	caseID := verifier.nearestControlAncestor(
		occurrence, catalog.KindCaseClause,
	)
	if caseID.IsZero() {
		return identity.OccurrenceID{}
	}
	current := verifier.expected.occurrence(caseID)
	var next identity.OccurrenceID
	for _, siblingID := range verifier.children[current.Parent()] {
		sibling := verifier.expected.occurrence(siblingID)
		if sibling.Kind() != catalog.KindCaseClause ||
			sibling.Ordinal() <= current.Ordinal() {
			continue
		}
		if next.IsZero() ||
			sibling.Ordinal() <
				verifier.expected.occurrence(next).Ordinal() {
			next = siblingID
		}
	}
	return next
}
