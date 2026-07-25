package identity

import (
	"strconv"
	"strings"
)

func ParseSemanticTypeID(value string) (SemanticTypeID, error) {
	const prefix = "semantic-type/sha256:"
	if !strings.HasPrefix(value, prefix) {
		return SemanticTypeID{}, semanticParseError(
			"semantic-type", value, "missing canonical digest prefix",
		)
	}
	id, err := NewSemanticTypeID(strings.TrimPrefix(value, prefix))
	if err != nil {
		return SemanticTypeID{}, err
	}
	if id.String() != value {
		return SemanticTypeID{}, semanticParseError(
			"semantic-type", value, "serialization is not canonical",
		)
	}
	return id, nil
}

func ParseSemanticDeclarationID(
	value string,
) (SemanticDeclarationID, error) {
	switch {
	case strings.HasPrefix(value, "lang#predeclared/"):
		return parsePredeclaredDeclaration(value)
	case strings.Contains(value, "#local-declaration/"):
		return parseOccurrenceDeclaration(value)
	case strings.Contains(value, "#declaration/"):
		return parsePackageDeclaration(value)
	case strings.Contains(value, "#member/"):
		return parseMemberDeclaration(value)
	default:
		return SemanticDeclarationID{}, semanticParseError(
			"semantic-declaration", value,
			"not a canonical semantic declaration",
		)
	}
}

func parsePackageDeclaration(
	value string,
) (SemanticDeclarationID, error) {
	const marker = "#declaration/"
	index := strings.LastIndex(value, marker)
	if index < 0 {
		return SemanticDeclarationID{}, semanticParseError(
			"semantic-declaration", value, "missing declaration marker",
		)
	}
	pkg, err := ParsePackageID(value[:index])
	if err != nil {
		return SemanticDeclarationID{}, err
	}
	parts := strings.SplitN(value[index+len(marker):], "/", 2)
	if len(parts) != 2 {
		return SemanticDeclarationID{}, semanticParseError(
			"semantic-declaration", value,
			"malformed package declaration",
		)
	}
	class := parseSemanticObjectClass(parts[0])
	id, err := NewPackageDeclarationID(pkg, class, parts[1])
	return canonicalDeclaration(value, id, err)
}

func parseMemberDeclaration(
	value string,
) (SemanticDeclarationID, error) {
	const marker = "#member/"
	index := strings.LastIndex(value, marker)
	if index < 0 {
		return SemanticDeclarationID{}, semanticParseError(
			"semantic-declaration", value, "missing member marker",
		)
	}
	owner, err := ParseSemanticTypeID(value[:index])
	if err != nil {
		return SemanticDeclarationID{}, err
	}
	payload := value[index+len(marker):]
	for _, class := range []SemanticObjectClass{
		SemanticObjectField,
		SemanticObjectMethod,
	} {
		classMarker := "/" + class.String() + "/"
		classIndex := strings.LastIndex(payload, classMarker)
		if classIndex < 0 {
			continue
		}
		namespace := payload[:classIndex]
		rest := payload[classIndex+len(classMarker):]
		memberPackage := PackageID{}
		if namespace != "exported" {
			memberPackage, err = ParsePackageID(namespace)
			if err != nil {
				return SemanticDeclarationID{}, err
			}
		}
		ordinal := 0
		name := rest
		if class == SemanticObjectField {
			parts := strings.SplitN(rest, "/", 2)
			if len(parts) != 2 {
				break
			}
			ordinal, err = strconv.Atoi(parts[0])
			if err != nil {
				break
			}
			name = parts[1]
		}
		id, constructErr := NewMemberDeclarationID(
			owner, memberPackage, class, name, ordinal,
		)
		return canonicalDeclaration(value, id, constructErr)
	}
	return SemanticDeclarationID{}, semanticParseError(
		"semantic-declaration", value, "malformed member declaration",
	)
}

func parsePredeclaredDeclaration(
	value string,
) (SemanticDeclarationID, error) {
	parts := strings.Split(
		strings.TrimPrefix(value, "lang#predeclared/"), "/",
	)
	if len(parts) != 2 {
		return SemanticDeclarationID{}, semanticParseError(
			"semantic-declaration", value,
			"malformed predeclared declaration",
		)
	}
	predeclared, err := strconv.ParseUint(parts[0], 10, 16)
	if err != nil {
		return SemanticDeclarationID{}, semanticParseError(
			"semantic-declaration", value,
			"malformed predeclared identity",
		)
	}
	id, err := NewPredeclaredDeclarationID(
		uint16(predeclared), parseSemanticObjectClass(parts[1]),
	)
	return canonicalDeclaration(value, id, err)
}

func parseOccurrenceDeclaration(
	value string,
) (SemanticDeclarationID, error) {
	const marker = "#local-declaration/"
	index := strings.LastIndex(value, marker)
	if index < 0 {
		return SemanticDeclarationID{}, semanticParseError(
			"semantic-declaration", value,
			"missing local-declaration marker",
		)
	}
	owner, err := ParseOccurrenceID(value[:index])
	if err != nil {
		return SemanticDeclarationID{}, err
	}
	parts := strings.SplitN(value[index+len(marker):], "/", 4)
	if len(parts) != 4 {
		return SemanticDeclarationID{}, semanticParseError(
			"semantic-declaration", value,
			"malformed local declaration",
		)
	}
	ordinal, err := strconv.Atoi(parts[1])
	if err != nil {
		return SemanticDeclarationID{}, semanticParseError(
			"semantic-declaration", value,
			"malformed local declaration ordinal",
		)
	}
	occurrence, err := ParseOccurrenceID(parts[3])
	if err != nil {
		return SemanticDeclarationID{}, err
	}
	id, err := NewOccurrenceDeclarationID(
		owner,
		occurrence,
		parseSemanticObjectClass(parts[0]),
		parts[2],
		ordinal,
	)
	return canonicalDeclaration(value, id, err)
}

