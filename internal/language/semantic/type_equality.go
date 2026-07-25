package semantic

import "slices"

// Equal reports exact semantic descriptor equality without constructing a
// rendered canonical form.
func (record Type) Equal(other Type) bool {
	return record.id == other.id &&
		equalTypeSpec(record.spec, other.spec)
}

func equalTypeSpec(left TypeSpec, right TypeSpec) bool {
	return left.Kind == right.Kind &&
		left.Basic == right.Basic &&
		left.Declaration == right.Declaration &&
		left.Parameter == right.Parameter &&
		slices.Equal(left.Arguments, right.Arguments) &&
		left.Underlying == right.Underlying &&
		left.Target == right.Target &&
		left.Constraint == right.Constraint &&
		left.Element == right.Element &&
		left.Key == right.Key &&
		left.Length == right.Length &&
		left.Direction == right.Direction &&
		equalSignatures(left.Signature, right.Signature) &&
		slices.Equal(left.Fields, right.Fields) &&
		slices.Equal(left.Methods, right.Methods) &&
		slices.Equal(left.Embeddeds, right.Embeddeds) &&
		slices.Equal(left.Terms, right.Terms) &&
		left.TypeSet == right.TypeSet &&
		left.Comparable == right.Comparable &&
		slices.Equal(left.Elements, right.Elements)
}

func equalSignatures(left Signature, right Signature) bool {
	return left.Receiver == right.Receiver &&
		slices.Equal(
			left.ReceiverTypeParameters,
			right.ReceiverTypeParameters,
		) &&
		slices.Equal(left.TypeParameters, right.TypeParameters) &&
		slices.Equal(left.Parameters, right.Parameters) &&
		slices.Equal(left.Results, right.Results) &&
		left.Variadic == right.Variadic
}
