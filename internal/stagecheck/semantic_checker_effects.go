package stagecheck

import (
	"fmt"
	"go/ast"
	"go/types"
	"slices"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/semantic"
	"github.com/tsoniclang/gotots/internal/language/structure"
)

type checkerImplicitEffect struct {
	kind   catalog.ImplicitOp
	site   identity.OccurrenceID
	source types.Type
	target types.Type
}

func (verifier *checkerSemanticVerifier) verifyImplicitEffects(
	occurrence structure.Occurrence,
	node ast.Node,
	operation semantic.Operation,
	operands []identity.OccurrenceID,
) error {
	expected, err := verifier.independentImplicitEffects(
		occurrence, node, operation, operands,
	)
	if err != nil {
		return err
	}
	return verifier.compareImplicitEffects(
		operation.Spec().Implicit, expected,
	)
}

func (verifier *checkerSemanticVerifier) independentImplicitEffects(
	occurrence structure.Occurrence,
	node ast.Node,
	operation semantic.Operation,
	operands []identity.OccurrenceID,
) ([]checkerImplicitEffect, error) {
	var expected []checkerImplicitEffect
	appendEffect := func(
		kind catalog.ImplicitOp,
		site identity.OccurrenceID,
		source types.Type,
		target types.Type,
	) {
		expected = append(expected, checkerImplicitEffect{
			kind: kind, site: site,
			source: source, target: target,
		})
	}
	if verifier.independentZeroValue(occurrence) {
		appendEffect(
			catalog.ImplicitZeroing,
			occurrence.ID(),
			nil,
			independentNodeType(verifier.view, node),
		)
	}
	if operation.Kind() == semantic.OperationComposite {
		appendEffect(
			catalog.ImplicitZeroing,
			occurrence.ID(),
			nil,
			independentNodeType(verifier.view, node),
		)
	}
	if operation.Kind() == semantic.OperationBlock &&
		occurrence.Role() == catalog.RoleFunctionBody {
		for _, binding := range verifier.namedResultBindings(
			operation.Definition(),
		) {
			object, err := verifier.bindingCheckerObject(binding)
			if err != nil {
				return nil, err
			}
			appendEffect(
				catalog.ImplicitZeroing,
				binding.Source(),
				nil,
				object.Type(),
			)
		}
	}
	if independentOperationCopiesOperands(operation.Kind()) {
		for _, operand := range operands {
			if !independentImplicitCopySource(
				verifier.expected.occurrences[operand].Role(),
			) {
				continue
			}
			operandNode, present := verifier.index.OccurrenceNode(operand)
			if !present {
				return nil, fmt.Errorf(
					"operand %s has no checker node", operand,
				)
			}
			source := independentNodeType(
				verifier.view, operandNode,
			)
			target := independentExpectedType(
				verifier.expected,
				verifier.index,
				verifier.expected.occurrences[operand],
			)
			if independentValueCopies(source) {
				appendEffect(
					catalog.ImplicitValueCopy,
					operand, source, target,
				)
			}
			if source != nil &&
				target != nil &&
				!types.Identical(source, target) &&
				types.AssignableTo(source, target) {
				appendEffect(
					catalog.ImplicitAssignmentConversion,
					operand, source, target,
				)
			}
			if independentInterfaceConversion(source, target) {
				appendEffect(
					catalog.ImplicitInterfaceConversion,
					operand, source, target,
				)
				appendEffect(
					catalog.ImplicitBoxing,
					operand, source, target,
				)
			}
		}
	}
	selection := operation.Spec().Selection
	if !selection.IsZero() {
		selector, _ := node.(*ast.SelectorExpr)
		checker, present := verifier.view.SelectionOf(selector)
		if !present {
			return nil, fmt.Errorf(
				"selection effect has no checker selection",
			)
		}
		var target types.Type
		if function, ok := checker.Obj().(*types.Func); ok {
			signature, _ := function.Type().(*types.Signature)
			if signature != nil && signature.Recv() != nil {
				target = signature.Recv().Type()
			}
		}
		if checker.Indirect() {
			appendEffect(
				catalog.ImplicitReceiverAdjustment,
				occurrence.ID(), checker.Recv(), target,
			)
		}
		if len(checker.Index()) > 1 {
			appendEffect(
				catalog.ImplicitMethodPromotion,
				occurrence.ID(), checker.Recv(), target,
			)
		}
	}
	if len(operands) > 1 {
		appendEffect(
			catalog.ImplicitEvaluationOrder,
			occurrence.ID(), nil, nil,
		)
	}
	if verifier.independentOperationCanPanic(
		operation.Kind(),
		operation.Variant(),
		operation.Token(),
		node,
	) {
		appendEffect(
			catalog.ImplicitPanicBoundary,
			occurrence.ID(), nil, nil,
		)
	}
	return expected, nil
}

