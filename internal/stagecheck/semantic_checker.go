package stagecheck

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/types"
	"slices"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/semantic"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/language/typesemantics"
	"github.com/tsoniclang/gotots/internal/source"
)

type checkerSemanticVerifier struct {
	expected                semanticPackageExpectation
	actual                  semantic.Package
	index                   *structure.TransientIndex
	view                    *source.TypeInfoView
	types                   *checkerTypeVerifier
	resolutions             map[identity.OccurrenceID]semantic.OccurrenceResolution
	operations              map[identity.OperationID]semantic.Operation
	declarations            map[identity.SemanticDeclarationID]semantic.Declaration
	bindings                map[identity.SemanticBindingID]semantic.Binding
	bindingByObject         map[types.Object]identity.SemanticBindingID
	children                map[identity.OccurrenceID][]identity.OccurrenceID
	containment             map[identity.DefinitionID]checkerDefinitionInterval
	scopeOwners             map[*types.Scope]identity.OccurrenceID
	scopeByOccurrence       map[identity.OccurrenceID]identity.OccurrenceID
	scopeOccurrenceResolved map[identity.OccurrenceID]bool
	checkerScopeOwner       map[*types.Scope]identity.OccurrenceID
	checkerScopeResolved    map[*types.Scope]bool
	definitionByObject      map[types.Object]identity.DefinitionID
	sourceByObject          map[types.Object]identity.OccurrenceID
}

func verifyCheckerSemanticPackage(
	expected semanticPackageExpectation,
	actual semantic.Package,
	universe *source.Universe,
	index *structure.TransientIndex,
	localOnly bool,
) error {
	if expected.loaded.Disposition() ==
		source.DispositionBuiltinUniverse {
		return nil
	}
	view := expected.loaded.CheckerView()
	if view == nil {
		if localOnly &&
			len(expected.definitions) == 0 &&
			len(expected.domains) == 0 {
			return nil
		}
		return semanticVerificationError(
			"checker", "local semantic package has no checker view",
		)
	}
	containment, err := deriveCheckerDefinitionIntervals(expected)
	if err != nil {
		return semanticVerificationError("checker", err.Error())
	}
	verifier := &checkerSemanticVerifier{
		expected: expected, actual: actual,
		index: index, view: view,
		types: newCheckerTypeVerifier(
			expected, actual, universe, index,
		),
		resolutions:             map[identity.OccurrenceID]semantic.OccurrenceResolution{},
		operations:              map[identity.OperationID]semantic.Operation{},
		declarations:            map[identity.SemanticDeclarationID]semantic.Declaration{},
		bindings:                map[identity.SemanticBindingID]semantic.Binding{},
		bindingByObject:         map[types.Object]identity.SemanticBindingID{},
		children:                map[identity.OccurrenceID][]identity.OccurrenceID{},
		containment:             containment,
		scopeByOccurrence:       map[identity.OccurrenceID]identity.OccurrenceID{},
		scopeOccurrenceResolved: map[identity.OccurrenceID]bool{},
		checkerScopeOwner:       map[*types.Scope]identity.OccurrenceID{},
		checkerScopeResolved:    map[*types.Scope]bool{},
		definitionByObject:      map[types.Object]identity.DefinitionID{},
		sourceByObject:          map[types.Object]identity.OccurrenceID{},
	}
	for _, record := range actual.Resolutions() {
		if localOnly &&
			!expected.localOccurrence(
				record.Occurrence(), record.Owner(),
			) {
			continue
		}
		verifier.resolutions[record.Occurrence()] = record
	}
	for _, record := range actual.Operations() {
		if localOnly && record.ID().Source() &&
			!expected.localOccurrence(
				record.Occurrence(), record.Definition(),
			) {
			continue
		}
		verifier.operations[record.ID()] = record
	}
	for _, record := range actual.Declarations() {
		verifier.declarations[record.ID()] = record
	}
	for _, record := range actual.Bindings() {
		if localOnly && !expected.localBinding(record) {
			continue
		}
		verifier.bindings[record.ID()] = record
	}
	for _, occurrenceID := range expected.order {
		occurrence := expected.occurrences[occurrenceID]
		if !occurrence.Parent().IsZero() {
			verifier.children[occurrence.Parent()] = append(
				verifier.children[occurrence.Parent()],
				occurrenceID,
			)
		}
	}
	if err := verifier.deriveIndependentDefinitionOwnership(); err != nil {
		return semanticVerificationError("checker", err.Error())
	}
	if err := verifier.deriveIndependentPackageDeclarationSources(); err != nil {
		return semanticVerificationError("checker", err.Error())
	}
	scopeOwners, err := verifier.independentScopeOwners()
	if err != nil {
		return semanticVerificationError("checker", err.Error())
	}
	verifier.scopeOwners = scopeOwners
	localDeclarations, err :=
		verifier.independentLocalDeclarationIDs()
	if err != nil {
		return semanticVerificationError("checker", err.Error())
	}
	verifier.types.localDeclarations = localDeclarations
	if err := verifier.verifyBindings(); err != nil {
		return semanticVerificationError("checker", err.Error())
	}
	if err := verifier.verifyOccurrences(); err != nil {
		return semanticVerificationError("checker", err.Error())
	}
	if err := verifier.verifyDefinitions(); err != nil {
		return semanticVerificationError("checker", err.Error())
	}
	if err := verifier.verifyBindingCaptures(); err != nil {
		return semanticVerificationError("checker", err.Error())
	}
	if err := verifier.verifyImplicitPackageOperations(); err != nil {
		return semanticVerificationError("checker", err.Error())
	}
	return nil
}

