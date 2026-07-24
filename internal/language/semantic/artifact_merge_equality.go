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
	return left.ID() == right.ID() &&
		left.Canonical() == right.Canonical()
}

func equalOperationRecords(left Operation, right Operation) bool {
	a := left.Spec()
	b := right.Spec()
	return a.ID == b.ID &&
		a.Kind == b.Kind &&
		a.Syntax == b.Syntax &&
		a.Variant == b.Variant &&
		a.Role == b.Role &&
		a.Token == b.Token &&
		a.Mode == b.Mode &&
		a.Arity == b.Arity &&
		a.Place == b.Place &&
		a.ResultType == b.ResultType &&
		a.ExpectedType == b.ExpectedType &&
		a.Addressable == b.Addressable &&
		a.Assignable == b.Assignable &&
		a.HasOk == b.HasOk &&
		a.Constant == b.Constant &&
		a.Object == b.Object &&
		equalSelections(a.Selection, b.Selection) &&
		equalInstances(a.Instance, b.Instance) &&
		equalComparableSlices(a.Operands, b.Operands) &&
		equalComparableSlices(a.Definitions, b.Definitions) &&
		equalComparableSlices(a.Implicit, b.Implicit) &&
		a.ControlTarget == b.ControlTarget &&
		a.Label == b.Label
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
