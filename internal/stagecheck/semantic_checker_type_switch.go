package stagecheck

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/structure"
)

func (verifier *checkerSemanticVerifier) verifyTypeSwitchBindingAnchor(
	occurrence structure.Occurrence,
	node ast.Node,
) error {
	identifier, ok := node.(*ast.Ident)
	if !ok ||
		occurrence.Role() != catalog.RoleAssignmentTarget {
		return fmt.Errorf(
			"type-switch binding anchor %s is not an assignment identifier",
			occurrence.ID(),
		)
	}
	if object, _ := verifier.view.DefOf(identifier); object != nil {
		return fmt.Errorf(
			"type-switch binding anchor %s has definition object %s",
			occurrence.ID(), object.Name(),
		)
	}
	if object, _ := verifier.view.UseOf(identifier); object != nil {
		return fmt.Errorf(
			"type-switch binding anchor %s has use object %s",
			occurrence.ID(), object.Name(),
		)
	}
	assignmentOccurrence, present :=
		verifier.expected.occurrences.get(occurrence.Parent())
	if !present ||
		assignmentOccurrence.Kind() != catalog.KindAssignStmt ||
		assignmentOccurrence.Role() != catalog.RoleTypeSwitchGuard {
		return fmt.Errorf(
			"type-switch binding anchor %s has no guard assignment",
			occurrence.ID(),
		)
	}
	assignmentNode, present := verifier.index.OccurrenceNode(
		assignmentOccurrence.ID(),
	)
	assignment, ok := assignmentNode.(*ast.AssignStmt)
	if !present || !ok ||
		assignment.Tok != token.DEFINE ||
		len(assignment.Lhs) != 1 ||
		assignment.Lhs[0] != identifier ||
		len(assignment.Rhs) != 1 {
		return fmt.Errorf(
			"type-switch binding anchor %s has invalid guard shape",
			occurrence.ID(),
		)
	}
	assertion, ok := assignment.Rhs[0].(*ast.TypeAssertExpr)
	if !ok || assertion.Type != nil {
		return fmt.Errorf(
			"type-switch binding anchor %s lacks .(type)",
			occurrence.ID(),
		)
	}
	switchOccurrence, present :=
		verifier.expected.occurrences.get(assignmentOccurrence.Parent())
	if !present ||
		switchOccurrence.Kind() != catalog.KindTypeSwitchStmt {
		return fmt.Errorf(
			"type-switch binding anchor %s has no type-switch owner",
			occurrence.ID(),
		)
	}
	switchNode, present := verifier.index.OccurrenceNode(
		switchOccurrence.ID(),
	)
	typeSwitch, ok := switchNode.(*ast.TypeSwitchStmt)
	if !present || !ok || typeSwitch.Assign != assignment {
		return fmt.Errorf(
			"type-switch binding anchor %s disagrees with its owner",
			occurrence.ID(),
		)
	}
	return verifier.verifyTypeSwitchCaseBindings(
		identifier, typeSwitch,
	)
}

func (verifier *checkerSemanticVerifier) verifyTypeSwitchCaseBindings(
	identifier *ast.Ident,
	typeSwitch *ast.TypeSwitchStmt,
) error {
	for _, statement := range typeSwitch.Body.List {
		clause, ok := statement.(*ast.CaseClause)
		if !ok {
			return fmt.Errorf(
				"type-switch body contains %T instead of a case clause",
				statement,
			)
		}
		object, present := verifier.view.ImplicitOf(clause)
		variable, variableObject := object.(*types.Var)
		if !present ||
			!variableObject ||
			variable.Name() != identifier.Name {
			return fmt.Errorf(
				"type-switch case lacks implicit binding %q",
				identifier.Name,
			)
		}
		clauseID, present := verifier.index.OccurrenceID(clause)
		if !present {
			return fmt.Errorf(
				"type-switch case binding %q has no occurrence",
				identifier.Name,
			)
		}
		bindingID := verifier.bindingByObject[variable]
		binding := verifier.bindings[bindingID]
		if bindingID.IsZero() ||
			binding == nil ||
			bindingID.Owner() != clauseID ||
			binding.role != identity.SemanticBindingTypeSwitch ||
			!binding.source.IsZero() {
			return fmt.Errorf(
				"type-switch case %s has noncanonical binding %s",
				clauseID, bindingID,
			)
		}
	}
	return nil
}