func ParseSemanticBindingID(
	value string,
) (SemanticBindingID, error) {
	const marker = "#binding/"
	index := strings.LastIndex(value, marker)
	if index < 0 {
		return SemanticBindingID{}, semanticParseError(
			"semantic-binding", value, "missing binding marker",
		)
	}
	owner, err := ParseOccurrenceID(value[:index])
	if err != nil {
		return SemanticBindingID{}, err
	}
	parts := strings.SplitN(value[index+len(marker):], "/", 3)
	if len(parts) != 3 {
		return SemanticBindingID{}, semanticParseError(
			"semantic-binding", value, "malformed binding",
		)
	}
	role := parseSemanticBindingRole(parts[0])
	ordinal, err := strconv.Atoi(parts[1])
	if err != nil {
		return SemanticBindingID{}, semanticParseError(
			"semantic-binding", value, "malformed binding ordinal",
		)
	}
	declaration := OccurrenceID{}
	if parts[2] != "unnamed" {
		declaration, err = ParseOccurrenceID(parts[2])
		if err != nil {
			return SemanticBindingID{}, err
		}
	}
	id, err := NewSemanticBindingID(
		owner, declaration, role, ordinal,
	)
	if err != nil {
		return SemanticBindingID{}, err
	}
	if id.String() != value {
		return SemanticBindingID{}, semanticParseError(
			"semantic-binding", value,
			"serialization is not canonical",
		)
	}
	return id, nil
}

func ParseOperationID(value string) (OperationID, error) {
	const marker = "#operation/"
	index := strings.LastIndex(value, marker)
	if index < 0 {
		return OperationID{}, semanticParseError(
			"operation", value, "missing operation marker",
		)
	}
	definition, err := ParseDefinitionID(value[:index])
	if err != nil {
		return OperationID{}, err
	}
	payload := value[index+len(marker):]
	var id OperationID
	if strings.HasPrefix(payload, "implicit/") {
		parts := strings.Split(
			strings.TrimPrefix(payload, "implicit/"), "/",
		)
		if len(parts) != 2 {
			return OperationID{}, semanticParseError(
				"operation", value,
				"malformed implicit operation",
			)
		}
		operation := parseImplicitDefinitionOp(parts[0])
		ordinal, parseErr := strconv.Atoi(parts[1])
		if parseErr != nil {
			return OperationID{}, semanticParseError(
				"operation", value,
				"malformed implicit operation ordinal",
			)
		}
		id, err = NewImplicitOperationID(
			definition, operation, ordinal,
		)
	} else {
		var occurrence OccurrenceID
		occurrence, err = ParseOccurrenceID(payload)
		if err == nil {
			id, err = NewOperationID(definition, occurrence)
		}
	}
	if err != nil {
		return OperationID{}, err
	}
	if id.String() != value {
		return OperationID{}, semanticParseError(
			"operation", value, "serialization is not canonical",
		)
	}
	return id, nil
}

func ParseUnsupportedID(value string) (UnsupportedID, error) {
	const marker = "#unsupported/"
	index := strings.LastIndex(value, marker)
	if index < 0 {
		return UnsupportedID{}, semanticParseError(
			"unsupported", value, "missing unsupported marker",
		)
	}
	definition, err := ParseDefinitionID(value[:index])
	if err != nil {
		return UnsupportedID{}, err
	}
	occurrence, err := ParseOccurrenceID(value[index+len(marker):])
	if err != nil {
		return UnsupportedID{}, err
	}
	id, err := NewUnsupportedID(definition, occurrence)
	if err != nil {
		return UnsupportedID{}, err
	}
	if id.String() != value {
		return UnsupportedID{}, semanticParseError(
			"unsupported", value,
			"serialization is not canonical",
		)
	}
	return id, nil
}

func parseSemanticObjectClass(
	value string,
) SemanticObjectClass {
	for class := SemanticObjectClass(1); class.Valid(); class++ {
		if class.String() == value {
			return class
		}
	}
	return SemanticObjectInvalid
}

func parseSemanticBindingRole(
	value string,
) SemanticBindingRole {
	for role := SemanticBindingRole(1); role.Valid(); role++ {
		if role.String() == value {
			return role
		}
	}
	return SemanticBindingInvalid
}

func parseImplicitDefinitionOp(
	value string,
) ImplicitDefinitionOp {
	for operation := ImplicitDefinitionOp(1); operation.Valid(); operation++ {
		if operation.String() == value {
			return operation
		}
	}
	return ImplicitDefinitionInvalid
}

func canonicalDeclaration(
	value string,
	id SemanticDeclarationID,
	err error,
) (SemanticDeclarationID, error) {
	if err != nil {
		return SemanticDeclarationID{}, err
	}
	if id.String() != value {
		return SemanticDeclarationID{}, semanticParseError(
			"semantic-declaration", value,
			"serialization is not canonical",
		)
	}
	return id, nil
}

func semanticParseError(
	kind string,
	value string,
	reason string,
) error {
	return &Error{
		Identity: kind,
		Value:    value,
		Reason:   reason,
	}
}
