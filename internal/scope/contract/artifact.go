package contract

import (
	"bytes"
	"encoding/json"
	"io"
	"os"

	"github.com/tsoniclang/gotots/internal/identity"
)

type artifact struct {
	ID      string         `json:"id"`
	Version int            `json:"version"`
	Rules   []artifactRule `json:"rules"`
}

type artifactRule struct {
	Bind       string `json:"bind"`
	Definition string `json:"definition,omitempty"`
	Package    string `json:"package,omitempty"`
	Namespace  string `json:"namespace,omitempty"`
	Condition  string `json:"condition"`
	Fact       string `json:"fact,omitempty"`
	Provider   string `json:"provider"`
}

// Resolve loads the exact request-selected contract. Empty identity is an
// error; an artifact path never substitutes for an unknown identity.
func Resolve(id, digest, artifactPath string) (Contract, error) {
	if id == "" {
		return Contract{}, &Error{Reason: "compilation request selects no provider contract"}
	}
	var selected Contract
	var err error
	if artifactPath == "" {
		if id != DefaultID {
			return Contract{}, &Error{Reason: "unknown built-in contract " + id}
		}
		selected, err = Default()
	} else {
		selected, err = decodeArtifact(artifactPath)
		if err == nil && selected.ID() != id {
			err = &Error{Reason: "artifact declares " + selected.ID() + ", request selects " + id}
		}
	}
	if err != nil {
		return Contract{}, err
	}
	if digest != "" && digest != selected.Fingerprint() {
		return Contract{}, &Error{Reason: "contract digest mismatch"}
	}
	return selected, nil
}

func decodeArtifact(path string) (Contract, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Contract{}, &Error{Reason: "contract artifact unreadable: " + err.Error()}
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var stored artifact
	if err := decoder.Decode(&stored); err != nil {
		return Contract{}, &Error{Reason: "contract artifact undecodable: " + err.Error()}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return Contract{}, &Error{Reason: "contract artifact contains trailing data"}
	}
	if stored.Version != SchemaVersion {
		return Contract{}, &Error{Reason: "unsupported contract schema version"}
	}
	rules := make([]Rule, 0, len(stored.Rules))
	for _, encoded := range stored.Rules {
		rule, err := decodeRule(encoded)
		if err != nil {
			return Contract{}, err
		}
		rules = append(rules, rule)
	}
	return New(stored.ID, rules)
}

func decodeRule(encoded artifactRule) (Rule, error) {
	condition := ConditionInvalid
	for candidate := Condition(1); candidate.Valid(); candidate++ {
		if candidate.String() == encoded.Condition {
			condition = candidate
			break
		}
	}
	if !condition.Valid() {
		return Rule{}, &Error{Reason: "unknown condition " + encoded.Condition}
	}
	fact := SelectionFactInvalid
	if encoded.Fact != "" {
		for candidate := SelectionFactKind(1); candidate.Valid(); candidate++ {
			if candidate.String() == encoded.Fact {
				fact = candidate
				break
			}
		}
		if !fact.Valid() {
			return Rule{}, &Error{Reason: "unknown fact " + encoded.Fact}
		}
	}
	provider := ProviderInvalid
	for candidate := Provider(1); candidate.Valid(); candidate++ {
		if candidate.String() == encoded.Provider {
			provider = candidate
			break
		}
	}
	if !provider.Valid() {
		return Rule{}, &Error{Reason: "unknown provider " + encoded.Provider}
	}
	switch encoded.Bind {
	case "definition":
		if encoded.Package != "" || encoded.Namespace != "" {
			return Rule{}, &Error{Reason: "definition rule carries fields from another selector"}
		}
		definition, err := identity.ParseDefinitionID(encoded.Definition)
		if err != nil {
			return Rule{}, err
		}
		return NewDefinitionRule(definition, condition, fact, provider)
	case "package":
		if encoded.Definition != "" || encoded.Namespace != "" {
			return Rule{}, &Error{Reason: "package rule carries fields from another selector"}
		}
		pkg, err := identity.ParsePackageID(encoded.Package)
		if err != nil {
			return Rule{}, err
		}
		return NewPackageRule(pkg, condition, fact, provider)
	case "namespace":
		if encoded.Definition != "" || encoded.Package != "" {
			return Rule{}, &Error{Reason: "namespace rule carries fields from another selector"}
		}
		var owner identity.OwnerClass
		for candidate := identity.OwnerClass(1); candidate.Valid(); candidate++ {
			if candidate.String() == encoded.Namespace {
				owner = candidate
				break
			}
		}
		return NewNamespaceRule(owner, condition, fact, provider)
	default:
		return Rule{}, &Error{Reason: "unknown selector " + encoded.Bind}
	}
}
