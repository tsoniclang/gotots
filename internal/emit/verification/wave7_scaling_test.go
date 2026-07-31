package emit_test

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestWaveSevenGenericBodiesAndCapabilitiesScaleByExactContract(
	t *testing.T,
) {
	counts := []int{1, 2, 4}
	sourceBytes := make([]int, len(counts))
	targetBytes := make([]int, len(counts))
	targetNodes := make([]int, len(counts))
	var genericBody []byte
	var genericBodyText string
	var genericBodyNodes int
	client, err := tsgo.StartClient(repositoryRoot(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close TS-Go client: %v", err)
		}
	})

	for index, count := range counts {
		measurement := measureWaveSevenGenericScale(t, client, count)
		sourceBytes[index] = measurement.sourceBytes
		targetBytes[index] = measurement.targetBytes
		targetNodes[index] = measurement.targetNodes
		if measurement.genericBodies != 1 {
			t.Fatalf(
				"%dx generic bodies = %d, want one",
				count,
				measurement.genericBodies,
			)
		}
		if measurement.capabilities != count {
			t.Fatalf(
				"%dx capability definitions = %d, want %d distinct signatures; functions=%v",
				count,
				measurement.capabilities,
				count,
				measurement.functionNames,
			)
		}
		if index == 0 {
			genericBody = measurement.genericBody
			genericBodyText = measurement.genericBodyText
			genericBodyNodes = measurement.genericBodyNodes
			continue
		}
		if !bytes.Equal(measurement.genericBody, genericBody) ||
			measurement.genericBodyText != genericBodyText ||
			measurement.genericBodyNodes != genericBodyNodes {
			t.Fatalf(
				"generic body changed at %dx instantiations: encoded=%d/%d printed=%d/%d nodes=%d/%d",
				count,
				len(genericBody),
				len(measurement.genericBody),
				len(genericBodyText),
				len(measurement.genericBodyText),
				genericBodyNodes,
				measurement.genericBodyNodes,
			)
		}
	}
	assertWaveFourLinearDoubling(t, "generic source bytes", sourceBytes)
	assertWaveFourLinearDoubling(t, "generic target bytes", targetBytes)
	assertWaveFourLinearDoubling(t, "generic target AST nodes", targetNodes)
	t.Logf(
		"generic scaling instantiations=%v source=%v target=%v nodes=%v body-bytes=%d body-nodes=%d capabilities=%v",
		counts,
		sourceBytes,
		targetBytes,
		targetNodes,
		len(genericBodyText),
		genericBodyNodes,
		counts,
	)
}

type waveSevenGenericScale struct {
	sourceBytes      int
	targetBytes      int
	targetNodes      int
	genericBodies    int
	capabilities     int
	genericBody      []byte
	genericBodyText  string
	genericBodyNodes int
	functionNames    []string
}

func measureWaveSevenGenericScale(
	t *testing.T,
	client *tsgo.Client,
	count int,
) waveSevenGenericScale {
	t.Helper()
	directory := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(directory, "go.mod"),
		"module example.com/wave7scaling\n\ngo 1.26.4\n",
	)
	source := waveSevenGenericScalingSource(count)
	writeProgramFile(
		t,
		filepath.Join(directory, "source.go"),
		source,
	)
	program, err := load.Load(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, roots)
	if err != nil {
		t.Fatal(err)
	}
	measurement := waveSevenGenericScale{sourceBytes: len(source)}
	for _, file := range emission.Files() {
		printed, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := tsgo.EncodeSourceFile(file.SourceFile())
		if err != nil {
			t.Fatal(err)
		}
		measurement.targetBytes += len(printed)
		measurement.targetNodes += waveFourEncodedNodes(t, encoded)
		for _, statement := range file.SourceFile().Statements() {
			function, ok := statement.(tsgo.FunctionDeclaration)
			if !ok {
				continue
			}
			measurement.functionNames = append(
				measurement.functionNames,
				function.Name().Text(),
			)
			switch {
			case function.Name().Text() == "Add":
				measurement.genericBodies++
				measurement.genericBody, err = tsgo.EncodeNode(function)
				if err != nil {
					t.Fatal(err)
				}
				measurement.genericBodyText, err = client.PrintNode(
					function,
					tsgo.PrintOptions{},
				)
				if err != nil {
					t.Fatal(err)
				}
				measurement.genericBodyNodes = waveFourEncodedNodes(
					t,
					measurement.genericBody,
				)
			case strings.HasPrefix(
				function.Name().Text(),
				"$goCapability_",
			):
				measurement.capabilities++
			}
		}
	}
	return measurement
}

