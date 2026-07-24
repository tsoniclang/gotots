package semantic

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
)

type OperationSpec struct {
	ID      identity.OperationID
	Kind    OperationKind
	Syntax  catalog.Kind
	Variant catalog.Variant
	Role    catalog.Role
	Token   catalog.TokenKind

	Mode         ValueMode
	Arity        ResultArity
	Place        PlaceKind
	ResultType   identity.SemanticTypeID
	ExpectedType identity.SemanticTypeID
	Addressable  bool
	Assignable   bool
	HasOk        bool
	Constant     Constant
	Object       ObjectReference
	Selection    Selection
	Instance     Instance

	Operands      []identity.OccurrenceID
	Definitions   []identity.DefinitionID
	Implicit      []ImplicitOperation
	ControlTarget identity.OperationID
	Label         identity.SemanticBindingID
}

type Operation struct {
	spec OperationSpec
}

func NewOperation(spec OperationSpec) (Operation, error) {
	return validateOperationSpec(cloneOperationSpec(spec))
}

func validateOperationSpec(spec OperationSpec) (Operation, error) {
	if spec.ID.IsZero() ||
		!spec.Kind.Valid() ||
		!spec.Mode.Valid() ||
		!spec.Arity.Valid() ||
		!spec.Place.Valid() ||
		!spec.Object.Valid() {
		return Operation{}, fmt.Errorf(
			"operation %s has invalid required fields", spec.ID,
		)
	}
	if err := validateOperationOrigin(spec); err != nil {
		return Operation{}, err
	}
	if err := validateTypeSwitchGuardOperation(spec); err != nil {
		return Operation{}, err
	}
	if spec.Mode == ValueModePlace {
		if spec.Place == PlaceNone ||
			!spec.Assignable ||
			(spec.Place != PlaceBlank &&
				spec.ResultType.IsZero()) {
			return Operation{}, fmt.Errorf(
				"place operation %s lacks place semantics", spec.ID,
			)
		}
	} else if spec.Place != PlaceNone {
		return Operation{}, fmt.Errorf(
			"non-place operation %s carries place class", spec.ID,
		)
	}
	needsResultType := operationNeedsResultType(spec.Mode)
	if spec.Mode == ValueModePlace &&
		spec.Place == PlaceBlank {
		needsResultType = false
	}
	if needsResultType != !spec.ResultType.IsZero() {
		return Operation{}, fmt.Errorf(
			"operation %s result type disagrees with value mode",
			spec.ID,
		)
	}
	if !spec.Constant.IsZero() &&
		(spec.Mode != ValueModeValue &&
			spec.Mode != ValueModePlace) {
		return Operation{}, fmt.Errorf(
			"non-value operation %s carries constant", spec.ID,
		)
	}
	if !spec.ControlTarget.IsZero() &&
		spec.ControlTarget.Definition() != spec.ID.Definition() {
		return Operation{}, fmt.Errorf(
			"operation %s has cross-definition control target",
			spec.ID,
		)
	}
	if !spec.Selection.IsZero() {
		switch spec.Kind {
		case OperationFieldSelect,
			OperationMethodValue,
			OperationMethodExpression:
		default:
			return Operation{}, fmt.Errorf(
				"operation %s carries an inapplicable selection",
				spec.ID,
			)
		}
	}
	if !spec.Instance.IsZero() &&
		spec.Kind != OperationGenericInstantiate &&
		spec.Kind != OperationCall {
		return Operation{}, fmt.Errorf(
			"operation %s carries an inapplicable generic instance",
			spec.ID,
		)
	}
	seenOperands := map[identity.OccurrenceID]bool{}
	for _, operand := range spec.Operands {
		if operand.IsZero() || seenOperands[operand] {
			return Operation{}, fmt.Errorf(
				"operation %s has invalid ordered operands", spec.ID,
			)
		}
		seenOperands[operand] = true
	}
	seenDefinitions := map[identity.DefinitionID]bool{}
	for _, definition := range spec.Definitions {
		if definition.IsZero() || seenDefinitions[definition] {
			return Operation{}, fmt.Errorf(
				"operation %s has invalid nested definitions", spec.ID,
			)
		}
		seenDefinitions[definition] = true
	}
	type implicitKey struct {
		kind    catalog.ImplicitOp
		site    identity.OccurrenceID
		ordinal int
	}
	seenImplicit := map[implicitKey]bool{}
	for _, implicit := range spec.Implicit {
		key := implicitKey{
			kind: implicit.Kind(), site: implicit.Site(),
			ordinal: implicit.Ordinal(),
		}
		if !implicit.Kind().Valid() ||
			seenImplicit[key] {
			return Operation{}, fmt.Errorf(
				"operation %s has invalid implicit operations",
				spec.ID,
			)
		}
		seenImplicit[key] = true
	}
	return Operation{spec: spec}, nil
}

