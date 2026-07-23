package semantic

import "github.com/tsoniclang/gotots/internal/identity"

type artifactError struct {
	reason string
}

func (err *artifactError) Error() string {
	return "GOTOTS_SEMANTIC_PROVIDER_ARTIFACT: " + err.reason
}

func parseOptionalPackage(
	value string,
) (identity.PackageID, error) {
	if value == "" {
		return identity.PackageID{}, nil
	}
	return identity.ParsePackageID(value)
}

func parseOptionalDefinition(
	value string,
) (identity.DefinitionID, error) {
	if value == "" {
		return identity.DefinitionID{}, nil
	}
	return identity.ParseDefinitionID(value)
}

func parseOptionalOccurrence(
	value string,
) (identity.OccurrenceID, error) {
	if value == "" {
		return identity.OccurrenceID{}, nil
	}
	return identity.ParseOccurrenceID(value)
}

func parseOptionalDeclaration(
	value string,
) (identity.SemanticDeclarationID, error) {
	if value == "" {
		return identity.SemanticDeclarationID{}, nil
	}
	return identity.ParseSemanticDeclarationID(value)
}

func parseOptionalBinding(
	value string,
) (identity.SemanticBindingID, error) {
	if value == "" {
		return identity.SemanticBindingID{}, nil
	}
	return identity.ParseSemanticBindingID(value)
}

func parseOptionalType(
	value string,
) (identity.SemanticTypeID, error) {
	if value == "" {
		return identity.SemanticTypeID{}, nil
	}
	return identity.ParseSemanticTypeID(value)
}

func parseOptionalOperation(
	value string,
) (identity.OperationID, error) {
	if value == "" {
		return identity.OperationID{}, nil
	}
	return identity.ParseOperationID(value)
}

func parseOptionalUnsupported(
	value string,
) (identity.UnsupportedID, error) {
	if value == "" {
		return identity.UnsupportedID{}, nil
	}
	return identity.ParseUnsupportedID(value)
}

func parseDefinitions(
	values []string,
) ([]identity.DefinitionID, error) {
	out := make([]identity.DefinitionID, 0, len(values))
	for _, value := range values {
		id, err := identity.ParseDefinitionID(value)
		if err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
}

func parseOccurrences(
	values []string,
) ([]identity.OccurrenceID, error) {
	out := make([]identity.OccurrenceID, 0, len(values))
	for _, value := range values {
		id, err := identity.ParseOccurrenceID(value)
		if err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
}

func parseDeclarations(
	values []string,
) ([]identity.SemanticDeclarationID, error) {
	out := make(
		[]identity.SemanticDeclarationID, 0, len(values),
	)
	for _, value := range values {
		id, err := identity.ParseSemanticDeclarationID(value)
		if err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
}

func parseBindings(
	values []string,
) ([]identity.SemanticBindingID, error) {
	out := make([]identity.SemanticBindingID, 0, len(values))
	for _, value := range values {
		id, err := identity.ParseSemanticBindingID(value)
		if err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
}

func parseTypes(
	values []string,
) ([]identity.SemanticTypeID, error) {
	out := make([]identity.SemanticTypeID, 0, len(values))
	for _, value := range values {
		id, err := identity.ParseSemanticTypeID(value)
		if err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
}
