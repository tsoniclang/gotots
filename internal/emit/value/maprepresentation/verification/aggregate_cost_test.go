package maprepresentation_test

import (
	"sort"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/output"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const (
	parentNativeBoxMapRevision     = "97d6dc94"
	parentNativeBoxMapBytes        = 5730
	parentNativeBoxMapSyntaxNodes  = 772
	maximumNativeBoxMapBytes       = 3800
	maximumNativeBoxMapSyntaxNodes = 500
	maximumMapExpansionBytes       = 7200
	maximumMapExpansionSyntaxNodes = 900
)

type mapExpansionCost struct {
	path         string
	storage      string
	definitions  int
	actualBytes  int
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
	if len(inventory) != 7 {
		t.Fatalf("map expansion inventory = %d, want seven reached classes", len(inventory))
	}
	for index, expansion := range inventory[:min(20, len(inventory))] {
		t.Logf(
			"map expansion rank=%d path=%s storage=%s definitions=%d payload-bytes=%d actual-bytes=%d syntax-nodes=%d key=%q value=%q",
			index+1,
			expansion.path,
			expansion.storage,
			expansion.definitions,
			expansion.bytes,
			expansion.actualBytes,
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
		if file.OutputPath() != output.MapSpecializationSupportPath {
			continue
		}
		source := readFile(t, artifacts.file(t, file.OutputPath()))
		nodesByName := make(map[string]int)
		for _, statement := range file.SourceFile().Statements() {
			declaration, ok := statement.(tsgo.ClassDeclaration)
			if !ok || !strings.HasPrefix(declaration.Name().Text(), "$goMap$") {
				continue
			}
			nodes, err := tsgo.EncodedSyntaxNodeCount(declaration)
			if err != nil {
				t.Fatal(err)
			}
			nodesByName[declaration.Name().Text()] = nodes
		}
		for name, classSource := range mapClassSources(source) {
			storage := "invalid"
			switch {
			case strings.Contains(classSource, "private readonly values: Map<"):
				storage = "native"
			case strings.Contains(classSource, "private readonly buckets: Map<number,"):
				storage = "hashed"
			}
			result = append(result, mapExpansionCost{
				path:        file.OutputPath() + "#" + name,
				storage:     storage,
				definitions: 1,
				actualBytes: len(classSource),
				bytes:       mapClassPayloadBytes(name, classSource),
				syntaxNodes: nodesByName[name],
				keySurface:  mapExpansionKeySurface(classSource),
				valueSurface: mapExpansionLine(
					classSource,
					"private static $copyValue",
				),
				source: classSource,
			})
		}
	}
	sort.Slice(result, func(left int, right int) bool {
		if result[left].bytes == result[right].bytes {
			return result[left].path < result[right].path
		}
		return result[left].bytes > result[right].bytes
	})
	return result
}

func mapClassPayloadBytes(name string, source string) int {
	const localFamilyName = "GoMap"
	if len(name) <= len(localFamilyName) {
		return len(source)
	}
	return len(source) -
		strings.Count(source, name)*(len(name)-len(localFamilyName))
}

func mapClassSources(source string) map[string]string {
	const prefix = "export class $goMap$"
	result := make(map[string]string)
	for start := strings.Index(source, prefix); start >= 0; {
		remainder := source[start:]
		end := len(remainder)
		if next := strings.Index(remainder[len(prefix):], prefix); next >= 0 {
			end = len(prefix) + next
		}
		classSource := remainder[:end]
		if annotation := strings.Index(classSource, "\nattribute<"); annotation >= 0 {
			classSource = classSource[:annotation+1]
		}
		nameStart := len("export class ")
		nameEnd := strings.IndexAny(classSource[nameStart:], " {<\n")
		if nameEnd < 0 {
			break
		}
		result[classSource[nameStart:nameStart+nameEnd]] = classSource
		start += end
		if start >= len(source) {
			break
		}
		next := strings.Index(source[start:], prefix)
		if next < 0 {
			break
		}
		start += next
	}
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
		if strings.Contains(candidate.source, "Map<int32, Box__from_aggregatemap>") &&
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