func (verifier *checkerSemanticVerifier) verifyOccurrences() error {
	for _, occurrenceID := range verifier.expected.order {
		occurrence := verifier.expected.occurrences[occurrenceID]
		resolution, present := verifier.resolutions[occurrenceID]
		if !present {
			continue
		}
		node, present := verifier.index.OccurrenceNode(occurrenceID)
		if !present {
			return fmt.Errorf(
				"occurrence %s has no transient node", occurrenceID,
			)
		}
		if verifier.index.CheckedUnmapped(occurrenceID) {
			if resolution.Kind() != semantic.ResolutionUnsupported {
				return fmt.Errorf(
					"checked-unmapped occurrence %s is not unsupported",
					occurrenceID,
				)
			}
			continue
		}
		if resolution.Kind() == semantic.ResolutionStructuralOnly &&
			resolution.Structural().Disposition() ==
				semantic.StructuralIntrinsicContract {
			if resolution.Variant() != catalog.VariantNone {
				return fmt.Errorf(
					"intrinsic occurrence %s carries variant %s",
					occurrenceID, resolution.Variant(),
				)
			}
			continue
		}
		variant, err := independentSemanticVariant(
			verifier.expected,
			verifier.index,
			occurrence,
			node,
		)
		if err != nil {
			if resolution.Variant() == catalog.VariantNone &&
				resolution.Kind() ==
					semantic.ResolutionStructuralOnly {
				continue
			}
			return fmt.Errorf(
				"occurrence %s variant: %w", occurrenceID, err,
			)
		}
		if resolution.Variant() != variant {
			return fmt.Errorf(
				"occurrence %s variant=%s, checker=%s",
				occurrenceID, resolution.Variant(), variant,
			)
		}
		if resolution.Kind() == semantic.ResolutionOperation {
			operation := verifier.operations[resolution.Operation()]
			if operation.ID().IsZero() {
				return fmt.Errorf(
					"occurrence %s operation is absent", occurrenceID,
				)
			}
			if err := verifier.verifyOperation(
				occurrence, node, operation,
			); err != nil {
				return err
			}
		} else if err := verifier.verifyResolutionTarget(
			occurrence, resolution, node,
		); err != nil {
			return fmt.Errorf(
				"occurrence %s (%s/%s) resolution %s: %w",
				occurrenceID,
				occurrence.Kind(),
				occurrence.Role(),
				resolution.Kind(),
				err,
			)
		}
	}
	return nil
}

