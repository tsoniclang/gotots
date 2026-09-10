package reflectvalue_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func compileReflectFixture(
	test *testing.T,
	project string,
	source string,
	roots []string,
	profiles ...emit.IntegerRepresentation,
) emit.ProgramEmission {
	test.Helper()
	emission, err := tryCompileReflectFixture(test, project, source, roots, profiles...)
	if err != nil {
		test.Fatal(err)
	}
	return emission
}

func tryCompileReflectFixture(
	test *testing.T,
	project string,
	source string,
	roots []string,
	profiles ...emit.IntegerRepresentation,
) (emit.ProgramEmission, error) {
	test.Helper()
	writeProgramFile(test, filepath.Join(project, "go.mod"), "module example.com/reflectvalue\n\ngo 1.26.4\n")
	writeProgramFile(test, filepath.Join(project, "source.go"), source)
	program, err := load.Load(context.Background(), load.Request{
		Directory:    project,
		Pattern:      ".",
		BuildProfile: linkedProviderBuildProfile(test),
	})
	if err != nil {
		return emit.ProgramEmission{}, err
	}
	scope := program.Roots()[0].Types().Scope()
	selected := make([]emit.Root, 0, len(roots))
	for _, name := range roots {
		selected = append(selected, mustRoot(test, scope.Lookup(name)))
	}
	options := emit.Options{
		IntegerRepresentation: emit.IntegerRepresentationNumber,
		EvaluationOrder:       emit.EvaluationOrderDirect,
		StandardLibrary:       linkedProviderCertificate(test),
	}
	if len(profiles) > 1 {
		test.Fatal("fixture requires one explicit integer profile")
	}
	if len(profiles) == 1 {
		options.IntegerRepresentation = profiles[0]
	}
	return emit.CompileWithOptions(program, selected, options)
}
