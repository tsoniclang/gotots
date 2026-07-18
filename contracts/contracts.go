// Package contracts embeds the reviewed semantic-class support registry
// (contracts/support-classes.json). The registry is joined to every
// generated operation census: a class absent from it is unimplemented,
// never implicitly supported, and a generated body using an
// unregistered class fails the run — support decisions are explicit,
// reviewed, and committed.
package contracts

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
)

//go:embed support-classes.json
var supportClassesJSON []byte

// SupportClass is one reviewed operation-class decision.
type SupportClass struct {
	Key        string `json:"key"`
	State      string `json:"state"`
	SpecClause string `json:"specClause"`
}

// Registry is the parsed support-class registry.
type Registry struct {
	SchemaVersion int            `json:"schemaVersion"`
	Classes       []SupportClass `json:"classes"`

	byKey map[string]SupportClass
}

// LoadRegistry parses the embedded registry fail-closed.
func LoadRegistry() (*Registry, error) {
	var registry Registry
	decoder := json.NewDecoder(bytes.NewReader(supportClassesJSON))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&registry); err != nil {
		return nil, fmt.Errorf("parse support-class registry: %w", err)
	}
	if decoder.More() {
		return nil, fmt.Errorf("support-class registry: trailing content")
	}
	if registry.SchemaVersion != 1 {
		return nil, fmt.Errorf("support-class registry: unsupported schemaVersion %d", registry.SchemaVersion)
	}
	registry.byKey = map[string]SupportClass{}
	keys := make([]string, 0, len(registry.Classes))
	for _, class := range registry.Classes {
		if class.Key == "" || class.State != "generated" || class.SpecClause == "" {
			return nil, fmt.Errorf("support-class registry: class %+v is incomplete", class)
		}
		if _, duplicate := registry.byKey[class.Key]; duplicate {
			return nil, fmt.Errorf("support-class registry: duplicate class %q", class.Key)
		}
		registry.byKey[class.Key] = class
		keys = append(keys, class.Key)
	}
	if !sort.StringsAreSorted(keys) {
		return nil, fmt.Errorf("support-class registry: classes must be sorted by key")
	}
	return &registry, nil
}

// Keys returns every reviewed class key in sorted order.
func (r *Registry) Keys() []string {
	out := make([]string, 0, len(r.byKey))
	for key := range r.byKey {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

// Generated reports whether the operation class has a reviewed
// generated-support decision. Absence is unimplemented, never implicit
// support.
func (r *Registry) Generated(key string) bool {
	_, ok := r.byKey[key]
	return ok
}
