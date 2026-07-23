package compiler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/scope/contract"
)

type stage1DefinitionRecord struct {
	definition structure.ImplementationDefinition
	site       structure.DefinitionSite
	header     structure.HeaderRegion
	boundary   structure.ExecutionBoundary
}

func collectStage1Definitions(
	t *testing.T,
	inspection *Inspection,
) map[identity.DefinitionID]stage1DefinitionRecord {
	t.Helper()
	records := map[identity.DefinitionID]stage1DefinitionRecord{}
	if err := inspection.Structure().VisitPackages(func(
		pkg structure.PackageGraph,
	) error {
		for _, definition := range pkg.Definitions() {
			record := records[definition.ID()]
			record.definition = definition
			records[definition.ID()] = record
		}
		for _, site := range pkg.Sites() {
			record := records[site.Definition()]
			record.site = site
			records[site.Definition()] = record
		}
		for _, header := range pkg.Headers() {
			definition := header.ID().Definition()
			record := records[definition]
			record.header = header
			records[definition] = record
		}
		for _, boundary := range pkg.Boundaries() {
			definition := boundary.ID().Definition()
			record := records[definition]
			record.boundary = boundary
			records[definition] = record
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for definition, record := range records {
		if record.definition.ID() != definition ||
			record.site.Definition() != definition ||
			record.header.ID().Definition() != definition ||
			record.boundary.ID().Definition() != definition {
			t.Fatalf("definition %s lacks its exact site/header/boundary", definition)
		}
	}
	return records
}

func findDefinitionsByName(
	records map[identity.DefinitionID]stage1DefinitionRecord,
	name string,
) []identity.DefinitionID {
	var out []identity.DefinitionID
	for definition, record := range records {
		if record.definition.Name() == name {
			out = append(out, definition)
		}
	}
	return out
}

type contractArtifact struct {
	ID      string                 `json:"id"`
	Version int                    `json:"version"`
	Rules   []contractArtifactRule `json:"rules"`
}

type contractArtifactRule struct {
	Bind       string `json:"bind"`
	Definition string `json:"definition,omitempty"`
	Package    string `json:"package,omitempty"`
	Namespace  string `json:"namespace,omitempty"`
	Condition  string `json:"condition"`
	Fact       string `json:"fact,omitempty"`
	Provider   string `json:"provider"`
}

func writeDepthContract(
	t *testing.T,
	id string,
	fullDefinitions ...identity.DefinitionID,
) string {
	t.Helper()
	artifact := contractArtifact{
		ID: id, Version: contract.SchemaVersion,
		Rules: []contractArtifactRule{{
			Bind: "namespace", Namespace: identity.OwnerModule.String(),
			Condition: "always", Provider: "gostdlib",
		}},
	}
	for _, definition := range fullDefinitions {
		artifact.Rules = append(artifact.Rules, contractArtifactRule{
			Bind: "definition", Definition: definition.String(),
			Condition: "always", Provider: "automatic-translation",
		})
	}
	raw, err := json.Marshal(artifact)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "contract.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeContractArtifact(
	t *testing.T,
	selected contract.Contract,
) string {
	t.Helper()
	artifact := contractArtifact{
		ID: selected.ID(), Version: contract.SchemaVersion,
	}
	for _, rule := range selected.Rules() {
		encoded := contractArtifactRule{
			Condition: rule.Condition().String(),
			Provider:  rule.Provider().String(),
		}
		if rule.FactKind().Valid() {
			encoded.Fact = rule.FactKind().String()
		}
		switch rule.Selector() {
		case contract.SelectorExactDefinition:
			encoded.Bind = "definition"
			encoded.Definition = rule.Definition().String()
		case contract.SelectorExactPackage:
			encoded.Bind = "package"
			encoded.Package = rule.Package().String()
		case contract.SelectorNamespace:
			encoded.Bind = "namespace"
			encoded.Namespace = rule.Namespace().String()
		default:
			t.Fatalf("contract contains invalid selector %s", rule.Selector())
		}
		artifact.Rules = append(artifact.Rules, encoded)
	}
	raw, err := json.Marshal(artifact)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "contract.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
