package semantic

import (
	"bytes"
	"strings"
	"testing"
)

type binaryMutationLayout struct {
	version          int
	packageReference int
	definitionCount  int
	definitionForm   int
	bindingCount     int
	typeKind         int
}

func TestBinarySemanticShardRejectsStructuralMutations(t *testing.T) {
	pkg := semanticWirePackage(t)
	encoded, entry := binarySemanticFixture(t, pkg)
	layout := inspectBinaryMutationLayout(t, encoded, entry)
	tests := []struct {
		name   string
		mutate func([]byte) []byte
		want   string
	}{
		{
			name: "version",
			mutate: func(value []byte) []byte {
				value[layout.version] = ProviderArtifactVersion + 1
				return value
			},
			want: "version",
		},
		{
			name: "zero-package-reference",
			mutate: func(value []byte) []byte {
				value[layout.packageReference] = 0
				return value
			},
			want: "disagrees with manifest",
		},
		{
			name: "section-count",
			mutate: func(value []byte) []byte {
				value[layout.definitionCount] = 0
				return value
			},
			want: "count",
		},
		{
			name: "active-payload-form",
			mutate: func(value []byte) []byte {
				value[layout.definitionForm] = 0
				return value
			},
			want: "definition form",
		},
		{
			name: "relation-count",
			mutate: func(value []byte) []byte {
				value[layout.bindingCount] = 127
				return value
			},
		},
		{
			name: "type-kind",
			mutate: func(value []byte) []byte {
				value[layout.typeKind] = 0
				return value
			},
			want: "type kind",
		},
		{
			name: "dictionary-order",
			mutate: func(value []byte) []byte {
				return duplicateBinaryTypeIdentity(t, value, pkg)
			},
			want: "identity table is not canonical",
		},
		{
			name: "unreferenced-dictionary-entry",
			mutate: func(value []byte) []byte {
				return appendUnreferencedBinaryTypeIdentity(
					t, value, pkg,
				)
			},
			want: "type identity 3 is unreferenced",
		},
		{
			name: "truncation",
			mutate: func(value []byte) []byte {
				return value[:len(value)-1]
			},
		},
		{
			name: "trailing-byte",
			mutate: func(value []byte) []byte {
				return append(value, 0)
			},
			want: "trailing bytes",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mutated := test.mutate(append([]byte(nil), encoded...))
			mutatedEntry := entry
			mutatedEntry.ShardBytes = int64(len(mutated))
			_, err := decodeSemanticShard(
				bytes.NewReader(mutated),
				semanticFixture(t).authority,
				mutatedEntry,
			)
			if err == nil {
				t.Fatal("semantic binary mutation was accepted")
			}
			if test.want != "" &&
				!strings.Contains(err.Error(), test.want) {
				t.Fatalf(
					"mutation error = %v, want %q", err, test.want,
				)
			}
		})
	}
}

func binarySemanticFixture(
	t *testing.T,
	pkg Package,
) ([]byte, packageShardManifest) {
	t.Helper()
	var output bytes.Buffer
	if _, err := writeSemanticShard(&output, pkg); err != nil {
		t.Fatal(err)
	}
	return output.Bytes(), semanticWireManifestEntry(
		t,
		pkg,
		int64(output.Len()),
	)
}

func inspectBinaryMutationLayout(
	t *testing.T,
	encoded []byte,
	entry packageShardManifest,
) binaryMutationLayout {
	t.Helper()
	open := func() binarySemanticShard {
		shard, err := beginBinarySemanticShard(
			bytes.NewReader(encoded), entry,
		)
		if err != nil {
			t.Fatal(err)
		}
		return shard
	}
	header := open()
	layout := binaryMutationLayout{
		version:          len(semanticShardMagic),
		packageReference: int(header.decoder.consumed) - 1,
		definitionCount:  int(header.decoder.consumed),
	}
	if _, err := header.decoder.unsigned("definitions"); err != nil {
		t.Fatal(err)
	}
	if _, err := header.decoder.unsigned("definition id"); err != nil {
		t.Fatal(err)
	}
	if _, err := header.decoder.unsigned("definition package"); err != nil {
		t.Fatal(err)
	}
	layout.definitionForm = int(header.decoder.consumed)
	if _, err := header.decoder.unsigned("definition form"); err != nil {
		t.Fatal(err)
	}
	if _, err := header.decoder.text("definition name"); err != nil {
		t.Fatal(err)
	}
	layout.bindingCount = int(header.decoder.consumed)

	typeShard := open()
	if _, err := readBinaryDefinitions(
		typeShard.decoder, entry.DefinitionCount, 1,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := readBinaryResolutions(
		typeShard.decoder, entry.ResolutionCount,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := readBinaryDeclarations(
		typeShard.decoder, entry.DeclarationCount, 1,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := readBinaryBindings(
		typeShard.decoder, entry.BindingCount, 1,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := typeShard.decoder.unsigned("types"); err != nil {
		t.Fatal(err)
	}
	if _, err := typeShard.decoder.unsigned("type id"); err != nil {
		t.Fatal(err)
	}
	layout.typeKind = int(typeShard.decoder.consumed)
	return layout
}

func duplicateBinaryTypeIdentity(
	t *testing.T,
	encoded []byte,
	pkg Package,
) []byte {
	t.Helper()
	if len(pkg.identities.types) < 2 {
		t.Fatal("semantic fixture requires two type identities")
	}
	first := []byte(pkg.identities.types[0].digest)
	second := []byte(pkg.identities.types[1].digest)
	if len(first) != len(second) {
		t.Fatal("semantic type digest widths differ")
	}
	secondOffset := bytes.Index(encoded, second)
	if secondOffset < 0 {
		t.Fatal("second semantic type digest is absent")
	}
	copy(encoded[secondOffset:secondOffset+len(second)], first)
	return encoded
}

func appendUnreferencedBinaryTypeIdentity(
	t *testing.T,
	encoded []byte,
	pkg Package,
) []byte {
	t.Helper()
	types := pkg.identities.types
	if len(types) == 0 || len(types) >= 127 {
		t.Fatal("semantic fixture requires a one-byte type count")
	}
	first := []byte(types[0].digest)
	last := []byte(types[len(types)-1].digest)
	firstOffset := bytes.Index(encoded, first)
	lastOffset := bytes.Index(encoded, last)
	if firstOffset < 2 || lastOffset < firstOffset {
		t.Fatal("semantic type identity table is absent")
	}
	countOffset := firstOffset - 2
	if encoded[countOffset] != byte(len(types)) ||
		encoded[countOffset+1] != byte(len(first)) {
		t.Fatal("semantic type identity count layout changed")
	}
	extra := []byte(strings.Repeat("f", len(last)))
	if bytes.Compare(extra, last) <= 0 {
		t.Fatal("semantic fixture has no greater test digest")
	}
	insertAt := lastOffset + len(last)
	mutated := make([]byte, 0, len(encoded)+1+len(extra))
	mutated = append(mutated, encoded[:insertAt]...)
	mutated = append(mutated, byte(len(extra)))
	mutated = append(mutated, extra...)
	mutated = append(mutated, encoded[insertAt:]...)
	mutated[countOffset]++
	return mutated
}
