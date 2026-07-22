package scope

import (
	"bytes"
	"encoding/json"
	"os"

	"github.com/tsoniclang/gotots/internal/identity"
)

// contractArtifact is the on-disk JSON schema of a provider-contract
// artifact. Every field is explicit and closed-decoded; unknown selector,
// condition, provider, or namespace spellings fail.
type contractArtifact struct {
	ID      string         `json:"id"`
	Version int            `json:"version"`
	Rules   []ruleArtifact `json:"rules"`
}

// ruleArtifact is one rule's explicit serialized form.
type ruleArtifact struct {
	Bind      string `json:"bind"`                // "unit" | "package" | "namespace"
	Unit      string `json:"unit,omitempty"`      // canonical unit identity
	Package   string `json:"package,omitempty"`   // canonical package identity
	Namespace string `json:"namespace,omitempty"` // owner-class name
	Condition string `json:"condition"`           // explicit; no default
	Provider  string `json:"provider"`
}

// loadContractArtifact decodes and validates one contract-artifact file
// through the same validating constructors as built-in contracts.
func loadContractArtifact(path string) (ProviderContract, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ProviderContract{}, &SelectionError{Reason: "unknown provider contract " + path +
			": not a registry identity and not a readable artifact: " + err.Error()}
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var artifact contractArtifact
	if err := decoder.Decode(&artifact); err != nil {
		return ProviderContract{}, &SelectionError{Reason: "contract artifact " + path + " undecodable: " + err.Error()}
	}
	if artifact.Version != ContractVersion {
		return ProviderContract{}, &SelectionError{Reason: "contract artifact " + path + " declares unsupported schema version"}
	}
	rules := make([]ContractRule, 0, len(artifact.Rules))
	for _, r := range artifact.Rules {
		rule, err := decodeRule(r)
		if err != nil {
			return ProviderContract{}, err
		}
		rules = append(rules, rule)
	}
	return NewProviderContract(artifact.ID, rules)
}

// decodeRule closed-decodes one serialized rule.
func decodeRule(r ruleArtifact) (ContractRule, error) {
	condition, err := conditionByName(r.Condition)
	if err != nil {
		return ContractRule{}, err
	}
	provider, err := providerByName(r.Provider)
	if err != nil {
		return ContractRule{}, err
	}
	switch r.Bind {
	case "unit":
		ref, err := identity.ParseUnitRef(r.Unit)
		if err != nil {
			return ContractRule{}, err
		}
		return NewExactUnitRule(ref, condition, provider)
	case "package":
		pkg, err := identity.ParsePackageID(r.Package)
		if err != nil {
			return ContractRule{}, err
		}
		return NewExactPackageRule(pkg, condition, provider)
	case "namespace":
		class, err := ownerClassByName(r.Namespace)
		if err != nil {
			return ContractRule{}, err
		}
		return NewNamespaceRule(class, condition, provider)
	}
	return ContractRule{}, &SelectionError{Reason: "contract rule declares unknown selector " + r.Bind}
}

// conditionByName maps a serialized condition name to its closed value.
func conditionByName(name string) (EvidenceCondition, error) {
	for c := EvidenceCondition(1); c < numConditions; c++ {
		if conditionNames[c] == name {
			return c, nil
		}
	}
	return ConditionInvalid, &SelectionError{Reason: "contract rule declares unknown condition " + name}
}

// providerByName maps a serialized provider name to its closed value.
func providerByName(name string) (Provider, error) {
	for p := Provider(1); p < numProviders; p++ {
		if providerNames[p] == name {
			return p, nil
		}
	}
	return ProviderInvalid, &SelectionError{Reason: "contract rule declares unknown provider " + name}
}

// ownerClassByName maps a serialized owner-class name to its closed value.
func ownerClassByName(name string) (identity.OwnerClass, error) {
	for c := identity.OwnerClass(1); c.Valid(); c++ {
		if c.String() == name {
			return c, nil
		}
	}
	return 0, &SelectionError{Reason: "contract rule declares unknown namespace " + name}
}
