package maprepresentation_test

import (
	"sort"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const (
	parentNativeBoxMapRevision     = "97d6dc94"
	parentNativeBoxMapBytes        = 5730
	parentNativeBoxMapSyntaxNodes  = 772
	maximumNativeBoxMapBytes       = 3800
	maximumNativeBoxMapSyntaxNodes = 500
	maximumMapExpansionBytes       = 7000
	maximumMapExpansionSyntaxNodes = 900
)

type mapExpansionCost struct {
	path         string
	storage      string
	definitions  int
	bytes        int
	syntaxNodes  int
	keySurface   string
	valueSurface string
	source       string
}

func TestAggregateMapGeneratedCostInventory(t *testing.T) {
	emission := compileAggregateMapProfile(
		t,
		emit.IntegerRepresentationNumber,
	)
	artifacts := materialize(t, emission, t.TempDir())
	inventory := mapExpansionCosts(t, emission, artifacts)
	if len(inventory) != 6 {
		t.Fatalf("map expansion inventory = %d, want six reached classes", len(inventory))
	}
	for index, expansion := range inventory[:min(20, len(inventory))] {
		t.Logf(
			"map expansion rank=%d path=%s storage=%s definitions=%d bytes=%d syntax-nodes=%d key=%q value=%q",
			index+1,
			expansion.path,
			expansion.storage,
			expansion.definitions,
			expansion.bytes,
			expansion.syntaxNodes,
			expansion.keySurface,
			expansion.valueSurface,
		)
		if expansion.storage == "invalid" || expansion.definitions != 1 {
			t.Fatalf(
				"map expansion %s storage/definitions = %s/%d",
				expansion.path,
				expansion.storage,
				expansion.definitions,
			)
		}
		if expansion.bytes > maximumMapExpansionBytes ||
			expansion.syntaxNodes > maximumMapExpansionSyntaxNodes {
			t.Fatalf(
				"map expansion %s = %d bytes/%d syntax nodes, want at most %d/%d",
				expansion.path,
				expansion.bytes,
				expansion.syntaxNodes,
				maximumMapExpansionBytes,
				maximumMapExpansionSyntaxNodes,
			)
		}
	}
	assertNativeBoxMapCost(t, inventory)
}

func mapExpansionCosts(
	t *testing.T,
	emission emit.ProgramEmission,
	artifacts materialized,
) []mapExpansionCost {
	t.Helper()
	var result []mapExpansionCost
	for _, file := range emission.Files() {
		if !strings.HasPrefix(file.OutputPath(), "support/maps/") {
			continue
		}
		source := readFile(t, artifacts.file(t, file.OutputPath()))
		nodes, err := tsgo.EncodedSyntaxNodeCount(file.SourceFile())
		if err != nil {
			t.Fatal(err)
		}
		storage := "invalid"
		switch {
		case strings.Contains(source, "private readonly values: Map<"):
			storage = "native"
		case strings.Contains(source, "private readonly buckets: Map<number,"):
			storage = "hashed"
		}
		result = append(result, mapExpansionCost{
			path:        file.OutputPath(),
			storage:     storage,
			definitions: strings.Count(source, "export class $goMap_"),
			bytes:       len(source),
			syntaxNodes: nodes,
			keySurface:  mapExpansionKeySurface(source),
			valueSurface: mapExpansionLine(
				source,
				"private static $copyValue",
			),
			source: source,
		})
	}
	sort.Slice(result, func(left int, right int) bool {
		if result[left].bytes == result[right].bytes {
			return result[left].path < result[right].path
		}
		return result[left].bytes > result[right].bytes
	})
	return result
}

func mapExpansionKeySurface(source string) string {
	if line := mapExpansionLine(source, "private static $hash"); line != "" {
		return line
	}
	return mapExpansionLine(source, "private readonly values: Map<")
}

func mapExpansionLine(source string, match string) string {
	for _, line := range strings.Split(source, "\n") {
		if strings.Contains(line, match) {
			return strings.TrimSpace(line)
		}
	}
	return ""
}

func assertNativeBoxMapCost(t *testing.T, inventory []mapExpansionCost) {
	t.Helper()
	var representative *mapExpansionCost
	for index := range inventory {
		candidate := &inventory[index]
		if strings.Contains(candidate.source, "Map<int32, [") &&
			strings.Contains(candidate.source, "Box__from_aggregatemap") {
			if representative != nil {
				t.Fatal("native BoxMap has more than one expansion owner")
			}
			representative = candidate
		}
	}
	if representative == nil {
		t.Fatal("native BoxMap expansion is absent")
	}
	if representative.storage != "native" ||
		representative.definitions != 1 {
		t.Fatalf(
			"native BoxMap storage/definitions = %s/%d, want native/1",
			representative.storage,
			representative.definitions,
		)
	}
	if representative.bytes > maximumNativeBoxMapBytes ||
		representative.syntaxNodes > maximumNativeBoxMapSyntaxNodes {
		t.Fatalf(
			"native BoxMap = %d bytes/%d syntax nodes, one constructor plus 12 methods must stay within %d/%d",
			representative.bytes,
			representative.syntaxNodes,
			maximumNativeBoxMapBytes,
			maximumNativeBoxMapSyntaxNodes,
		)
	}
	if representative.bytes >= parentNativeBoxMapBytes ||
		representative.syntaxNodes >= parentNativeBoxMapSyntaxNodes {
		t.Fatalf(
			"native BoxMap did not improve on parent %s at %d bytes/%d syntax nodes: current=%d/%d",
			parentNativeBoxMapRevision,
			parentNativeBoxMapBytes,
			parentNativeBoxMapSyntaxNodes,
			representative.bytes,
			representative.syntaxNodes,
		)
	}
	t.Logf(
		"native BoxMap parent=%s:%d bytes/%d syntax-nodes current=%d/%d delta=%d/%d",
		parentNativeBoxMapRevision,
		parentNativeBoxMapBytes,
		parentNativeBoxMapSyntaxNodes,
		representative.bytes,
		representative.syntaxNodes,
		representative.bytes-parentNativeBoxMapBytes,
		representative.syntaxNodes-parentNativeBoxMapSyntaxNodes,
	)
}
