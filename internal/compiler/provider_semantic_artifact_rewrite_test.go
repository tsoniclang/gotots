package compiler

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func rewriteSemanticArtifact(
	t *testing.T,
	path string,
	mutate semanticShardMutation,
) (string, string, string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) < 16 {
		t.Fatal("semantic artifact is shorter than its header")
	}
	manifestBytes := int(binary.BigEndian.Uint64(raw[8:16]))
	shardBase := 16 + manifestBytes
	if manifestBytes <= 0 || shardBase > len(raw) {
		t.Fatal("semantic artifact has an invalid manifest length")
	}
	var manifest mutableSemanticManifest
	if err := json.Unmarshal(raw[16:shardBase], &manifest); err != nil {
		t.Fatal(err)
	}
	shards := make([][]byte, len(manifest.Packages))
	mutated := false
	mutatedPackage := ""
	var nextOffset int64
	for index := range manifest.Packages {
		entry := &manifest.Packages[index]
		start := shardBase + int(entry.ShardOffset)
		end := start + int(entry.ShardBytes)
		if start < shardBase || end > len(raw) || start >= end {
			t.Fatal("semantic shard extent is invalid")
		}
		encoded := append([]byte(nil), raw[start:end]...)
		if !mutated {
			var shard map[string]json.RawMessage
			if err := json.Unmarshal(encoded, &shard); err != nil {
				t.Fatal(err)
			}
			if mutate(shard, entry) {
				encoded, err = marshalMutableSemanticShard(shard)
				if err != nil {
					t.Fatal(err)
				}
				mutated = true
				mutatedPackage = entry.Package
			}
		}
		digest := sha256.Sum256(encoded)
		entry.ShardOffset = nextOffset
		entry.ShardBytes = int64(len(encoded))
		entry.ShardDigest = hex.EncodeToString(digest[:])
		nextOffset += entry.ShardBytes
		shards[index] = encoded
	}
	if !mutated {
		t.Fatal("semantic artifact had no shard accepted by the mutation")
	}
	encodedManifest, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	rewritten := make(
		[]byte, 0, 16+len(encodedManifest)+int(nextOffset),
	)
	rewritten = append(rewritten, raw[:8]...)
	var length [8]byte
	binary.BigEndian.PutUint64(
		length[:], uint64(len(encodedManifest)),
	)
	rewritten = append(rewritten, length[:]...)
	rewritten = append(rewritten, encodedManifest...)
	for _, shard := range shards {
		rewritten = append(rewritten, shard...)
	}
	rewrittenPath := filepath.Join(
		t.TempDir(), "resealed-provider.semantic.gotots",
	)
	if err := os.WriteFile(rewrittenPath, rewritten, 0o600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(rewritten)
	return rewrittenPath, hex.EncodeToString(digest[:]), mutatedPackage
}

func marshalMutableSemanticShard(
	shard map[string]json.RawMessage,
) ([]byte, error) {
	order := []string{
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
	known := map[string]bool{}
	var output bytes.Buffer
	output.WriteByte('{')
	first := true
	writeField := func(name string) error {
		value, present := shard[name]
		if !present {
			return fmt.Errorf(
				"semantic mutation shard lacks field %s",
				name,
			)
		}
		if !first {
			output.WriteByte(',')
		}
		first = false
		encodedName, err := json.Marshal(name)
		if err != nil {
			return err
		}
		output.Write(encodedName)
		output.WriteByte(':')
		output.Write(value)
		known[name] = true
		return nil
	}
	for _, name := range order {
		if err := writeField(name); err != nil {
			return nil, err
		}
	}
	var extra []string
	for name := range shard {
		if !known[name] {
			extra = append(extra, name)
		}
	}
	sort.Strings(extra)
	for _, name := range extra {
		if err := writeField(name); err != nil {
			return nil, err
		}
	}
	output.WriteByte('}')
	return output.Bytes(), nil
}
