package semantic

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/tsoniclang/gotots/internal/identity"
)

type artifactError struct {
	reason string
}

func (err *artifactError) Error() string {
	return "GOTOTS_SEMANTIC_PROVIDER_ARTIFACT: " + err.reason
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

func requireJSONEnd(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf(
				"semantic provider manifest has trailing JSON",
			)
		}
		return err
	}
	return nil
}
