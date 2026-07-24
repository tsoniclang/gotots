package semantic

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestNormalizedSemanticShardRejectsDictionaryRangeAndPayloadMutations(
	t *testing.T,
) {
	tests := []struct {
		name   string
		mutate func(map[string]json.RawMessage) bool
		want   string
	}{
		{
			name: "zero-required-package-reference",
			mutate: func(shard map[string]json.RawMessage) bool {
				shard["package"] = json.RawMessage("0")
				return true
			},
			want: "disagrees with manifest",
		},
		{
			name:   "noncanonical-identity-table",
			mutate: reverseSemanticTypeIdentities,
			want:   "identity table types is not canonical",
		},
		{
			name:   "duplicate-identity-table-entry",
			mutate: duplicateSemanticTypeIdentity,
			want:   "identity table types is not canonical",
		},
		{
			name:   "unreferenced-identity-table-entry",
			mutate: appendUnreferencedSemanticTypeIdentity,
			want:   "wire types identity",
		},
		{
			name:   "shifted-relation-range",
			mutate: shiftCallableDeclarationRange,
			want:   "callable declarations range",
		},
		{
			name:   "payload-tag-mismatch",
			mutate: changeSignaturePayloadTag,
			want:   "payload tag",
		},
		{
			name:   "second-active-payload",
			mutate: addInactiveTypePayload,
			want:   "selects 2 payloads",
		},
		{
			name:   "omitted-active-payload",
			mutate: removeActiveTypePayload,
			want:   "selects 0 payloads",
		},
		{
			name:   "old-rendered-identity-field",
			mutate: addRenderedDefinitionIdentity,
			want:   "unknown field",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pkg := semanticWirePackage(t)
			encoded, entry := mutableNormalizedShard(t, pkg)
			if !test.mutate(encoded) {
				t.Fatal("mutation found no applicable semantic record")
			}
			mutated := marshalNormalizedShard(t, encoded)
			entry.ShardBytes = int64(len(mutated))
			_, err := decodeSemanticShard(
				bytes.NewReader(mutated),
				semanticFixture(t).authority,
				entry,
			)
			if err == nil ||
				!strings.Contains(err.Error(), test.want) {
				t.Fatalf(
					"mutation error = %v, want %q",
					err,
					test.want,
				)
			}
		})
	}
}

func mutableNormalizedShard(
	t *testing.T,
	pkg Package,
) (
	map[string]json.RawMessage,
	packageShardManifest,
) {
	t.Helper()
	var output bytes.Buffer
	if _, err := writeSemanticShard(&output, pkg); err != nil {
		t.Fatal(err)
	}
	var shard map[string]json.RawMessage
	if err := json.Unmarshal(output.Bytes(), &shard); err != nil {
		t.Fatal(err)
	}
	return shard, semanticWireManifestEntry(
		t,
		pkg,
		int64(output.Len()),
	)
}

func marshalNormalizedShard(
	t *testing.T,
	shard map[string]json.RawMessage,
) []byte {
	t.Helper()
	order := [...]string{
		"version",
		"provenance",
		"counts",
		"identities",
		"package",
		"definitions",
		"resolutions",
		"declarations",
		"bindings",
		"types",
		"operations",
		"unsupported",
	}
	var output bytes.Buffer
	output.WriteByte('{')
	for index, name := range order {
		value, present := shard[name]
		if !present {
			t.Fatalf("normalized semantic shard lacks %s", name)
		}
		if index != 0 {
			output.WriteByte(',')
		}
		encodedName, err := json.Marshal(name)
		if err != nil {
			t.Fatal(err)
		}
		output.Write(encodedName)
		output.WriteByte(':')
		output.Write(value)
	}
	for name := range shard {
		known := false
		for _, expected := range order {
			if name == expected {
				known = true
				break
			}
		}
		if known {
			continue
		}
		encodedName, err := json.Marshal(name)
		if err != nil {
			t.Fatal(err)
		}
		output.WriteByte(',')
		output.Write(encodedName)
		output.WriteByte(':')
		output.Write(shard[name])
	}
	output.WriteByte('}')
	return output.Bytes()
}