func waveSevenGenericScalingSource(count int) string {
	types := []string{"int8", "int16", "int32", "int64"}
	var source strings.Builder
	source.WriteString(`package wave7scaling

type Number interface {
	~int8 | ~int16 | ~int32 | ~int64
}

func Add[T Number](left, right T) T {
	return left + right
}

func Audit() int64 {
	var result int64
`)
	for index := range count {
		sourceType := types[index]
		for repetition := range 2 {
			fmt.Fprintf(
				&source,
				"\tresult += int64(Add(%s(%d), %s(%d)))\n",
				sourceType,
				index+repetition+1,
				sourceType,
				index+repetition+2,
			)
		}
	}
	source.WriteString("\treturn result\n}\n")
	return source.String()
}

func TestWaveSevenRecursiveGenericContractsConverge(t *testing.T) {
	first := waveSevenRecursiveGenericArtifacts(t)
	second := waveSevenRecursiveGenericArtifacts(t)
	if len(first) != 3 || len(second) != 3 {
		t.Fatalf(
			"recursive generic declarations = %d/%d, want 3/3",
			len(first),
			len(second),
		)
	}
	for name, firstBody := range first {
		secondBody, ok := second[name]
		if !ok {
			t.Fatalf("second fixed point lacks %s", name)
		}
		if !bytes.Equal(firstBody, secondBody) {
			t.Fatalf("%s changed across identical fixed points", name)
		}
	}
}

func waveSevenRecursiveGenericArtifacts(t *testing.T) map[string][]byte {
	t.Helper()
	program, err := load.Load(context.Background(), load.Request{
		Directory: waveSevenGenericDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	root, err := emit.NewRoot(
		program.Roots()[0].Types().Scope().Lookup("Audit"),
	)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, []emit.Root{root})
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]bool{
		"RecursiveAdd": false,
		"MutualAddA":   false,
		"MutualAddB":   false,
	}
	result := make(map[string][]byte, len(expected))
	for _, file := range emission.Files() {
		for _, statement := range file.SourceFile().Statements() {
			function, ok := statement.(tsgo.FunctionDeclaration)
			if !ok {
				continue
			}
			name := function.Name().Text()
			if _, selected := expected[name]; !selected {
				continue
			}
			if expected[name] {
				t.Fatalf("recursive generic body %s is duplicated", name)
			}
			expected[name] = true
			if len(function.TypeParameters()) != 1 ||
				len(function.Parameters()) != 5 {
				t.Fatalf(
					"%s shape = %d type parameters/%d parameters, want 1/5",
					name,
					len(function.TypeParameters()),
					len(function.Parameters()),
				)
			}
			operations := map[string]int{
				"$go$binary_add_": 0,
				"$go$copy_":       0,
			}
			for index, parameter := range function.Parameters() {
				identifier, ok := parameter.Name().(tsgo.Identifier)
				if !ok {
					t.Fatalf("%s parameter %d is not an identifier", name, index)
				}
				for prefix := range operations {
					if strings.HasPrefix(identifier.Text(), prefix) {
						operations[prefix]++
						if index >= 2 {
							t.Fatalf(
								"%s capability %s follows source parameters",
								name,
								identifier.Text(),
							)
						}
					}
				}
			}
			for prefix, count := range operations {
				if count != 1 {
					t.Fatalf(
						"%s capabilities %s = %d, want one",
						name,
						prefix,
						count,
					)
				}
			}
			encoded, err := tsgo.EncodeNode(function)
			if err != nil {
				t.Fatal(err)
			}
			result[name] = encoded
		}
	}
	for name, found := range expected {
		if !found {
			t.Fatalf("recursive generic body %s is absent", name)
		}
	}
	return result
}

