package tsgo

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"
)

func TestGeneratedBindingsCoverEverySchemaDefinition(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(schemaDirectory(), "upstream", "ast.json"))
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Nodes struct {
			Definitions map[string]json.RawMessage `json:"definitions"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}
	expected := make([]string, 0, len(schema.Nodes.Definitions))
	for name := range schema.Nodes.Definitions {
		expected = append(expected, name)
	}
	sort.Strings(expected)
	actual := generatedSchemaNodeNames[:]
	if !slices.Equal(actual, expected) {
		t.Fatalf("generated schema definitions differ:\nactual: %q\nexpected: %q", actual, expected)
	}
}

func TestGeneratedChildPropertiesMatchPinnedProtocol(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(schemaDirectory(), "upstream", "protocol.generated.ts"))
	if err != nil {
		t.Fatal(err)
	}
	start := bytes.Index(data, []byte("export const childProperties:"))
	if start < 0 {
		t.Fatal("childProperties declaration not found")
	}
	end := bytes.Index(data[start:], []byte("\n};"))
	if end < 0 {
		t.Fatal("childProperties declaration is unterminated")
	}
	entryPattern := regexp.MustCompile(`(?m)^\s+\[SyntaxKind\.([A-Za-z0-9]+)\]: \[([^\]]*)\],\r?$`)
	propertyPattern := regexp.MustCompile(`"([^"]+)"`)
	expected := make(map[string][]string)
	for _, entry := range entryPattern.FindAllSubmatch(data[start:start+end], -1) {
		properties := propertyPattern.FindAllSubmatch(entry[2], -1)
		values := make([]string, len(properties))
		for index, property := range properties {
			values[index] = string(property[1])
		}
		expected[string(entry[1])] = values
	}
	if !reflect.DeepEqual(generatedChildProperties, expected) {
		t.Fatalf(
			"generated child properties differ from pinned protocol:\ngenerated: %#v\npinned: %#v",
			generatedChildProperties,
			expected,
		)
	}
}

func TestGeneratedFilesReproduce(t *testing.T) {
	output := t.TempDir()
	command := exec.Command(
		"go",
		"run",
		"./generate",
		"-schema",
		filepath.Join("..", "..", "..", "schema", "tsgo"),
		"-output",
		output,
	)
	if data, err := command.CombinedOutput(); err != nil {
		t.Fatalf("generate: %v\n%s", err, data)
	}
	for _, name := range []string{
		"contract_generated_test.go",
		"encoding_generated.go",
		"factories_generated.go",
		"kinds_generated.go",
		"nodes_generated.go",
		"protocol_generated.go",
	} {
		actual, err := os.ReadFile(filepath.Join(output, name))
		if err != nil {
			t.Fatal(err)
		}
		expected, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(actual, expected) {
			t.Fatalf("%s is stale; run go generate ./internal/target/tsgo", name)
		}
	}
}

func TestGeneratedFactoryRejectsInvalidChildAtCompileTime(t *testing.T) {
	command := exec.Command("go", "test", "./testdata/invalid-child")
	data, err := command.CombinedOutput()
	if err == nil {
		t.Fatal("invalid child fixture compiled")
	}
	if !strings.Contains(string(data), "does not implement tsgo.BindingName") {
		t.Fatalf("unexpected compile failure:\n%s", data)
	}
}

func TestSourceFileDataMatchesPinnedHandWrittenContract(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(schemaDirectory(), "upstream", "ast.ts"))
	if err != nil {
		t.Fatal(err)
	}
	sourceFileFields := interfaceFields(t, string(data), "SourceFile")
	actual := []string{"kind", "statements", "endOfFileToken"}
	valueType := reflect.TypeFor[SourceFileData]()
	for index := range valueType.NumField() {
		actual = append(actual, lowerInitialism(valueType.Field(index).Name))
	}
	sort.Strings(actual)
	sort.Strings(sourceFileFields)
	if !slices.Equal(actual, sourceFileFields) {
		t.Fatalf("SourceFile fields differ:\nactual: %q\npinned: %q", actual, sourceFileFields)
	}
}

func TestSourceFileDataIsCopied(t *testing.T) {
	factory := NewFactory()
	references := []FileReference{{FileName: "a.d.ts"}}
	source := factory.SourceFile(
		nil,
		factory.EndOfFile(),
		SourceFileData{ReferencedFiles: references},
	)
	references[0].FileName = "changed.d.ts"
	data := source.SourceData()
	if data.ReferencedFiles[0].FileName != "a.d.ts" {
		t.Fatal("SourceFile retained caller-owned reference storage")
	}
	data.ReferencedFiles[0].FileName = "changed-again.d.ts"
	if source.SourceData().ReferencedFiles[0].FileName != "a.d.ts" {
		t.Fatal("SourceData exposed backing storage")
	}
}

func interfaceFields(t *testing.T, source string, name string) []string {
	t.Helper()
	pattern := regexp.MustCompile(`(?s)export interface ` + regexp.QuoteMeta(name) + ` extends [^{]+ \{(.*?)\r?\n\}`)
	match := pattern.FindStringSubmatch(source)
	if match == nil {
		t.Fatalf("interface %s not found", name)
	}
	fieldPattern := regexp.MustCompile(`(?m)^\s+readonly ([A-Za-z][A-Za-z0-9]*)(?:\?)?:`)
	matches := fieldPattern.FindAllStringSubmatch(match[1], -1)
	result := make([]string, 0, len(matches))
	for _, field := range matches {
		result = append(result, field[1])
	}
	return result
}

func lowerInitialism(name string) string {
	switch {
	case strings.HasPrefix(name, "JSDoc"):
		return "jsDoc" + name[len("JSDoc"):]
	case strings.HasPrefix(name, "URL"):
		return "url" + name[len("URL"):]
	default:
		return strings.ToLower(name[:1]) + name[1:]
	}
}
