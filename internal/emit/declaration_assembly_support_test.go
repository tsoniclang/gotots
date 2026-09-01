package emit

import (
	"context"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	canonicalsourcefact "github.com/tsoniclang/gotots/internal/emit/sourcefact"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func drainProgramSession(t *testing.T, session *programSession) {
	t.Helper()
	if err := session.settle(); err != nil {
		t.Fatal(err)
	}
}

func declarationForObject(
	t *testing.T,
	session *programSession,
	object types.Object,
) *targetDeclaration {
	t.Helper()
	site := session.sites[object]
	builder, err := session.builder(site)
	if err != nil {
		t.Fatal(err)
	}
	index, ok := builder.indexByOwner[sourceArtifactOwner(object)]
	if !ok {
		t.Fatalf("declaration %s is absent", object.Name())
	}
	return &builder.declarations[index]
}

func assertOneFinalDeclarationAssembly(
	t *testing.T,
	files []TargetFile,
	owner string,
) {
	t.Helper()
	classCount := 0
	operationCounts := map[string]int{
		"$zero":  0,
		"$copy":  0,
		"$equal": 0,
	}
	for _, file := range files {
		if file.Kind() != TargetFileSource {
			continue
		}
		for _, statement := range file.SourceFile().Statements() {
			switch statement := statement.(type) {
			case tsgo.ClassDeclaration:
				if statement.Name().Text() != owner {
					continue
				}
				classCount++
				for _, member := range statement.Members() {
					method, ok := member.(tsgo.MethodDeclaration)
					if !ok {
						continue
					}
					operationCounts[method.Name().(tsgo.Identifier).Text()]++
				}
			case tsgo.FunctionDeclaration:
				if strings.HasPrefix(statement.Name().Text(), owner+"$") {
					t.Fatalf("top-level operation helper %s remains", statement.Name().Text())
				}
			}
		}
	}
	if classCount != 1 {
		t.Fatalf("%s final class count = %d, want one", owner, classCount)
	}
	for name, count := range operationCounts {
		if count != 1 {
			t.Fatalf("%s.%s final definition count = %d, want one", owner, name, count)
		}
	}
}

func TestContextualConstantFactsExactJoinEmittedProjectionBindings(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(directory, "go.mod"),
		[]byte("module example.com/constantfacts\n\ngo 1.26.4\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(directory, "source.go"),
		[]byte("package constantfacts\n\nconst Huge = 1 << 63\nconst Other = 1\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	program, err := load.Load(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	sourcePackage := program.Roots()[0]
	constant := sourcePackage.Types().Scope().Lookup("Huge").(*types.Const)
	other := sourcePackage.Types().Scope().Lookup("Other").(*types.Const)
	session, err := newProgramSession(program, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	site := session.sites[constant]
	emitter := session.emitters[sourcePackage]
	context, err := emitter.fileContext(site.SourceFile.Syntax(), site.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	origin, err := canonicalsourcefact.Origin(
		site.Source,
		site.SourceFile,
		site.OutputPath,
		site.Occurrence,
	)
	if err != nil {
		t.Fatal(err)
	}
	baseName, err := context.Names().Declare(constant)
	if err != nil {
		t.Fatal(err)
	}
	projectionName, err := api.ConstantProjectionName(baseName, types.Uint64)
	if err != nil {
		t.Fatal(err)
	}
	statement := session.factory.VariableStatement(
		[]tsgo.ModifierLike{session.factory.ExportKeyword()},
		session.factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{session.factory.VariableDeclaration(
				session.factory.Identifier(projectionName),
				nil,
				nil,
				session.factory.BigIntLiteral("9223372036854775808n", tsgo.TokenFlagsNone),
			)},
			tsgo.NodeFlagsConst,
		),
	)
	requirement, err := api.NewConstantProjectionRequirement(constant, types.Uint64)
	if err != nil {
		t.Fatal(err)
	}
	facts, err := canonicalsourcefact.DeclarationWithRequirements(
		context,
		constant,
		origin,
		[]tsgo.Statement{statement},
		[]api.DeclarationRequirement{requirement},
		[]string{projectionName},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(facts.Statements()) != 2 {
		t.Fatalf("contextual constant fact statements = %d, want 2", len(facts.Statements()))
	}
	for name, mutation := range map[string]struct {
		requirements []api.DeclarationRequirement
		bindings     []string
	}{
		"missing binding": {[]api.DeclarationRequirement{requirement}, nil},
		"wrong binding":   {[]api.DeclarationRequirement{requirement}, []string{projectionName + "$wrong"}},
		"duplicate projection": {
			[]api.DeclarationRequirement{requirement, requirement},
			[]string{projectionName},
		},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := canonicalsourcefact.DeclarationWithRequirements(
				context,
				constant,
				origin,
				[]tsgo.Statement{statement},
				mutation.requirements,
				mutation.bindings,
			); err == nil {
				t.Fatal("inexact contextual constant fact denominator was admitted")
			}
		})
	}
	foreign, err := api.NewConstantProjectionRequirement(other, types.Uint64)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := canonicalsourcefact.DeclarationWithRequirements(
		context,
		constant,
		origin,
		[]tsgo.Statement{statement},
		[]api.DeclarationRequirement{foreign},
		[]string{projectionName},
	); err == nil {
		t.Fatal("foreign contextual constant fact requirement was admitted")
	}
}