func TestWaveSevenGeneratedTailIsEncodedAndBounded(t *testing.T) {
	program, err := load.Load(context.Background(), load.Request{
		Directory: waveSevenGenericDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	scope := program.Roots()[0].Types().Scope()
	rootNames := []string{
		"Audit",
		"AuditBigIntOperations",
		"AuditFunctions",
		"AuditGenericMethodAdapters",
		"AuditIteratorRanges",
	}
	roots := make([]emit.Root, 0, len(rootNames))
	for _, name := range rootNames {
		root, err := emit.NewRoot(scope.Lookup(name))
		if err != nil {
			t.Fatal(err)
		}
		roots = append(roots, root)
	}
	emission, err := emit.CompileWithOptions(
		program,
		roots,
		emit.Options{
			IntegerRepresentation: emit.IntegerRepresentationBigInt,
			EvaluationOrder:       emit.EvaluationOrderPreserveGo,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	client, err := tsgo.StartClient(repositoryRoot(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close TS-Go client: %v", err)
		}
	})

	var artifacts []waveSevenTailArtifact
	identities := make(map[string]struct{})
	counts := make(map[string]int)
	for _, file := range emission.Files() {
		for _, statement := range file.SourceFile().Statements() {
			name, kind, selected := waveSevenTailIdentity(statement)
			if !selected {
				continue
			}
			identity := file.OutputPath() + "|" + kind + "|" + name
			if _, duplicate := identities[identity]; duplicate {
				t.Fatalf("generic artifact %s is duplicated", identity)
			}
			identities[identity] = struct{}{}
			printed, err := client.PrintNode(statement, tsgo.PrintOptions{})
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := tsgo.EncodeNode(statement)
			if err != nil {
				t.Fatal(err)
			}
			artifacts = append(artifacts, waveSevenTailArtifact{
				path:  file.OutputPath(),
				name:  name,
				kind:  kind,
				bytes: len(printed),
				nodes: waveFourEncodedNodes(t, encoded),
			})
			counts[kind]++
		}
	}
	if len(artifacts) < 20 {
		t.Fatalf(
			"generic tail has %d inspected artifacts, want at least 20",
			len(artifacts),
		)
	}
	for kind, minimum := range map[string]int{
		"capability":       5,
		"generic-alias":    1,
		"generic-class":    5,
		"generic-function": 20,
	} {
		if counts[kind] < minimum {
			t.Fatalf(
				"%s artifacts = %d, want at least %d",
				kind,
				counts[kind],
				minimum,
			)
		}
	}
	for _, artifact := range artifacts {
		maximum := waveSevenTailBounds[artifact.kind]
		if artifact.bytes > maximum.bytes ||
			artifact.nodes > maximum.nodes {
			t.Fatalf(
				"%s %s:%s tail = %d bytes/%d nodes, bound %d/%d",
				artifact.kind,
				artifact.path,
				artifact.name,
				artifact.bytes,
				artifact.nodes,
				maximum.bytes,
				maximum.nodes,
			)
		}
	}
	sort.Slice(artifacts, func(left, right int) bool {
		if artifacts[left].bytes != artifacts[right].bytes {
			return artifacts[left].bytes > artifacts[right].bytes
		}
		if artifacts[left].path != artifacts[right].path {
			return artifacts[left].path < artifacts[right].path
		}
		return artifacts[left].name < artifacts[right].name
	})
	top := 20
	for index := range top {
		artifact := artifacts[index]
		t.Logf(
			"tail[%02d] kind=%s bytes=%d nodes=%d artifact=%s:%s",
			index+1,
			artifact.kind,
			artifact.bytes,
			artifact.nodes,
			artifact.path,
			artifact.name,
		)
	}
	for _, kind := range []string{
		"capability",
		"generic-alias",
		"generic-class",
		"generic-function",
	} {
		for _, artifact := range artifacts {
			if artifact.kind != kind {
				continue
			}
			t.Logf(
				"largest-%s count=%d bytes=%d nodes=%d artifact=%s:%s",
				kind,
				counts[kind],
				artifact.bytes,
				artifact.nodes,
				artifact.path,
				artifact.name,
			)
			break
		}
	}
}

type waveSevenTailArtifact struct {
	path  string
	name  string
	kind  string
	bytes int
	nodes int
}

var waveSevenTailBounds = map[string]struct {
	bytes int
	nodes int
}{
	"capability":    {bytes: 2_000, nodes: 400},
	"generic-alias": {bytes: 500, nodes: 100},
	// The generic pointer-equality fixture selects RuntimeSlice.address and
	// $view. The complete demanded class measures 6,280 bytes/1,289 nodes;
	// the full TS-Go corpus already selected the same unchanged class.
	"generic-class": {bytes: 6_500, nodes: 1_500},
	// The 2,500-byte bound includes explicit storage-facet conversion
	// capabilities. GenericIteratorCopy measures 2,341 bytes/304 nodes; the
	// prior one-facet ABI could not represent its T-backed struct field.
	"generic-function": {bytes: 2_500, nodes: 350},
}

func waveSevenTailIdentity(
	statement tsgo.Statement,
) (string, string, bool) {
	switch statement := statement.(type) {
	case tsgo.FunctionDeclaration:
		if strings.HasPrefix(statement.Name().Text(), "$goCapability_") {
			return statement.Name().Text(), "capability", true
		}
		if len(statement.TypeParameters()) != 0 {
			return statement.Name().Text(), "generic-function", true
		}
	case tsgo.ClassDeclaration:
		if len(statement.TypeParameters()) != 0 {
			return statement.Name().Text(), "generic-class", true
		}
	case tsgo.TypeAliasDeclaration:
		if len(statement.TypeParameters()) != 0 {
			return statement.Name().Text(), "generic-alias", true
		}
	}
	return "", "", false
}
