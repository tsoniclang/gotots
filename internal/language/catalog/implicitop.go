package catalog

import "fmt"

// ImplicitOp is the closed catalog of implicit Go operations — behavior the
// source performs without spelling it. Values are explicit and permanent.
type ImplicitOp uint16

// Explicit, permanent implicit-operation identities. Do not renumber; append
// only.
const (
	ImplicitInvalid ImplicitOp = 0

	ImplicitZeroing              ImplicitOp = 1
	ImplicitValueCopy            ImplicitOp = 2
	ImplicitAssignmentConversion ImplicitOp = 3
	ImplicitReceiverAdjustment   ImplicitOp = 4
	ImplicitMethodPromotion      ImplicitOp = 5
	ImplicitInterfaceConversion  ImplicitOp = 6
	ImplicitBoxing               ImplicitOp = 7
	ImplicitInitialization       ImplicitOp = 8
	ImplicitEvaluationOrder      ImplicitOp = 9
	ImplicitPanicBoundary        ImplicitOp = 10

	// implicitOpCount is the highest assigned identity; append-only.
	implicitOpCount = 10
)

// ImplicitOwner is the closed statement of which pipeline stage produces
// occurrences of one implicit operation.
type ImplicitOwner uint8

const (
	ImplicitOwnerInvalid ImplicitOwner = iota
	// ImplicitOwnerInventory: the construct inventory detects the operation
	// from syntax plus go/types evidence.
	ImplicitOwnerInventory
	// ImplicitOwnerSemanticModel: the operation is produced by the semantic
	// model phase, which owns whole-body evaluation evidence.
	ImplicitOwnerSemanticModel

	numImplicitOwners
)

var implicitOwnerNames = [numImplicitOwners]string{
	ImplicitOwnerInventory:     "inventory-detected",
	ImplicitOwnerSemanticModel: "semantic-model-owned",
}

// Valid reports whether o names an owner.
func (o ImplicitOwner) Valid() bool { return o > ImplicitOwnerInvalid && o < numImplicitOwners }

// String renders o for reports.
func (o ImplicitOwner) String() string {
	if o.Valid() {
		return implicitOwnerNames[o]
	}
	return fmt.Sprintf("catalog.ImplicitOwner(%d)", uint8(o))
}

type implicitDescriptor struct {
	name     string
	owner    ImplicitOwner
	evidence string // the required typed evidence for one detection
}

var implicitOps = [implicitOpCount + 1]implicitDescriptor{
	ImplicitZeroing:              {"zeroing", ImplicitOwnerInventory, "var declaration binding without initializer"},
	ImplicitValueCopy:            {"value-copy", ImplicitOwnerInventory, "struct/array-typed value crossing an assignment or call boundary"},
	ImplicitAssignmentConversion: {"assignment-conversion", ImplicitOwnerSemanticModel, "assignability step with distinct source/target types"},
	ImplicitReceiverAdjustment:   {"receiver-adjustment", ImplicitOwnerInventory, "method selection with go/types-reported indirection"},
	ImplicitMethodPromotion:      {"method-promotion", ImplicitOwnerInventory, "selection index path longer than one"},
	ImplicitInterfaceConversion:  {"interface-conversion", ImplicitOwnerInventory, "non-interface operand meeting an interface-typed target"},
	ImplicitBoxing:               {"boxing", ImplicitOwnerSemanticModel, "value representation change at a dynamic-type boundary"},
	ImplicitInitialization:       {"initialization", ImplicitOwnerSemanticModel, "package initialization ordering edges"},
	ImplicitEvaluationOrder:      {"evaluation-order", ImplicitOwnerSemanticModel, "multi-operand sequencing constraints"},
	ImplicitPanicBoundary:        {"panic-boundary", ImplicitOwnerSemanticModel, "operation with a defined runtime panic condition"},
}

// Valid reports whether op names an implicit operation.
func (op ImplicitOp) Valid() bool { return op >= 1 && op <= implicitOpCount }

// Name is the stable descriptive name.
func (op ImplicitOp) Name() string {
	if !op.Valid() {
		return ""
	}
	return implicitOps[op].name
}

// Owner is the pipeline stage that produces occurrences of op.
func (op ImplicitOp) Owner() ImplicitOwner {
	if !op.Valid() {
		return ImplicitOwnerInvalid
	}
	return implicitOps[op].owner
}

// Evidence is the required typed evidence for one detection of op.
func (op ImplicitOp) Evidence() string {
	if !op.Valid() {
		return ""
	}
	return implicitOps[op].evidence
}

// String renders op for reports.
func (op ImplicitOp) String() string {
	if name := op.Name(); name != "" {
		return name
	}
	return fmt.Sprintf("catalog.ImplicitOp(%d)", uint16(op))
}

// AllImplicitOps returns every implicit operation in ascending identity order.
func AllImplicitOps() []ImplicitOp {
	out := make([]ImplicitOp, 0, implicitOpCount)
	for id := 1; id <= implicitOpCount; id++ {
		out = append(out, ImplicitOp(id))
	}
	return out
}
