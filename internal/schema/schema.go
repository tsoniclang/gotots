// Package schema owns the machine-contract layer: a strict, closed
// subset of JSON Schema for every durable report and product input, a
// canonical loader, and fail-closed validation. Unknown schema keywords,
// unknown document fields under closed objects, and trailing data are
// rejected — validation never narrows to the understood subset silently.
package schema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// Schema is one node of the strict subset: explicit types, closed
// objects, closed enums, and recursive items/properties.
type Schema struct {
	// Type is one of: object, array, string, integer, number, boolean.
	Type string `json:"type"`
	// Properties and Required define object shapes. Objects are closed:
	// AdditionalProperties must be present and false for object types
	// unless MapValues is set (a homogeneous string-keyed map).
	Properties           map[string]*Schema `json:"properties,omitempty"`
	Required             []string           `json:"required,omitempty"`
	AdditionalProperties *bool              `json:"additionalProperties,omitempty"`
	// MapValues validates every value of a string-keyed map object.
	MapValues *Schema `json:"mapValues,omitempty"`
	// Items validates every element of an array.
	Items *Schema `json:"items,omitempty"`
	// Enum closes a string type over exact values.
	Enum []string `json:"enum,omitempty"`
	// Const requires one exact value (numbers compare exactly).
	Const *json.RawMessage `json:"const,omitempty"`
	// Pattern names a reviewed named pattern (never a raw regex):
	// sha256-hex, module-relative-path, canonical-id, or semver-ish.
	Pattern string `json:"pattern,omitempty"`
	// MinItems guards identity-bearing arrays that must not be empty.
	MinItems *int `json:"minItems,omitempty"`
}

// Contract is one entry of the schema manifest.
type Contract struct {
	ID            string `json:"id"`
	Path          string `json:"path"`
	Contract      string `json:"contract"`
	SchemaVersion int    `json:"schemaVersion"`
}

// Manifest is the exact schema inventory.
type Manifest struct {
	SchemaVersion int        `json:"schemaVersion"`
	Schemas       []Contract `json:"schemas"`
}

// LoadManifest reads and structurally validates schemas/manifest.json.
func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read schema manifest: %w", err)
	}
	var manifest Manifest
	if err := strictDecode(data, &manifest); err != nil {
		return nil, fmt.Errorf("schema manifest %s: %w", path, err)
	}
	if manifest.SchemaVersion != 1 {
		return nil, fmt.Errorf("schema manifest %s: unsupported schemaVersion %d", path, manifest.SchemaVersion)
	}
	seen := map[string]bool{}
	for _, contract := range manifest.Schemas {
		if contract.ID == "" || contract.Path == "" || contract.Contract == "" || contract.SchemaVersion < 1 {
			return nil, fmt.Errorf("schema manifest %s: contract %+v is incomplete", path, contract)
		}
		if seen[contract.ID] {
			return nil, fmt.Errorf("schema manifest %s: duplicate contract id %q", path, contract.ID)
		}
		seen[contract.ID] = true
	}
	sorted := sort.SliceIsSorted(manifest.Schemas, func(i, j int) bool {
		return manifest.Schemas[i].ID < manifest.Schemas[j].ID
	})
	if !sorted {
		return nil, fmt.Errorf("schema manifest %s: contracts must be sorted by id", path)
	}
	return &manifest, nil
}

// Load reads one schema file fail-closed.
func Load(path string) (*Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read schema: %w", err)
	}
	var schema Schema
	if err := strictDecode(data, &schema); err != nil {
		return nil, fmt.Errorf("schema %s: %w", path, err)
	}
	if err := schema.check("$"); err != nil {
		return nil, fmt.Errorf("schema %s: %w", path, err)
	}
	return &schema, nil
}

