package api_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/api/sourceevidence"
)

func TestSourceFileEvidenceRejectsIncompleteAndForeignOccurrences(t *testing.T) {
	fileSet := token.NewFileSet()
	syntax, err := parser.ParseFile(
		fileSet,
		"source.go",
		"package source\nvar Value = 1\n",
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	newEvidence := func(
		outputPath string,
		sourceIdentity string,
		sourceDigest string,
		programDigest string,
		goVersion string,
	) (sourceevidence.File, error) {
		return sourceevidence.NewFile(
			fileSet,
			syntax,
			"example.com/source",
			"example.com/source",
			"",
			"workspace",
			"example.com/source@workspace",
			outputPath,
			sourceIdentity,
			sourceDigest,
			programDigest,
			goVersion,
		)
	}
	for name, mutation := range map[string][5]string{
		"missing output":          {"", "checked-syntax:source.go", "source-digest", "program-digest", "go1.26"},
		"missing source identity": {"modules/example.com/source/source.ts", "", "source-digest", "program-digest", "go1.26"},
		"missing source digest":   {"modules/example.com/source/source.ts", "checked-syntax:source.go", "", "program-digest", "go1.26"},
		"missing program digest":  {"modules/example.com/source/source.ts", "checked-syntax:source.go", "source-digest", "", "go1.26"},
		"missing Go version":      {"modules/example.com/source/source.ts", "checked-syntax:source.go", "source-digest", "program-digest", ""},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := newEvidence(
				mutation[0],
				mutation[1],
				mutation[2],
				mutation[3],
				mutation[4],
			); err == nil {
				t.Fatal("incomplete source-file evidence was admitted")
			}
		})
	}
	evidence, err := newEvidence(
		"modules/example.com/source/source.ts",
		"checked-syntax:source.go",
		"source-digest",
		"program-digest",
		"go1.26",
	)
	if err != nil {
		t.Fatal(err)
	}
	occurrence, err := evidence.Occurrence(syntax.Decls[0])
	if err != nil {
		t.Fatal(err)
	}
	if occurrence.SourceIdentity() != "checked-syntax:source.go" ||
		occurrence.OutputPath() != "modules/example.com/source/source.ts" ||
		occurrence.Start() < 0 || occurrence.End() <= occurrence.Start() {
		t.Fatalf("source occurrence = %#v", occurrence)
	}
	foreign, err := parser.ParseFile(
		fileSet,
		"foreign.go",
		"package source\nvar Foreign = 2\n",
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := evidence.Occurrence(foreign.Decls[0]); err == nil {
		t.Fatal("foreign source occurrence was admitted")
	}
	if _, err := (api.Context{}).SourceOccurrence(syntax.Decls[0]); err == nil {
		t.Fatal("source occurrence without attached evidence was admitted")
	}
	packageEvidence, err := sourceevidence.NewPackage([]sourceevidence.File{evidence})
	if err != nil {
		t.Fatal(err)
	}
	context := (api.Context{}).WithSourceEvidence(packageEvidence)
	if _, err := context.SourceOccurrence(syntax.Decls[0]); err != nil {
		t.Fatal(err)
	}
}

func TestSourcePackageEvidenceResolvesSiblingFilesExactly(t *testing.T) {
	fileSet := token.NewFileSet()
	first, err := parser.ParseFile(
		fileSet,
		"first.go",
		"package source\ntype Record struct{}\n",
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	second, err := parser.ParseFile(
		fileSet,
		"second.go",
		"package source\nfunc (Record) Method() { type Local struct{} }\n",
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	newFile := func(syntax *ast.File, identity string) sourceevidence.File {
		evidence, evidenceErr := sourceevidence.NewFile(
			fileSet,
			syntax,
			"example.com/source",
			"example.com/source",
			"",
			"workspace",
			"example.com/source@workspace",
			"modules/example.com/source/first.ts",
			identity,
			identity+"-digest",
			"program-digest",
			"go1.26",
		)
		if evidenceErr != nil {
			t.Fatal(evidenceErr)
		}
		return evidence
	}
	evidence, err := sourceevidence.NewPackage([]sourceevidence.File{
		newFile(first, "checked-syntax:first.go"),
		newFile(second, "checked-syntax:second.go"),
	})
	if err != nil {
		t.Fatal(err)
	}
	method := second.Decls[0].(*ast.FuncDecl)
	local := method.Body.List[0].(*ast.DeclStmt).Decl.(*ast.GenDecl).Specs[0]
	occurrence, err := evidence.Occurrence(local)
	if err != nil {
		t.Fatal(err)
	}
	if occurrence.SourceIdentity() != "checked-syntax:second.go" ||
		occurrence.OutputPath() != "modules/example.com/source/first.ts" {
		t.Fatalf("sibling occurrence = %#v", occurrence)
	}
	foreign, err := parser.ParseFile(
		fileSet,
		"foreign.go",
		"package source\nvar Foreign = 1\n",
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := evidence.Occurrence(foreign.Decls[0]); err == nil {
		t.Fatal("foreign package occurrence was admitted")
	}
}
