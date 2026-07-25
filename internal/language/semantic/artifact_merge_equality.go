package semantic

func equalDefinitionRecords(
	left DefinitionSemantics,
	right DefinitionSemantics,
) bool {
	a := left.Spec()
	b := right.Spec()
	return a.Definition == b.Definition &&
		a.Package == b.Package &&
		a.Form == b.Form &&
		a.Name == b.Name &&
		a.Signature == b.Signature &&
		a.Receiver == b.Receiver &&
		a.Implicit == b.Implicit &&
		equalComparableSlices(a.Declarations, b.Declarations) &&
		equalComparableSlices(a.Bindings, b.Bindings) &&
		equalComparableSlices(
			a.InitializerEntries,
			b.InitializerEntries,
		)
}

func equalResolutionRecords(
	left OccurrenceResolution,
	right OccurrenceResolution,
) bool {
	return left.Occurrence() == right.Occurrence() &&
		left.Owner() == right.Owner() &&
		left.Syntax() == right.Syntax() &&
		left.Role() == right.Role() &&
		left.Variant() == right.Variant() &&
		left.Domain() == right.Domain() &&
		left.Kind() == right.Kind() &&
		left.Structural() == right.Structural() &&
		left.Component() == right.Component() &&
		left.Definition() == right.Definition() &&
		left.Declaration() == right.Declaration() &&
		left.Binding() == right.Binding() &&
		left.Type() == right.Type() &&
		left.Operation() == right.Operation() &&
		left.Unsupported() == right.Unsupported()
}

func equalDeclarationRecords(
	left Declaration,
	right Declaration,
) bool {
	return left.ID() == right.ID() &&
		left.Package() == right.Package() &&
		left.Class() == right.Class() &&
		left.Name() == right.Name() &&
		left.Type() == right.Type() &&
		left.Exported() == right.Exported() &&
		left.Constant() == right.Constant()
}

func equalBindingRecords(left Binding, right Binding) bool {
	return left.ID() == right.ID() &&
		left.Package() == right.Package() &&
		left.Definition() == right.Definition() &&
		left.Role() == right.Role() &&
		left.Name() == right.Name() &&
		left.Type() == right.Type() &&
		left.Source() == right.Source() &&
		equalComparableSlices(
			left.CapturedBy(),
			right.CapturedBy(),
		)
}

func equalTypeRecords(left Type, right Type) bool {
	return left.Equal(right)
}

func equalOperationRecords(left Operation, right Operation) bool {
	if left.ID() != right.ID() ||
		left.Kind() != right.Kind() ||
		left.Syntax() != right.Syntax() ||
		left.Variant() != right.Variant() ||
		left.Role() != right.Role() ||
		left.Token() != right.Token() ||
		left.Mode() != right.Mode() ||
		left.Arity() != right.Arity() ||
		left.Place() != right.Place() ||
		left.ResultType() != right.ResultType() ||
		left.ExpectedType() != right.ExpectedType() ||
		left.Addressable() != right.Addressable() ||
		left.Assignable() != right.Assignable() ||
		left.HasOk() != right.HasOk() ||
		left.Constant() != right.Constant() ||
		left.Object() != right.Object() ||
		!equalSelections(left.Selection(), right.Selection()) ||
		!equalInstances(left.Instance(), right.Instance()) ||
		left.ControlTarget() != right.ControlTarget() ||
		left.Label() != right.Label() ||
		left.OperandCount() != right.OperandCount() ||
		left.NestedDefinitionCount() !=
			right.NestedDefinitionCount() ||
		left.ImplicitCount() != right.ImplicitCount() {
		return false
	}
	for index := 0; index < left.OperandCount(); index++ {
		leftValue, _ := left.Operand(index)
		rightValue, _ := right.Operand(index)
		if leftValue != rightValue {
			return false
		}
	}
	for index := 0; index < left.NestedDefinitionCount(); index++ {
		leftValue, _ := left.NestedDefinition(index)
		rightValue, _ := right.NestedDefinition(index)
		if leftValue != rightValue {
			return false
		}
	}
	for index := 0; index < left.ImplicitCount(); index++ {
		leftValue, _ := left.Implicit(index)
		rightValue, _ := right.Implicit(index)
		if leftValue != rightValue {
			return false
		}
	}
	return true
}

func equalSelections(left Selection, right Selection) bool {
	return left.kind == right.kind &&
		left.receiver == right.receiver &&
		left.object == right.object &&
		left.indirect == right.indirect &&
		equalComparableSlices(left.index, right.index)
}

func equalInstances(left Instance, right Instance) bool {
	return left.target == right.target &&
		left.signature == right.signature &&
		equalComparableSlices(left.types, right.types)
}

func equalUnsupportedRecords(
	left Unsupported,
	right Unsupported,
) bool {
	return left.ID() == right.ID() &&
		left.Reason() == right.Reason() &&
		left.Evidence() == right.Evidence()
}

func equalComparableSlices[Value comparable](
	left []Value,
	right []Value,
) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
