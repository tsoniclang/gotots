package runtimecontract

import (
	"errors"
	"testing"
)

func TestDecodeRetainsImmutableRuntimeRequirements(t *testing.T) {
	requirements, err := Decode([]byte(`{
  "schemaVersion": 1,
  "integerRepresentations": ["number"],
  "primitiveAliases": [{"id": 4, "export": "int32"}],
  "runtimeSymbols": [{"id": 300, "export": "RuntimeSlice"}]
}`))
	if err != nil {
		t.Fatal(err)
	}
	if !requirements.Valid() ||
		!requirements.AllowsProfile(ProfileNumber) ||
		requirements.AllowsProfile(ProfileBigInt) {
		t.Fatal("runtime profile admission is invalid")
	}
	aliases := requirements.PrimitiveAliases()
	symbols := requirements.RuntimeSymbols()
	if len(aliases) != 1 || aliases[0].ID() != 4 ||
		aliases[0].Export() != "int32" {
		t.Fatalf("primitive aliases = %v", aliases)
	}
	if len(symbols) != 1 || symbols[0].ID() != 300 ||
		symbols[0].Export() != "RuntimeSlice" {
		t.Fatalf("runtime symbols = %v", symbols)
	}
	aliases[0] = Entry{}
	symbols[0] = Entry{}
	if requirements.PrimitiveAliases()[0].ID() != 4 ||
		requirements.RuntimeSymbols()[0].ID() != 300 {
		t.Fatal("runtime requirements expose mutable backing storage")
	}
}

func TestDecodeRejectsRuntimeRequirementMutations(t *testing.T) {
	tests := map[string]string{
		"unknown field": `{
  "schemaVersion": 1,
  "integerRepresentations": ["number"],
  "primitiveAliases": [{"id": 4, "export": "int32"}],
  "runtimeSymbols": [{"id": 300, "export": "RuntimeSlice"}],
  "extra": true
}`,
		"second document": `{
  "schemaVersion": 1,
  "integerRepresentations": ["number"],
  "primitiveAliases": [{"id": 4, "export": "int32"}],
  "runtimeSymbols": [{"id": 300, "export": "RuntimeSlice"}]
} {}`,
		"unknown profile": `{
  "schemaVersion": 1,
  "integerRepresentations": ["wide"],
  "primitiveAliases": [{"id": 4, "export": "int32"}],
  "runtimeSymbols": [{"id": 300, "export": "RuntimeSlice"}]
}`,
		"duplicate profile": `{
  "schemaVersion": 1,
  "integerRepresentations": ["number", "number"],
  "primitiveAliases": [{"id": 4, "export": "int32"}],
  "runtimeSymbols": [{"id": 300, "export": "RuntimeSlice"}]
}`,
		"duplicate alias id": `{
  "schemaVersion": 1,
  "integerRepresentations": ["number"],
  "primitiveAliases": [
    {"id": 4, "export": "int32"},
    {"id": 4, "export": "other"}
  ],
  "runtimeSymbols": [{"id": 300, "export": "RuntimeSlice"}]
}`,
		"duplicate symbol export": `{
  "schemaVersion": 1,
  "integerRepresentations": ["number"],
  "primitiveAliases": [{"id": 4, "export": "int32"}],
  "runtimeSymbols": [
    {"id": 300, "export": "RuntimeSlice"},
    {"id": 301, "export": "RuntimeSlice"}
  ]
}`,
		"empty export": `{
  "schemaVersion": 1,
  "integerRepresentations": ["number"],
  "primitiveAliases": [{"id": 4, "export": ""}],
  "runtimeSymbols": [{"id": 300, "export": "RuntimeSlice"}]
}`,
		"empty surface": `{
  "schemaVersion": 1,
  "integerRepresentations": ["number"],
  "primitiveAliases": [],
  "runtimeSymbols": []
}`,
	}
	for name, source := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := Decode([]byte(source))
			var contractError *Error
			if !errors.As(err, &contractError) {
				t.Fatalf("error = %T %v", err, err)
			}
		})
	}
}