func (verifier *checkerSemanticVerifier) verifyOperation(
	occurrence structure.Occurrence,
	node ast.Node,
	operation semantic.Operation,
) error {
	wantKind := independentOperationKind(
		verifier.view, occurrence, node, operation.Variant(),
	)
	if operation.Kind() != wantKind {
		return fmt.Errorf(
			"operation %s kind=%s, checker=%s",
			operation.ID(), operation.Kind(), wantKind,
		)
	}
	if err := verifier.verifyOperationValue(
		occurrence, node, operation,
	); err != nil {
		return err
	}
	spec := operation.Spec()
	wantOperands, err := verifier.operationOperands(occurrence)
	if err != nil {
		return fmt.Errorf(
			"operation %s operands: %w", operation.ID(), err,
		)
	}
	if !slices.Equal(spec.Operands, wantOperands) {
		return fmt.Errorf(
			"operation %s operands=%v, checker=%v",
			operation.ID(), spec.Operands, wantOperands,
		)
	}
	if err := verifier.verifyImplicitEffects(
		occurrence, node, operation, wantOperands,
	); err != nil {
		return fmt.Errorf(
			"operation %s implicit effects: %w",
			operation.ID(), err,
		)
	}
	wantDefinitions := verifier.operationDefinitions(operation)
	if !slices.Equal(spec.Definitions, wantDefinitions) {
		return fmt.Errorf(
			"operation %s nested definitions=%v, checker=%v",
			operation.ID(), spec.Definitions, wantDefinitions,
		)
	}
	if spec.Selection.IsZero() {
		if err := verifier.verifyOperationObject(
			occurrence, node, spec.Object,
		); err != nil {
			return fmt.Errorf(
				"operation %s object: %w", operation.ID(), err,
			)
		}
	} else if spec.Object.Kind() !=
		semantic.ObjectReferenceDeclaration ||
		spec.Object.Declaration() != spec.Selection.Object() {
		return fmt.Errorf(
			"operation %s selection/object identity differs",
			operation.ID(),
		)
	}
	if err := verifier.verifySelection(
		node, spec.Selection,
	); err != nil {
		return fmt.Errorf(
			"operation %s selection: %w", operation.ID(), err,
		)
	}
	if err := verifier.verifyInstance(
		node, operation.Kind(), spec.Instance,
	); err != nil {
		return fmt.Errorf(
			"operation %s instance: %w", operation.ID(), err,
		)
	}
	return verifier.verifyOperationControl(
		occurrence, node, operation,
	)
}

type checkerOperationValue struct {
	mode        semantic.ValueMode
	arity       semantic.ResultArity
	place       semantic.PlaceKind
	typ         types.Type
	addressable bool
	assignable  bool
	hasOK       bool
	constant    constant.Value
}

func (verifier *checkerSemanticVerifier) verifyOperationValue(
	occurrence structure.Occurrence,
	node ast.Node,
	operation semantic.Operation,
) error {
	want, err := verifier.operationValue(
		occurrence, node, operation.Kind(),
	)
	if err != nil {
		return fmt.Errorf("operation %s value: %w", operation.ID(), err)
	}
	spec := operation.Spec()
	if spec.Mode != want.mode ||
		spec.Arity != want.arity ||
		spec.Place != want.place ||
		spec.Addressable != want.addressable ||
		spec.Assignable != want.assignable ||
		spec.HasOk != want.hasOK {
		return fmt.Errorf(
			"operation %s value differs: semantic=%v/%v/%v addressable=%t assignable=%t has-ok=%t checker=%v/%v/%v addressable=%t assignable=%t has-ok=%t",
			operation.ID(),
			spec.Mode,
			spec.Arity,
			spec.Place,
			spec.Addressable,
			spec.Assignable,
			spec.HasOk,
			want.mode,
			want.arity,
			want.place,
			want.addressable,
			want.assignable,
			want.hasOK,
		)
	}
	if err := verifier.types.verify(
		spec.ResultType, want.typ,
	); err != nil {
		return fmt.Errorf(
			"operation %s result: %w", operation.ID(), err,
		)
	}
	if want.constant == nil {
		if !spec.Constant.IsZero() {
			return fmt.Errorf(
				"operation %s has unexpected constant", operation.ID(),
			)
		}
	} else if spec.Constant.Exact() != want.constant.ExactString() ||
		spec.Constant.Kind() !=
			semanticConstantKind(want.constant.Kind()) {
		return fmt.Errorf(
			"operation %s constant differs", operation.ID(),
		)
	}
	expectedType := independentExpectedType(
		verifier.expected,
		verifier.index,
		occurrence,
	)
	if err := verifier.types.verify(
		spec.ExpectedType, expectedType,
	); err != nil {
		return fmt.Errorf(
			"operation %s expected type: %w",
			operation.ID(), err,
		)
	}
	return nil
}

