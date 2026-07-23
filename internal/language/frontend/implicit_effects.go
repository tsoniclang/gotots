package frontend

import (
	"go/ast"
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/semantic"
)

func (builder *packageBuilder) implicitEffects(
	item pendingOperation,
	operands []identity.OccurrenceID,
	selection semantic.Selection,
) ([]semantic.ImplicitOperation, error) {
	var out []semantic.ImplicitOperation
	ordinals := map[catalog.ImplicitOp]int{}
	appendEffect := func(
		kind catalog.ImplicitOp,
		site identity.OccurrenceID,
		source types.Type,
		target types.Type,
	) error {
		sourceID, err := builder.optionalType(source)
		if err != nil {
			return err
		}
		targetID, err := builder.optionalType(target)
		if err != nil {
			return err
		}
		effect, err := semantic.NewImplicitOperation(
			kind, site, ordinals[kind], sourceID, targetID,
		)
		if err != nil {
			return err
		}
		ordinals[kind]++
		out = append(out, effect)
		return nil
	}
	if item.context.zeroValue {
		if err := appendEffect(
			catalog.ImplicitZeroing,
			item.record.occurrence.ID(),
			nil,
			expressionTypeOf(builder, item.record.node),
		); err != nil {
			return nil, err
		}
	}
	if item.kind == semantic.OperationComposite {
		if err := appendEffect(
			catalog.ImplicitZeroing,
			item.record.occurrence.ID(),
			nil,
			expressionTypeOf(builder, item.record.node),
		); err != nil {
			return nil, err
		}
	}
	if item.kind == semantic.OperationBlock &&
		item.record.occurrence.Role() ==
			catalog.RoleFunctionBody {
		var candidates []*bindingCandidate
		for _, candidate := range builder.objects.bindingByObject {
			if candidate.definition != item.record.owner ||
				candidate.role != identity.SemanticBindingResult ||
				candidate.source.IsZero() {
				continue
			}
			candidates = append(candidates, candidate)
		}
		sort.Slice(candidates, func(left, right int) bool {
			return builder.objects.bindingIDs[candidates[left]].String() <
				builder.objects.bindingIDs[candidates[right]].String()
		})
		for _, candidate := range candidates {
			if err := appendEffect(
				catalog.ImplicitZeroing,
				candidate.source,
				nil,
				candidate.typ,
			); err != nil {
				return nil, err
			}
		}
	}
	if operationCopiesOperands(item.kind) {
		for _, operand := range operands {
			record := builder.input.occurrences[operand]
			if record == nil ||
				!implicitCopySource(record.occurrence.Role()) {
				continue
			}
			source := expressionTypeOf(builder, record.node)
			target := builder.contexts.byOccurrence[operand].expected
			if valueCopies(source) {
				if err := appendEffect(
					catalog.ImplicitValueCopy,
					operand,
					source,
					target,
				); err != nil {
					return nil, err
				}
			}
			if source != nil &&
				target != nil &&
				!types.Identical(source, target) &&
				types.AssignableTo(source, target) {
				if err := appendEffect(
					catalog.ImplicitAssignmentConversion,
					operand,
					source,
					target,
				); err != nil {
					return nil, err
				}
			}
			if interfaceConversion(source, target) {
				if err := appendEffect(
					catalog.ImplicitInterfaceConversion,
					operand,
					source,
					target,
				); err != nil {
					return nil, err
				}
				if err := appendEffect(
					catalog.ImplicitBoxing,
					operand,
					source,
					target,
				); err != nil {
					return nil, err
				}
			}
		}
	}
	if !selection.IsZero() {
		selector, _ := item.record.node.(*ast.SelectorExpr)
		checkerSelection, _ := builder.input.loaded.CheckerView().
			SelectionOf(selector)
		var target types.Type
		if checkerSelection != nil {
			if function, ok := checkerSelection.Obj().(*types.Func); ok {
				if signature, ok := function.Type().(*types.Signature); ok &&
					signature.Recv() != nil {
					target = signature.Recv().Type()
				}
			}
		}
		site := item.record.occurrence.ID()
		if selection.Indirect() {
			if err := appendEffect(
				catalog.ImplicitReceiverAdjustment,
				site,
				checkerSelection.Recv(),
				target,
			); err != nil {
				return nil, err
			}
		}
		if len(selection.Index()) > 1 {
			if err := appendEffect(
				catalog.ImplicitMethodPromotion,
				site,
				checkerSelection.Recv(),
				target,
			); err != nil {
				return nil, err
			}
		}
	}
	if len(operands) > 1 {
		if err := appendEffect(
			catalog.ImplicitEvaluationOrder,
			item.record.occurrence.ID(),
			nil,
			nil,
		); err != nil {
			return nil, err
		}
	}
	if builder.operationCanPanic(item) {
		if err := appendEffect(
			catalog.ImplicitPanicBoundary,
			item.record.occurrence.ID(),
			nil,
			nil,
		); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func implicitCopySource(role catalog.Role) bool {
	switch role {
	case catalog.RoleAssignmentTarget,
		catalog.RoleAssignablePlace,
		catalog.RoleRangeKey,
		catalog.RoleRangeValue:
		return false
	default:
		return true
	}
}

func expressionTypeOf(
	builder *packageBuilder,
	node ast.Node,
) types.Type {
	expression, ok := node.(ast.Expr)
	if !ok {
		return nil
	}
	return expressionType(
		builder.input.loaded.CheckerView(), expression,
	)
}

func valueCopies(typ types.Type) bool {
	if typ == nil {
		return false
	}
	switch types.Unalias(typ).Underlying().(type) {
	case *types.Struct, *types.Array:
		return true
	default:
		return false
	}
}

func interfaceConversion(
	source types.Type,
	target types.Type,
) bool {
	if source == nil || target == nil {
		return false
	}
	_, targetInterface := types.Unalias(target).Underlying().(*types.Interface)
	_, sourceInterface := types.Unalias(source).Underlying().(*types.Interface)
	return targetInterface && !sourceInterface
}

func operationCopiesOperands(
	kind semantic.OperationKind,
) bool {
	switch kind {
	case semantic.OperationAssign,
		semantic.OperationCall,
		semantic.OperationBuiltinCall,
		semantic.OperationConvert,
		semantic.OperationReturn,
		semantic.OperationSend,
		semantic.OperationComposite,
		semantic.OperationKeyedElement,
		semantic.OperationDeclare:
		return true
	default:
		return false
	}
}

func (builder *packageBuilder) operationCanPanic(
	item pendingOperation,
) bool {
	switch item.kind {
	case semantic.OperationIndex,
		semantic.OperationSlice,
		semantic.OperationDereference,
		semantic.OperationCall,
		semantic.OperationSend:
		return true
	case semantic.OperationTypeAssert:
		return item.variant == catalog.VariantAssertValue
	case semantic.OperationBinary:
		return item.record.occurrence.Token() == catalog.TokenQUO ||
			item.record.occurrence.Token() == catalog.TokenREM
	case semantic.OperationBuiltinCall:
		call, _ := item.record.node.(*ast.CallExpr)
		if call == nil {
			return false
		}
		identifier := calleeIdentifier(call.Fun)
		if identifier == nil {
			return false
		}
		object, present := builder.input.loaded.CheckerView().
			UseOf(identifier)
		if !present {
			return false
		}
		if _, ok := object.(*types.Builtin); !ok {
			return false
		}
		switch builder.objects.predeclared[object] {
		case catalog.PredeclaredClose,
			catalog.PredeclaredMake,
			catalog.PredeclaredNew,
			catalog.PredeclaredPanic:
			return true
		}
	}
	return false
}
