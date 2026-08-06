package runtimecontract

import (
	"errors"
	"strings"
	"testing"
)

const validRuntimeRequirements = `{
  "schemaVersion": 2,
  "integerRepresentations": ["number", "fixed64-bigint", "bigint"],
  "providerIntegerRepresentation": "bigint",
  "providerScalarModule": "./internal/scalars.js",
  "nativeIntegerBits": 64,
  "primitiveAliases": [{"id": 4, "export": "int32", "providerCarrier": "number"}],
  "runtimeSymbols": [{"id": 300, "export": "RuntimeSlice"}]
}`

func TestDecodeRetainsImmutableRuntimeRequirements(t *testing.T) {
	requirements, err := Decode([]byte(validRuntimeRequirements))
	if err != nil {
		t.Fatal(err)
	}
	if !requirements.Valid() ||
		!requirements.AllowsProfile(ProfileNumber) ||
		!requirements.AllowsProfile(ProfileFixed64BigInt) ||
		!requirements.AllowsProfile(ProfileBigInt) ||
		requirements.ProviderProfile() != ProfileBigInt ||
		requirements.ProviderScalarModule() != "./internal/scalars.js" ||
		requirements.NativeIntegerBits() != 64 {
		t.Fatal("runtime profile admission is invalid")
	}
	aliases := requirements.PrimitiveAliases()
	symbols := requirements.RuntimeSymbols()
	if len(aliases) != 1 || aliases[0].ID() != 4 ||
		aliases[0].Export() != "int32" ||
		aliases[0].ProviderCarrier() != PrimitiveCarrierNumber {
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
		"old schema": strings.Replace(
			validRuntimeRequirements, `"schemaVersion": 2`, `"schemaVersion": 1`, 1,
		),
		"missing provider profile": strings.Replace(
			validRuntimeRequirements,
			`  "providerIntegerRepresentation": "bigint",`+"\n",
			"",
			1,
		),
		"provider profile not admitted": strings.Replace(
			validRuntimeRequirements,
			`["number", "fixed64-bigint", "bigint"]`,
			`["number"]`,
			1,
		),
		"missing provider scalar module": strings.Replace(
			validRuntimeRequirements,
			`  "providerScalarModule": "./internal/scalars.js",`+"\n",
			"",
			1,
		),
		"invalid provider scalar module": strings.Replace(
			validRuntimeRequirements,
			`"./internal/scalars.js"`,
			`"../foreign.js"`,
			1,
		),
		"invalid native width": strings.Replace(
			validRuntimeRequirements, `"nativeIntegerBits": 64`, `"nativeIntegerBits": 48`, 1,
		),
		"missing provider carrier": strings.Replace(
			validRuntimeRequirements, `, "providerCarrier": "number"`, ``, 1,
		),
		"unknown field": strings.Replace(
			validRuntimeRequirements,
			`  "runtimeSymbols"`,
			`  "extra": true,`+"\n"+`  "runtimeSymbols"`,
			1,
		),
		"second document": validRuntimeRequirements + ` {}`,
		"unknown profile": strings.Replace(
			validRuntimeRequirements, `"number", "fixed64-bigint"`, `"number", "wide"`, 1,
		),
		"duplicate profile": strings.Replace(
			validRuntimeRequirements, `"number", "fixed64-bigint"`, `"number", "number", "fixed64-bigint"`, 1,
		),
		"duplicate alias id": strings.Replace(
			validRuntimeRequirements,
			`[{"id": 4, "export": "int32", "providerCarrier": "number"}]`,
			`[{"id": 4, "export": "int32", "providerCarrier": "number"}, {"id": 4, "export": "other", "providerCarrier": "number"}]`,
			1,
		),
		"duplicate symbol export": strings.Replace(
			validRuntimeRequirements,
			`[{"id": 300, "export": "RuntimeSlice"}]`,
			`[{"id": 300, "export": "RuntimeSlice"}, {"id": 301, "export": "RuntimeSlice"}]`,
			1,
		),
		"empty export": strings.Replace(
			validRuntimeRequirements, `"export": "int32"`, `"export": ""`, 1,
		),
		"empty surface": strings.NewReplacer(
			`[{"id": 4, "export": "int32", "providerCarrier": "number"}]`, `[]`,
			`[{"id": 300, "export": "RuntimeSlice"}]`, `[]`,
		).Replace(validRuntimeRequirements),
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