// check validates the schema definition itself.
func (s *Schema) check(at string) error {
	switch s.Type {
	case "object":
		if s.MapValues != nil {
			if len(s.Properties) > 0 || len(s.Required) > 0 {
				return fmt.Errorf("%s: mapValues excludes properties/required", at)
			}
			return s.MapValues.check(at + ".mapValues")
		}
		if s.AdditionalProperties == nil || *s.AdditionalProperties {
			return fmt.Errorf("%s: object schemas must set additionalProperties to false", at)
		}
		names := make([]string, 0, len(s.Properties))
		for name := range s.Properties {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			if err := s.Properties[name].check(at + "." + name); err != nil {
				return err
			}
		}
		for _, required := range s.Required {
			if _, ok := s.Properties[required]; !ok {
				return fmt.Errorf("%s: required property %q is not defined", at, required)
			}
		}
		return nil
	case "array":
		if s.Items == nil {
			return fmt.Errorf("%s: array schemas require items", at)
		}
		return s.Items.check(at + "[]")
	case "string", "integer", "number", "boolean":
		return nil
	}
	return fmt.Errorf("%s: unsupported type %q", at, s.Type)
}

// Validate checks one raw JSON document against the schema, rejecting
// trailing data and any field a closed object does not define.
func (s *Schema) Validate(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var document any
	if err := decoder.Decode(&document); err != nil {
		return fmt.Errorf("parse document: %w", err)
	}
	if decoder.More() {
		return fmt.Errorf("trailing data after JSON document")
	}
	return s.validateValue("$", document)
}

func (s *Schema) validateValue(at string, value any) error {
	switch s.Type {
	case "object":
		object, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("%s: expected object", at)
		}
		if s.MapValues != nil {
			keys := make([]string, 0, len(object))
			for key := range object {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				if err := s.MapValues.validateValue(at+"."+key, object[key]); err != nil {
					return err
				}
			}
			return nil
		}
		for _, required := range s.Required {
			if _, ok := object[required]; !ok {
				return fmt.Errorf("%s: missing required property %q", at, required)
			}
		}
		keys := make([]string, 0, len(object))
		for key := range object {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			property, ok := s.Properties[key]
			if !ok {
				return fmt.Errorf("%s: unknown property %q", at, key)
			}
			if err := property.validateValue(at+"."+key, object[key]); err != nil {
				return err
			}
		}
		return nil
	case "array":
		array, ok := value.([]any)
		if !ok {
			return fmt.Errorf("%s: expected array", at)
		}
		if s.MinItems != nil && len(array) < *s.MinItems {
			return fmt.Errorf("%s: fewer than %d items", at, *s.MinItems)
		}
		for i, element := range array {
			if err := s.Items.validateValue(fmt.Sprintf("%s[%d]", at, i), element); err != nil {
				return err
			}
		}
		return nil
	case "string":
		text, ok := value.(string)
		if !ok {
			return fmt.Errorf("%s: expected string", at)
		}
		if len(s.Enum) > 0 {
			for _, allowed := range s.Enum {
				if text == allowed {
					return nil
				}
			}
			return fmt.Errorf("%s: %q is not in the closed enum", at, text)
		}
		if s.Pattern != "" {
			if err := checkNamedPattern(s.Pattern, text); err != nil {
				return fmt.Errorf("%s: %w", at, err)
			}
		}
		return nil
	case "integer":
		number, ok := value.(json.Number)
		if !ok {
			return fmt.Errorf("%s: expected integer", at)
		}
		if strings.ContainsAny(number.String(), ".eE") {
			return fmt.Errorf("%s: %s is not an exact integer", at, number)
		}
		return nil
	case "number":
		if _, ok := value.(json.Number); !ok {
			return fmt.Errorf("%s: expected number", at)
		}
		return nil
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("%s: expected boolean", at)
		}
		return nil
	}
	return fmt.Errorf("%s: unsupported schema type %q", at, s.Type)
}

// checkNamedPattern validates reviewed named patterns; raw regular
// expressions are not a schema mechanism.
func checkNamedPattern(name, text string) error {
	switch name {
	case "sha256-hex":
		if len(text) != 64 {
			return fmt.Errorf("expected 64 lowercase hex characters")
		}
		for _, r := range text {
			if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
				return fmt.Errorf("expected lowercase hex, found %q", r)
			}
		}
		return nil
	case "nonempty":
		if text == "" {
			return fmt.Errorf("must not be empty")
		}
		return nil
	}
	return fmt.Errorf("unknown named pattern %q", name)
}

func strictDecode(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if decoder.More() {
		return fmt.Errorf("trailing content after JSON document")
	}
	return nil
}
