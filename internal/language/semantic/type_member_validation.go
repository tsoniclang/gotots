package semantic

import "fmt"

type typeSpecFields uint32

const (
	typeSpecBasic typeSpecFields = 1 << iota
	typeSpecDeclaration
	typeSpecParameter
	typeSpecArguments
	typeSpecUnderlying
	typeSpecTarget
	typeSpecConstraint
	typeSpecElement
	typeSpecKey
	typeSpecLength
	typeSpecDirection
	typeSpecSignature
	typeSpecFieldsValue
	typeSpecMethods
	typeSpecEmbeddeds
	typeSpecTerms
	typeSpecTypeSet
	typeSpecComparable
	typeSpecElements
)

func validateTypeMembers(spec TypeSpec) error {
	allowed := allowedTypeSpecFields(spec.Kind)
	if allowed == 0 {
		return fmt.Errorf("invalid type kind")
	}
	forbidden := presentTypeSpecFields(spec) &^ allowed
	if forbidden == 0 {
		return nil
	}
	return fmt.Errorf(
		"%s type carries forbidden %s field",
		spec.Kind,
		firstTypeSpecFieldName(forbidden),
	)
}

func allowedTypeSpecFields(kind TypeKind) typeSpecFields {
	switch kind {
	case TypeBasic:
		return typeSpecBasic
	case TypeNamed:
		return typeSpecDeclaration |
			typeSpecArguments |
			typeSpecUnderlying |
			typeSpecMethods
	case TypeAlias:
		return typeSpecDeclaration |
			typeSpecArguments |
			typeSpecTarget
	case TypeParameter:
		return typeSpecParameter | typeSpecConstraint
	case TypePointer, TypeSlice:
		return typeSpecElement
	case TypeArray:
		return typeSpecElement | typeSpecLength
	case TypeMap:
		return typeSpecKey | typeSpecElement
	case TypeChannel:
		return typeSpecElement | typeSpecDirection
	case TypeSignature:
		return typeSpecSignature
	case TypeStruct:
		return typeSpecFieldsValue
	case TypeInterface:
		return typeSpecMethods |
			typeSpecEmbeddeds |
			typeSpecTerms |
			typeSpecTypeSet |
			typeSpecComparable
	case TypeTuple:
		return typeSpecElements
	case TypeUnion:
		return typeSpecTerms
	default:
		return 0
	}
}

func presentTypeSpecFields(spec TypeSpec) typeSpecFields {
	var fields typeSpecFields
	if spec.Basic != BasicInvalid {
		fields |= typeSpecBasic
	}
	if !spec.Declaration.IsZero() {
		fields |= typeSpecDeclaration
	}
	if !spec.Parameter.IsZero() {
		fields |= typeSpecParameter
	}
	if len(spec.Arguments) != 0 {
		fields |= typeSpecArguments
	}
	if !spec.Underlying.IsZero() {
		fields |= typeSpecUnderlying
	}
	if !spec.Target.IsZero() {
		fields |= typeSpecTarget
	}
	if !spec.Constraint.IsZero() {
		fields |= typeSpecConstraint
	}
	if !spec.Element.IsZero() {
		fields |= typeSpecElement
	}
	if !spec.Key.IsZero() {
		fields |= typeSpecKey
	}
	if spec.Length != 0 {
		fields |= typeSpecLength
	}
	if spec.Direction != ChannelInvalid {
		fields |= typeSpecDirection
	}
	if signaturePresent(spec.Signature) {
		fields |= typeSpecSignature
	}
	if len(spec.Fields) != 0 {
		fields |= typeSpecFieldsValue
	}
	if len(spec.Methods) != 0 {
		fields |= typeSpecMethods
	}
	if len(spec.Embeddeds) != 0 {
		fields |= typeSpecEmbeddeds
	}
	if len(spec.Terms) != 0 {
		fields |= typeSpecTerms
	}
	if spec.TypeSet != TypeSetInvalid {
		fields |= typeSpecTypeSet
	}
	if spec.Comparable {
		fields |= typeSpecComparable
	}
	if len(spec.Elements) != 0 {
		fields |= typeSpecElements
	}
	return fields
}

func firstTypeSpecFieldName(fields typeSpecFields) string {
	names := [...]struct {
		field typeSpecFields
		name  string
	}{
		{typeSpecBasic, "basic"},
		{typeSpecDeclaration, "declaration"},
		{typeSpecParameter, "parameter"},
		{typeSpecArguments, "arguments"},
		{typeSpecUnderlying, "underlying"},
		{typeSpecTarget, "target"},
		{typeSpecConstraint, "constraint"},
		{typeSpecElement, "element"},
		{typeSpecKey, "key"},
		{typeSpecLength, "length"},
		{typeSpecDirection, "direction"},
		{typeSpecSignature, "signature"},
		{typeSpecFieldsValue, "fields"},
		{typeSpecMethods, "methods"},
		{typeSpecEmbeddeds, "embeddeds"},
		{typeSpecTerms, "terms"},
		{typeSpecTypeSet, "type-set"},
		{typeSpecComparable, "comparable"},
		{typeSpecElements, "elements"},
	}
	for _, candidate := range names {
		if fields&candidate.field != 0 {
			return candidate.name
		}
	}
	return "unknown"
}
