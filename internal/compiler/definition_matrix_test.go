package compiler

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/scope/contract"
	"github.com/tsoniclang/gotots/internal/source"
)

func TestDefinitionFormsAndParentDirectedClassification(t *testing.T) {
	directory := t.TempDir()
	writeCompilerFile(
		t,
		directory,
		"go.mod",
		"module example.com/definition-matrix\n\ngo 1.26.0\n",
	)
	writeCompilerFile(t, directory, "matrix.go", `package matrix

const CompileTime = 1
var Left, Right = one(), two()
var Transform = func(value int) int {
	return func(next int) int { return value + next }(1)
}

type Counter int

func one() int { return 1 }
func two() int { return 2 }
func (counter Counter) Value() int { return int(counter) }
func Assembly(buffer []byte) (count int, err error)
`)
	writeCompilerFile(t, directory, "assembly_amd64.s", `#include "textflag.h"

TEXT ·Assembly(SB), NOSPLIT, $0-48
	MOVQ $0, count+24(FP)
	MOVQ $0, err+32(FP)
	MOVQ $0, err+40(FP)
	RET
`)
	inspection, err := inspectConstructsForTest(t, source.Request{
		Dir: directory, Patterns: []string{"."},
		ProviderContract: contract.DefaultID,
	})
	if err != nil {
		t.Fatal(err)
	}
	records := collectStage1Definitions(t, inspection)
	counts := map[identity.DefinitionKind]int{}
	for _, record := range records {
		counts[record.definition.Kind()]++
	}
	want := map[identity.DefinitionKind]int{
		identity.DefinitionFuncDecl:           3,
		identity.DefinitionFuncLit:            2,
		identity.DefinitionPackageInitializer: 2,
		identity.DefinitionBodylessDecl:       1,
		identity.DefinitionImplicit:           1,
	}
	for kind, expected := range want {
		if counts[kind] != expected {
			t.Errorf("definitions[%s]=%d, want %d", kind, counts[kind], expected)
		}
	}
	if len(records) != 9 {
		t.Fatalf("definitions=%d, want 9", len(records))
	}
	if len(findDefinitionsByName(records, "CompileTime")) != 0 {
		t.Fatal("package const ValueSpec became an initializer definition")
	}
	initializers := findDefinitionsByName(records, "Left,Right")
	if len(initializers) != 1 {
		t.Fatalf("multi-name initializer definitions=%d, want 1", len(initializers))
	}
	if entries := records[initializers[0]].boundary.Entries(); len(entries) != 2 {
		t.Fatalf("ordered initializer entries=%d, want 2", len(entries))
	}
	transformInitializers := findDefinitionsByName(records, "Transform")
	literals := findDefinitionsByName(records, "func literal")
	if len(transformInitializers) != 1 || len(literals) != 2 {
		t.Fatalf(
			"Transform initializer/literals=%d/%d, want 1/2",
			len(transformInitializers),
			len(literals),
		)
	}
	outerLiteral := identity.DefinitionID{}
	innerLiteral := identity.DefinitionID{}
	for _, literal := range literals {
		parent := records[literal].site.ParentDefinition()
		switch parent {
		case transformInitializers[0]:
			outerLiteral = literal
		default:
			innerLiteral = literal
		}
	}
	if outerLiteral.IsZero() ||
		innerLiteral.IsZero() ||
		records[innerLiteral].site.ParentDefinition() != outerLiteral {
		t.Fatalf(
			"literal topology initializer=%s outer=%s inner=%s innerParent=%s",
			transformInitializers[0],
			outerLiteral,
			innerLiteral,
			records[innerLiteral].site.ParentDefinition(),
		)
	}
	for definition, record := range records {
		if record.site.Kind() == structure.DefinitionSiteSource &&
			record.site.Terminal() != definition.Root() {
			t.Errorf(
				"definition %s terminal=%s, want construct root %s",
				definition,
				record.site.Terminal(),
				definition.Root(),
			)
		}
	}
}

func TestHeaderAndExecutionContentDomainsAreIndependent(t *testing.T) {
	base := inspectContentVariant(t, "value", "1")
	bodyEdit := inspectContentVariant(t, "value", "2")
	headerEdit := inspectContentVariant(t, "other", "1")

	if base.definition.ID() != bodyEdit.definition.ID() ||
		base.definition.ID() != headerEdit.definition.ID() {
		t.Fatal("equal-width edits changed construct identity")
	}
	if base.header.Digest() != bodyEdit.header.Digest() {
		t.Fatal("body-only edit changed header digest")
	}
	if base.boundary.CombinedDigest() == bodyEdit.boundary.CombinedDigest() {
		t.Fatal("body-only edit did not change execution digest")
	}
	if base.header.Digest() == headerEdit.header.Digest() {
		t.Fatal("header-only edit did not change header digest")
	}
	if base.boundary.CombinedDigest() != headerEdit.boundary.CombinedDigest() {
		t.Fatal("header-only edit contaminated execution digest")
	}
}

func TestBodylessDefinitionCannotSelectFullSemantics(t *testing.T) {
	directory := t.TempDir()
	writeCompilerFile(
		t,
		directory,
		"go.mod",
		"module example.com/bodyless-depth\n\ngo 1.26.0\n",
	)
	writeCompilerFile(
		t,
		directory,
		"bodyless.go",
		"package bodyless\n\nfunc Assembly(value int) int\n",
	)
	writeCompilerFile(t, directory, "bodyless_amd64.s", `#include "textflag.h"

TEXT ·Assembly(SB), NOSPLIT, $0-16
	MOVQ value+0(FP), AX
	MOVQ AX, ret+8(FP)
	RET
`)
	base, err := inspectConstructsForTest(t, source.Request{
		Dir: directory, Patterns: []string{"."},
		ProviderContract: contract.DefaultID,
	})
	if err != nil {
		t.Fatal(err)
	}
	definitions := findDefinitionsByName(
		collectStage1Definitions(t, base),
		"Assembly",
	)
	if len(definitions) != 1 ||
		definitions[0].Kind() != identity.DefinitionBodylessDecl {
		t.Fatalf("Assembly definitions=%v", definitions)
	}
	contractPath := writeDepthContract(
		t,
		"invalid-bodyless-full@v1",
		definitions[0],
	)
	if _, err := inspectConstructsForTest(t, source.Request{
		Dir: directory, Patterns: []string{"."},
		ProviderContract:         "invalid-bodyless-full@v1",
		ProviderContractArtifact: contractPath,
	}); err == nil {
		t.Fatal("bodyless definition selected as full semantic")
	}
}

func inspectContentVariant(
	t *testing.T,
	parameter string,
	result string,
) stage1DefinitionRecord {
	t.Helper()
	directory := t.TempDir()
	writeCompilerFile(
		t,
		directory,
		"go.mod",
		"module example.com/content-domain\n\ngo 1.26.0\n",
	)
	writeCompilerFile(
		t,
		directory,
		"target.go",
		"package content\n\nfunc Target("+parameter+" int) int {\n\treturn "+result+"\n}\n",
	)
	inspection, err := inspectConstructsForTest(t, source.Request{
		Dir: directory, Patterns: []string{"."},
		ProviderContract: contract.DefaultID,
	})
	if err != nil {
		t.Fatal(err)
	}
	records := collectStage1Definitions(t, inspection)
	definitions := findDefinitionsByName(records, "Target")
	if len(definitions) != 1 {
		t.Fatalf("Target definitions=%d, want 1", len(definitions))
	}
	return records[definitions[0]]
}