func (verifier *checkerSemanticVerifier) compareImplicitEffects(
	actual []semantic.ImplicitOperation,
	expected []checkerImplicitEffect,
) error {
	if len(actual) != len(expected) {
		return fmt.Errorf(
			"effect count=%d %v, checker-derived=%d %v",
			len(actual),
			semanticEffectIdentities(actual),
			len(expected),
			checkerEffectIdentities(expected),
		)
	}
	ordinals := map[catalog.ImplicitOp]int{}
	for index, want := range expected {
		got := actual[index]
		ordinal := ordinals[want.kind]
		ordinals[want.kind]++
		if got.Kind() != want.kind ||
			got.Site() != want.site ||
			got.Ordinal() != ordinal {
			return fmt.Errorf(
				"effect %d identity=%s/%s/%d, checker=%s/%s/%d",
				index,
				got.Kind(),
				got.Site(),
				got.Ordinal(),
				want.kind,
				want.site,
				ordinal,
			)
		}
		if err := verifier.types.verify(
			got.Source(), want.source,
		); err != nil {
			return fmt.Errorf("effect %d source: %w", index, err)
		}
		if err := verifier.types.verify(
			got.Target(), want.target,
		); err != nil {
			return fmt.Errorf("effect %d target: %w", index, err)
		}
	}
	return nil
}

func semanticEffectIdentities(
	effects []semantic.ImplicitOperation,
) []string {
	out := make([]string, 0, len(effects))
	for _, effect := range effects {
		out = append(out, fmt.Sprintf(
			"%s@%s/%d",
			effect.Kind(), effect.Site(), effect.Ordinal(),
		))
	}
	return out
}

func checkerEffectIdentities(
	effects []checkerImplicitEffect,
) []string {
	ordinals := map[catalog.ImplicitOp]int{}
	out := make([]string, 0, len(effects))
	for _, effect := range effects {
		out = append(out, fmt.Sprintf(
			"%s@%s/%d",
			effect.kind,
			effect.site,
			ordinals[effect.kind],
		))
		ordinals[effect.kind]++
	}
	return out
}

func (verifier *checkerSemanticVerifier) independentZeroValue(
	occurrence structure.Occurrence,
) bool {
	if occurrence.Role() != catalog.RoleDeclarationName {
		return false
	}
	parentNode, present := verifier.index.OccurrenceNode(
		occurrence.Parent(),
	)
	value, valueSpec := parentNode.(*ast.ValueSpec)
	return present && valueSpec && len(value.Values) == 0
}

func (verifier *checkerSemanticVerifier) namedResultBindings(
	definition identity.DefinitionID,
) []semantic.Binding {
	var out []semantic.Binding
	for _, binding := range verifier.bindings {
		if binding.Definition() == definition &&
			binding.Role() == identity.SemanticBindingResult &&
			!binding.Source().IsZero() {
			out = append(out, binding)
		}
	}
	sort.Slice(out, func(left, right int) bool {
		return out[left].ID().String() < out[right].ID().String()
	})
	return out
}

func (verifier *checkerSemanticVerifier) bindingCheckerObject(
	binding semantic.Binding,
) (types.Object, error) {
	node, present := verifier.index.OccurrenceNode(binding.Source())
	identifier, identifierNode := node.(*ast.Ident)
	if !present || !identifierNode {
		return nil, fmt.Errorf(
			"binding %s has no identifier source", binding.ID(),
		)
	}
	object, present := verifier.view.DefOf(identifier)
	if !present {
		return nil, fmt.Errorf(
			"binding %s source has no checker object", binding.ID(),
		)
	}
	return object, nil
}