func (verifier *checkerSemanticVerifier) operationValue(
	occurrence structure.Occurrence,
	node ast.Node,
	kind semantic.OperationKind,
) (checkerOperationValue, error) {
	out := checkerOperationValue{
		mode:  semantic.ValueModeNone,
		arity: semantic.ResultArityZero,
		place: semantic.PlaceNone,
	}
	expression, expressionNode := node.(ast.Expr)
	if !expressionNode || kind == semantic.OperationKeyedElement {
		return out, nil
	}
	if identifier, ok := expression.(*ast.Ident); ok &&
		identifier.Name == "_" &&
		independentOperationNeedsPlace(occurrence, kind) {
		out.mode = semantic.ValueModePlace
		out.arity = semantic.ResultArityOne
		out.place = semantic.PlaceBlank
		out.assignable = true
		return out, nil
	}
	value, present := verifier.view.TypeOf(expression)
	if !present {
		return verifier.operationValueWithoutType(
			occurrence, expression, kind,
		)
	}
	out.constant = value.Value
	switch {
	case value.IsType():
		return out, fmt.Errorf("type-valued expression became operation")
	case value.IsBuiltin():
		out.mode = semantic.ValueModeBuiltin
		out.arity = semantic.ResultArityOne
		return out, nil
	case value.IsNil():
		out.mode = semantic.ValueModeNil
		out.arity = semantic.ResultArityOne
		return out, nil
	case value.IsVoid():
		out.mode = semantic.ValueModeVoid
		return out, nil
	}
	out.mode = semantic.ValueModeValue
	out.arity = semantic.ResultArityOne
	out.typ = value.Type
	out.addressable = value.Addressable()
	out.assignable = value.Assignable()
	if independentOperationNeedsPlace(occurrence, kind) {
		out.mode = semantic.ValueModePlace
		out.place = independentPlace(
			verifier.view, expression,
		)
		out.assignable = true
		if out.place == semantic.PlaceBlank {
			out.typ = nil
		}
	}
	out.hasOK = independentCommaOK(
		verifier.expected, verifier.index, occurrence,
	) || value.HasOk()
	if out.hasOK {
		out.arity = semantic.ResultArityCommaOk
	} else if _, tuple := types.Unalias(value.Type).(*types.Tuple); tuple {
		out.mode = semantic.ValueModeTuple
		out.arity = semantic.ResultArityTuple
	}
	return out, nil
}

func (verifier *checkerSemanticVerifier) operationValueWithoutType(
	occurrence structure.Occurrence,
	expression ast.Expr,
	kind semantic.OperationKind,
) (checkerOperationValue, error) {
	out := checkerOperationValue{
		arity: semantic.ResultArityOne,
		place: semantic.PlaceNone,
	}
	identifier, ok := expression.(*ast.Ident)
	if !ok {
		return out, fmt.Errorf("expression has no checker type")
	}
	object := independentIdentifierObject(verifier.view, identifier)
	switch object.(type) {
	case *types.PkgName:
		out.mode = semantic.ValueModePackage
	case *types.Label:
		out.mode = semantic.ValueModeLabel
	case *types.Builtin:
		out.mode = semantic.ValueModeBuiltin
	default:
		if object == nil {
			return out, fmt.Errorf("identifier has no checker object")
		}
		out.mode = semantic.ValueModeValue
		out.typ = object.Type()
		if independentOperationNeedsPlace(occurrence, kind) {
			out.mode = semantic.ValueModePlace
			out.place = semantic.PlaceBinding
			out.addressable = true
			out.assignable = true
		}
	}
	return out, nil
}

func independentOperationNeedsPlace(
	occurrence structure.Occurrence,
	kind semantic.OperationKind,
) bool {
	switch occurrence.Role() {
	case catalog.RoleAssignmentTarget,
		catalog.RoleAssignablePlace,
		catalog.RoleRangeKey,
		catalog.RoleRangeValue:
		return true
	default:
		return kind == semantic.OperationDeclare ||
			kind == semantic.OperationStore
	}
}

func independentPlace(
	view *source.TypeInfoView,
	expression ast.Expr,
) semantic.PlaceKind {
	switch node := expression.(type) {
	case *ast.Ident:
		if node.Name == "_" {
			return semantic.PlaceBlank
		}
		return semantic.PlaceBinding
	case *ast.SelectorExpr:
		return semantic.PlaceField
	case *ast.StarExpr:
		return semantic.PlacePointerDereference
	case *ast.IndexExpr:
		value, present := view.TypeOf(node.X)
		if !present {
			return semantic.PlaceInvalid
		}
		core, _ := typesemantics.Core(value.Type)
		switch core.(type) {
		case *types.Map:
			return semantic.PlaceMapElement
		case *types.Array:
			return semantic.PlaceArrayElement
		case *types.Slice:
			return semantic.PlaceSliceElement
		}
	}
	return semantic.PlaceInvalid
}

func independentIdentifierObject(
	view *source.TypeInfoView,
	identifier *ast.Ident,
) types.Object {
	if object, present := view.DefOf(identifier); present {
		return object
	}
	object, _ := view.UseOf(identifier)
	return object
}
