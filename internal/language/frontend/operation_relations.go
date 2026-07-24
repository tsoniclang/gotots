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
	for _, childReference := range children {
		child := builder.input.occurrenceRecord(childReference)
		if child == nil || !runtimeOperand(child, builder) {
			continue
		}
		out = append(out, child.occurrence.ID())
	}
	return out
}

func orderedOperationChildren(
	input *packageInput,
	record *occurrenceInput,
) []packageOccurrenceRef {
	if record.occurrence.Kind() != catalog.KindForStmt &&
		record.occurrence.Kind() != catalog.KindRangeStmt {
		return append([]packageOccurrenceRef(nil), record.children...)
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
	out := append([]packageOccurrenceRef(nil), record.children...)
	sort.SliceStable(out, func(left, right int) bool {
		leftRole := input.occurrenceRecord(out[left]).occurrence.Role()
		rightRole := input.occurrenceRecord(out[right]).occurrence.Role()
		return ranks[leftRole] < ranks[rightRole]
	})
	return out
}

func runtimeOperand(
	record *occurrenceInput,
	builder *packageBuilder,
) bool {
	if builder.contexts.context(record.occurrence.ID()).compileTime {
		return false
	}
	if !catalog.RoleMayContributeRuntimeEvaluation(
		record.occurrence.Role(),
	) {
		return false
	}
	switch record.occurrence.Role() {
	case catalog.RoleElementKey:
		parent := builder.input.occurrence(record.occurrence.Parent())
		return parent == nil ||
			builder.variantByOccurrence[builder.input.occurrenceReference(
				parent.occurrence.ID(),
			)] !=
				catalog.VariantKeyFieldName
	default:
		return true
	}
}

func (builder *packageBuilder) operationDefinitions(
	record *occurrenceInput,
) []identity.DefinitionID {
	definition := builder.input.definition(
		builder.input.occurrenceOwner(record),
	)
	if definition == nil || !definition.hasRegion {
		return nil
	}
	region := definition.region
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
	var targetReference packageOccurrenceRef
	switch branch.Tok {
	case token.BREAK:
		targetReference = item.context.breakTarget
	case token.CONTINUE:
		targetReference = item.context.continueTarget
	case token.FALLTHROUGH:
		targetReference = item.context.fallthroughTarget
	}
	if !targetReference.valid() {
		return identity.OperationID{}, label, nil
	}
	targetRecord := builder.input.occurrenceRecord(targetReference)
	target := builder.operationByOccurrence[targetReference]
	if target.IsZero() {
		return identity.OperationID{}, label, fmt.Errorf(
			"branch %s target occurrence %s has no operation",
			item.record.occurrence.ID(),
			targetRecord.occurrence.ID(),
		)
	}
	return target, label, nil
}

func (builder *packageBuilder) labeledControlTarget(
	label identity.SemanticBindingID,
	lexical token.Token,
) (identity.OperationID, error) {
	declaration := label.Declaration()
	labelRecord := builder.input.occurrence(declaration)
	if labelRecord == nil {
		return identity.OperationID{}, fmt.Errorf(
			"label binding %s has no declaration occurrence", label,
		)
	}
	labeled := builder.input.occurrence(labelRecord.occurrence.Parent())
	if labeled == nil ||
		labeled.occurrence.Kind() != catalog.KindLabeledStmt {
		return identity.OperationID{}, fmt.Errorf(
			"label binding %s has no labeled statement", label,
		)
	}
	targetOccurrence := labeled.occurrence.ID()
	if lexical == token.BREAK || lexical == token.CONTINUE {
		for _, childID := range labeled.children {
			child := builder.input.occurrenceRecord(childID)
			if child != nil &&
				child.occurrence.Role() ==
					catalog.RoleLabeledStatement {
				targetOccurrence = child.occurrence.ID()
				break
			}
		}
	}
	target := builder.operationByOccurrence[builder.input.occurrenceReference(targetOccurrence)]
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
