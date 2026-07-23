package frontend

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
)

func (builder *packageBuilder) operationOperands(
	record *occurrenceInput,
) []identity.OccurrenceID {
	children := orderedOperationChildren(builder.input, record)
	out := make([]identity.OccurrenceID, 0, len(children))
	for _, childID := range children {
		child := builder.input.occurrences[childID]
		if child == nil || !runtimeOperand(child, builder) {
			continue
		}
		out = append(out, childID)
	}
	return out
}

func orderedOperationChildren(
	input *packageInput,
	record *occurrenceInput,
) []identity.OccurrenceID {
	if record.occurrence.Kind() != catalog.KindForStmt &&
		record.occurrence.Kind() != catalog.KindRangeStmt {
		return append([]identity.OccurrenceID(nil), record.children...)
	}
	ranks := map[catalog.Role]int{}
	if record.occurrence.Kind() == catalog.KindForStmt {
		ranks = map[catalog.Role]int{
			catalog.RoleInitStatement: 0,
			catalog.RoleCondition:     1,
			catalog.RoleBody:          2,
			catalog.RolePostStatement: 3,
		}
	} else {
		ranks = map[catalog.Role]int{
			catalog.RoleRangeOperand: 0,
			catalog.RoleRangeKey:     1,
			catalog.RoleRangeValue:   2,
			catalog.RoleBody:         3,
		}
	}
	out := append([]identity.OccurrenceID(nil), record.children...)
	sort.SliceStable(out, func(left, right int) bool {
		leftRole := input.occurrences[out[left]].occurrence.Role()
		rightRole := input.occurrences[out[right]].occurrence.Role()
		return ranks[leftRole] < ranks[rightRole]
	})
	return out
}

func runtimeOperand(
	record *occurrenceInput,
	builder *packageBuilder,
) bool {
	switch record.occurrence.Role() {
	case catalog.RoleDocumentation,
		catalog.RoleTrailingDocumentation,
		catalog.RoleCommentText,
		catalog.RolePackageName,
		catalog.RoleDeclaration,
		catalog.RoleDeclarationName,
		catalog.RoleTypeExpression,
		catalog.RoleFieldTag,
		catalog.RoleFieldGroup,
		catalog.RoleConstructedType,
		catalog.RoleSelectedName,
		catalog.RoleAssertedType,
		catalog.RoleArrayLength,
		catalog.RoleElementType,
		catalog.RoleStructFields,
		catalog.RoleTypeParameters,
		catalog.RoleParameters,
		catalog.RoleResults,
		catalog.RoleInterfaceMethods,
		catalog.RoleKeyType,
		catalog.RoleValueType,
		catalog.RoleLabelDeclaration,
		catalog.RoleLabelReference,
		catalog.RoleImportAlias,
		catalog.RoleImportPath,
		catalog.RoleReceiver,
		catalog.RoleFunctionSignature,
		catalog.RoleFunctionBody,
		catalog.RoleSpecification:
		return false
	case catalog.RoleElementKey:
		parent := builder.input.occurrences[record.occurrence.Parent()]
		return parent == nil ||
			builder.variantByOccurrence[parent.occurrence.ID()] !=
				catalog.VariantKeyFieldName
	default:
		return true
	}
}

func (builder *packageBuilder) operationDefinitions(
	record *occurrenceInput,
) []identity.DefinitionID {
	region, present := builder.input.regions[record.owner]
	if !present {
		return nil
	}
	type ordered struct {
		ordinal int
		id      identity.DefinitionID
	}
	var found []ordered
	for _, reference := range region.References() {
		if reference.Parent() == record.occurrence.ID() {
			found = append(found, ordered{
				ordinal: reference.Ordinal(),
				id:      reference.Child(),
			})
		}
	}
	sort.Slice(found, func(left, right int) bool {
		return found[left].ordinal < found[right].ordinal
	})
	out := make([]identity.DefinitionID, 0, len(found))
	for _, item := range found {
		out = append(out, item.id)
	}
	return out
}

func (builder *packageBuilder) operationControl(
	item pendingOperation,
) (
	identity.OperationID,
	identity.SemanticBindingID,
	error,
) {
	branch, branchNode := item.record.node.(*ast.BranchStmt)
	if !branchNode {
		return identity.OperationID{},
			identity.SemanticBindingID{}, nil
	}
	var label identity.SemanticBindingID
	if branch.Label != nil {
		object, present := builder.input.loaded.CheckerView().
			UseOf(branch.Label)
		if !present {
			return identity.OperationID{},
				identity.SemanticBindingID{}, fmt.Errorf(
					"branch %s has no checker label",
					item.record.occurrence.ID(),
				)
		}
		var found bool
		label, found = builder.objects.bindingID(object)
		if !found {
			return identity.OperationID{},
				identity.SemanticBindingID{}, fmt.Errorf(
					"branch %s label has no semantic binding",
					item.record.occurrence.ID(),
				)
		}
		target, err := builder.labeledControlTarget(
			label, branch.Tok,
		)
		return target, label, err
	}
	var targetOccurrence identity.OccurrenceID
	switch branch.Tok {
	case token.BREAK:
		targetOccurrence = item.context.breakTarget
	case token.CONTINUE:
		targetOccurrence = item.context.continueTarget
	case token.FALLTHROUGH:
		targetOccurrence = item.context.fallthroughTarget
	}
	if targetOccurrence.IsZero() {
		return identity.OperationID{}, label, nil
	}
	target := builder.operationByOccurrence[targetOccurrence]
	if target.IsZero() {
		return identity.OperationID{}, label, fmt.Errorf(
			"branch %s target occurrence %s has no operation",
			item.record.occurrence.ID(), targetOccurrence,
		)
	}
	return target, label, nil
}

func (builder *packageBuilder) labeledControlTarget(
	label identity.SemanticBindingID,
	lexical token.Token,
) (identity.OperationID, error) {
	declaration := label.Declaration()
	labelRecord := builder.input.occurrences[declaration]
	if labelRecord == nil {
		return identity.OperationID{}, fmt.Errorf(
			"label binding %s has no declaration occurrence", label,
		)
	}
	labeled := builder.input.occurrences[labelRecord.occurrence.Parent()]
	if labeled == nil ||
		labeled.occurrence.Kind() != catalog.KindLabeledStmt {
		return identity.OperationID{}, fmt.Errorf(
			"label binding %s has no labeled statement", label,
		)
	}
	targetOccurrence := labeled.occurrence.ID()
	if lexical == token.BREAK || lexical == token.CONTINUE {
		for _, childID := range labeled.children {
			child := builder.input.occurrences[childID]
			if child != nil &&
				child.occurrence.Role() ==
					catalog.RoleLabeledStatement {
				targetOccurrence = childID
				break
			}
		}
	}
	target := builder.operationByOccurrence[targetOccurrence]
	if target.IsZero() {
		return identity.OperationID{}, fmt.Errorf(
			"label binding %s target %s has no operation",
			label, targetOccurrence,
		)
	}
	return target, nil
}

func (builder *packageBuilder) identifierObject(
	identifier *ast.Ident,
) types.Object {
	view := builder.input.loaded.CheckerView()
	if object, present := view.DefOf(identifier); present {
		return object
	}
	object, _ := view.UseOf(identifier)
	return object
}