func validateTypeSwitchGuardOperation(spec OperationSpec) error {
	if spec.Variant != catalog.VariantTypeSwitchGuard {
		return nil
	}
	if spec.Kind != OperationTypeAssert ||
		spec.Syntax != catalog.KindTypeAssertExpr ||
		spec.Mode != ValueModeNone ||
		spec.Arity != ResultArityZero ||
		spec.Place != PlaceNone ||
		!spec.ResultType.IsZero() ||
		!spec.ExpectedType.IsZero() ||
		spec.Addressable ||
		spec.Assignable ||
		spec.HasOk ||
		!spec.Constant.IsZero() ||
		spec.Object.Kind() != ObjectReferenceNone ||
		!spec.Selection.IsZero() ||
		!spec.Instance.IsZero() ||
		len(spec.Operands) != 1 ||
		len(spec.Definitions) != 0 ||
		len(spec.Implicit) != 0 ||
		!spec.ControlTarget.IsZero() ||
		!spec.Label.IsZero() {
		return fmt.Errorf(
			"type-switch guard operation %s has noncanonical semantics",
			spec.ID,
		)
	}
	return nil
}

func validateOperationOrigin(spec OperationSpec) error {
	if spec.ID.Source() {
		if !spec.Syntax.Valid() ||
			spec.ID.Occurrence().KindID() != uint16(spec.Syntax) ||
			!spec.Variant.Valid() ||
			!spec.Role.Valid() ||
			(spec.Token != catalog.TokenInvalid &&
				!spec.Token.Valid()) {
			return fmt.Errorf(
				"source operation %s has invalid source origin",
				spec.ID,
			)
		}
		return nil
	}
	if !spec.ID.ImplicitOp().Valid() ||
		spec.Kind != OperationPackageInitialization ||
		spec.Syntax != catalog.KindInvalid ||
		spec.Variant != catalog.VariantInvalid ||
		spec.Role != catalog.RoleInvalid ||
		spec.Token != catalog.TokenInvalid {
		return fmt.Errorf(
			"implicit operation %s has source-shaped origin fields",
			spec.ID,
		)
	}
	return nil
}

func operationNeedsResultType(mode ValueMode) bool {
	switch mode {
	case ValueModeValue,
		ValueModeType,
		ValueModeTuple,
		ValueModePlace:
		return true
	default:
		return false
	}
}

func cloneOperationSpec(spec OperationSpec) OperationSpec {
	spec.Operands = append(
		[]identity.OccurrenceID(nil), spec.Operands...,
	)
	spec.Definitions = append(
		[]identity.DefinitionID(nil), spec.Definitions...,
	)
	spec.Implicit = append(
		[]ImplicitOperation(nil), spec.Implicit...,
	)
	return spec
}

func (operation Operation) ID() identity.OperationID {
	return operation.spec.ID
}
func (operation Operation) Definition() identity.DefinitionID {
	return operation.spec.ID.Definition()
}
func (operation Operation) Occurrence() identity.OccurrenceID {
	return operation.spec.ID.Occurrence()
}
func (operation Operation) Kind() OperationKind {
	return operation.spec.Kind
}
func (operation Operation) Syntax() catalog.Kind {
	return operation.spec.Syntax
}
func (operation Operation) Variant() catalog.Variant {
	return operation.spec.Variant
}
func (operation Operation) Role() catalog.Role {
	return operation.spec.Role
}
func (operation Operation) Token() catalog.TokenKind {
	return operation.spec.Token
}
func (operation Operation) Mode() ValueMode {
	return operation.spec.Mode
}
func (operation Operation) Arity() ResultArity {
	return operation.spec.Arity
}
func (operation Operation) Place() PlaceKind {
	return operation.spec.Place
}
func (operation Operation) ResultType() identity.SemanticTypeID {
	return operation.spec.ResultType
}
func (operation Operation) ExpectedType() identity.SemanticTypeID {
	return operation.spec.ExpectedType
}
func (operation Operation) Addressable() bool {
	return operation.spec.Addressable
}
func (operation Operation) Assignable() bool {
	return operation.spec.Assignable
}
func (operation Operation) HasOk() bool {
	return operation.spec.HasOk
}
func (operation Operation) Constant() Constant {
	return operation.spec.Constant
}
func (operation Operation) Object() ObjectReference {
	return operation.spec.Object
}
func (operation Operation) Selection() Selection {
	return operation.spec.Selection
}
func (operation Operation) Instance() Instance {
	return operation.spec.Instance
}
func (operation Operation) ControlTarget() identity.OperationID {
	return operation.spec.ControlTarget
}
func (operation Operation) Label() identity.SemanticBindingID {
	return operation.spec.Label
}
func (operation Operation) OperandCount() int {
	return len(operation.spec.Operands)
}
func (operation Operation) Operand(
	index int,
) (identity.OccurrenceID, bool) {
	if index < 0 || index >= len(operation.spec.Operands) {
		return identity.OccurrenceID{}, false
	}
	return operation.spec.Operands[index], true
}
func (operation Operation) NestedDefinitionCount() int {
	return len(operation.spec.Definitions)
}
func (operation Operation) NestedDefinition(
	index int,
) (identity.DefinitionID, bool) {
	if index < 0 || index >= len(operation.spec.Definitions) {
		return identity.DefinitionID{}, false
	}
	return operation.spec.Definitions[index], true
}
func (operation Operation) ImplicitCount() int {
	return len(operation.spec.Implicit)
}
func (operation Operation) Implicit(
	index int,
) (ImplicitOperation, bool) {
	if index < 0 || index >= len(operation.spec.Implicit) {
		return ImplicitOperation{}, false
	}
	return operation.spec.Implicit[index], true
}
func (operation Operation) Spec() OperationSpec {
	return cloneOperationSpec(operation.spec)
}