func independentImplicitCopySource(role catalog.Role) bool {
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

func independentValueCopies(typ types.Type) bool {
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

func independentInterfaceConversion(
	source types.Type,
	target types.Type,
) bool {
	if source == nil || target == nil {
		return false
	}
	_, targetInterface := types.Unalias(target).
		Underlying().(*types.Interface)
	_, sourceInterface := types.Unalias(source).
		Underlying().(*types.Interface)
	return targetInterface && !sourceInterface
}

func independentOperationCopiesOperands(
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

func (verifier *checkerSemanticVerifier) independentOperationCanPanic(
	kind semantic.OperationKind,
	variant catalog.Variant,
	lexical catalog.TokenKind,
	node ast.Node,
) bool {
	switch kind {
	case semantic.OperationIndex,
		semantic.OperationSlice,
		semantic.OperationDereference,
		semantic.OperationCall,
		semantic.OperationSend:
		return true
	case semantic.OperationTypeAssert:
		return variant == catalog.VariantAssertValue
	case semantic.OperationBinary:
		return lexical == catalog.TokenQUO ||
			lexical == catalog.TokenREM
	case semantic.OperationBuiltinCall:
		call, _ := node.(*ast.CallExpr)
		if call == nil {
			return false
		}
		object := independentExpressionObject(
			verifier.view, call.Fun,
		)
		builtin, ok := object.(*types.Builtin)
		if !ok {
			return false
		}
		switch independentPredeclaredKind(builtin) {
		case catalog.PredeclaredClose,
			catalog.PredeclaredMake,
			catalog.PredeclaredNew,
			catalog.PredeclaredPanic:
			return true
		}
	}
	return false
}

func (verifier *checkerSemanticVerifier) verifyImplicitPackageOperations() error {
	for definition := range verifier.expected.definitions {
		if definition.ImplicitOp() !=
			identity.ImplicitDefinitionPackageInit ||
			!verifier.expected.executable[definition] {
			continue
		}
		id, err := identity.NewImplicitOperationID(
			definition,
			identity.ImplicitDefinitionPackageInit,
			0,
		)
		if err != nil {
			return err
		}
		operation := verifier.operations[id]
		if operation.ID().IsZero() {
			return fmt.Errorf(
				"package initialization %s has no semantic operation",
				definition,
			)
		}
		if err := verifier.verifyPackageInitialization(
			operation,
		); err != nil {
			return err
		}
	}
	return nil
}

func (verifier *checkerSemanticVerifier) verifyPackageInitialization(
	operation semantic.Operation,
) error {
	operands, definitions, effects, err :=
		verifier.independentPackageInitialization()
	if err != nil {
		return err
	}
	spec := operation.Spec()
	if operation.Kind() !=
		semantic.OperationPackageInitialization ||
		!slices.Equal(spec.Operands, operands) ||
		!slices.Equal(spec.Definitions, definitions) {
		return fmt.Errorf(
			"package initialization %s sequence differs",
			operation.ID(),
		)
	}
	return verifier.compareImplicitEffects(spec.Implicit, effects)
}

func (verifier *checkerSemanticVerifier) independentPackageInitialization() (
	[]identity.OccurrenceID,
	[]identity.DefinitionID,
	[]checkerImplicitEffect,
	error,
) {
	var operands []identity.OccurrenceID
	var definitions []identity.DefinitionID
	var effects []checkerImplicitEffect
	seenDefinitions := map[identity.DefinitionID]bool{}
	initialized := map[*types.Var]bool{}
	for _, entry := range verifier.view.InitOrder() {
		entryDefinitions := map[identity.DefinitionID]bool{}
		for _, variable := range entry.Vars {
			if variable.Name() == "_" {
				continue
			}
			occurrence, err := verifier.index.IdentifierOccurrence(
				variable.Pos(), variable.Name(),
			)
			if err != nil {
				return nil, nil, nil, err
			}
			definition := verifier.expected.owners[occurrence]
			if definition.IsZero() {
				return nil, nil, nil, fmt.Errorf(
					"initializer variable %s has no definition",
					variable.Name(),
				)
			}
			entryDefinitions[definition] = true
		}
		occurrence, present := verifier.index.OccurrenceID(entry.Rhs)
		if present {
			definition := verifier.expected.owners[occurrence]
			if definition.IsZero() {
				return nil, nil, nil, fmt.Errorf(
					"initializer expression has no definition",
				)
			}
			entryDefinitions[definition] = true
		}
		full := map[identity.DefinitionID]bool{}
		for definition := range entryDefinitions {
			if verifier.expected.executable[definition] {
				full[definition] = true
			}
		}
		if len(full) == 0 {
			continue
		}
		if len(full) != len(entryDefinitions) || !present {
			return nil, nil, nil, fmt.Errorf(
				"checker initialization order crosses semantic depth",
			)
		}
		owner := verifier.expected.owners[occurrence]
		operands = append(operands, occurrence)
		if !seenDefinitions[owner] {
			seenDefinitions[owner] = true
			definitions = append(definitions, owner)
		}
		effects = append(effects, checkerImplicitEffect{
			kind: catalog.ImplicitInitialization,
			site: occurrence,
		})
		for _, variable := range entry.Vars {
			initialized[variable] = true
		}
	}
	type zeroCandidate struct {
		variable   *types.Var
		occurrence identity.OccurrenceID
	}
	var zeroes []zeroCandidate
	checkerPackage := verifier.expected.loaded.Types()
	if checkerPackage != nil {
		scope := checkerPackage.Scope()
		for _, name := range scope.Names() {
			variable, ok := scope.Lookup(name).(*types.Var)
			if !ok || initialized[variable] {
				continue
			}
			occurrence, err := verifier.index.IdentifierOccurrence(
				variable.Pos(), variable.Name(),
			)
			if err != nil {
				continue
			}
			zeroes = append(zeroes, zeroCandidate{
				variable: variable, occurrence: occurrence,
			})
		}
	}
	sort.Slice(zeroes, func(left, right int) bool {
		return zeroes[left].occurrence.String() <
			zeroes[right].occurrence.String()
	})
	for _, candidate := range zeroes {
		effects = append(effects, checkerImplicitEffect{
			kind:   catalog.ImplicitZeroing,
			site:   candidate.occurrence,
			target: candidate.variable.Type(),
		})
	}
	if len(operands) > 1 {
		effects = append(effects, checkerImplicitEffect{
			kind: catalog.ImplicitEvaluationOrder,
			site: operands[0],
		})
	}
	return operands, definitions, effects, nil
}
