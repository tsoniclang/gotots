package emit

import (
	"context"
	"go/types"
	"os"
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestAnonymousStructDemandsUseExistingArtifactFixedPoint(t *testing.T) {
	program := loadAnonymousStructAssemblyFixture(t)
	scope := program.Roots()[0].Types().Scope()
	definition := scope.Lookup("Definition").(*types.Func)
	copyValue := scope.Lookup("CopyValue").(*types.Func)
	equal := scope.Lookup("Equal").(*types.Func)
	session, err := newProgramSession(program, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}

	if err := session.require(definition); err != nil {
		t.Fatal(err)
	}
	drainProgramSession(t, session)
	artifact := onlyAnonymousStructArtifact(t, session)
	owner := api.MustGeneratedArtifactOwner(artifact)
	initialStaticRevision := session.artifacts.FacetRevision(
		owner,
		api.ArtifactFacetStaticSurface,
	)
	if _, fabricated := owner.Source(); fabricated {
		t.Fatal("anonymous artifact fabricated a go/types source object")
	}

	if err := session.require(copyValue); err != nil {
		t.Fatal(err)
	}
	drainProgramSession(t, session)
	copyStaticRevision := session.artifacts.FacetRevision(
		owner,
		api.ArtifactFacetStaticSurface,
	)
	if copyStaticRevision != initialStaticRevision+1 {
		t.Fatalf(
			"copy static revision = %d, want %d",
			copyStaticRevision,
			initialStaticRevision+1,
		)
	}
	if declarationForObject(t, session, copyValue).reconstructions != 1 {
		t.Fatal("copy consumer did not reconstruct through the existing artifact graph")
	}

	if err := session.require(equal); err != nil {
		t.Fatal(err)
	}
	drainProgramSession(t, session)
	if revision := session.artifacts.FacetRevision(
		owner,
		api.ArtifactFacetStaticSurface,
	); revision != copyStaticRevision+1 {
		t.Fatalf("equal static revision = %d, want %d", revision, copyStaticRevision+1)
	}
	builder := session.builders[artifact.OutputPath()]
	index, ok := builder.indexByOwner[owner]
	if !ok || builder.declarations[index].reconstructions != 2 {
		t.Fatalf("anonymous artifact reconstruction = %#v, %t", builder, ok)
	}
	if session.artifacts.HasPending() ||
		session.requirements.hasPending() ||
		session.scheduler.hasPending() {
		t.Fatal("anonymous-struct requirements did not converge")
	}
}

func loadAnonymousStructAssemblyFixture(t *testing.T) *load.Program {
	t.Helper()
	directory := t.TempDir()
	writeAnonymousAssemblyFile(
		t,
		filepath.Join(directory, "go.mod"),
		"module example.com/anonymousassembly\n\ngo 1.26.4\n",
	)
	writeAnonymousAssemblyFile(t, filepath.Join(directory, "source.go"), `package anonymousassembly

func Definition(value struct{ Field int32 }) int32 {
	return value.Field
}

func CopyValue(value struct{ Field int32 }) struct{ Field int32 } {
	copy := value
	return copy
}

func Equal(left, right struct{ Field int32 }) bool {
	return left == right
}
`)
	program, err := load.Load(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	return program
}

func writeAnonymousAssemblyFile(
	t *testing.T,
	path string,
	content string,
) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func onlyAnonymousStructArtifact(
	t *testing.T,
	session *programSession,
) *api.GeneratedArtifact {
	t.Helper()
	artifacts := session.registry.GeneratedArtifacts(
		api.GeneratedArtifactAnonymousStruct,
	)
	if len(artifacts) != 1 {
		t.Fatalf(
			"anonymous artifacts = %d, want one",
			len(artifacts),
		)
	}
	return artifacts[0]
}
