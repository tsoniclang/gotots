package naming

import (
	"go/types"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/output"
)

func TestGeneratedArtifactImportsUseShortestCollisionFreeFamilyName(t *testing.T) {
	firstName := "$goMap$MapOf_int32_To_string"
	secondName := "$goMap$MapOf_int64_To_string"
	first := generatedImportMapArtifact(t, strings.Repeat("1", 64), firstName)
	second := generatedImportMapArtifact(t, strings.Repeat("2", 64), secondName)
	file := &File{
		owner: &Owner{
			sourceNameBases: map[string]struct{}{},
		},
		importNames:     make(map[string]struct{}),
		generatedNames:  make(map[string]struct{}),
		artifactImports: make(map[generatedArtifactImport]string),
	}

	if got := file.generatedArtifactLocalName(first, firstName); got != "GoMap" {
		t.Fatalf("first map local name = %q, want GoMap", got)
	}
	if got := file.generatedArtifactLocalName(second, secondName); got != secondName {
		t.Fatalf("colliding map local name = %q, want full semantic name %q", got, secondName)
	}
	if got := file.generatedArtifactLocalName(first, firstName); got != "GoMap" {
		t.Fatalf("cached first map local name = %q, want GoMap", got)
	}
}

func TestGeneratedArtifactImportAvoidsAuthoredAndSemanticNameCollisions(t *testing.T) {
	name := "$goMap$MapOf_int32_To_string"
	artifact := generatedImportMapArtifact(t, strings.Repeat("3", 64), name)
	file := &File{
		owner: &Owner{
			sourceNameBases: map[string]struct{}{
				"GoMap": {},
				name:    {},
			},
		},
		importNames:     make(map[string]struct{}),
		generatedNames:  make(map[string]struct{}),
		artifactImports: make(map[generatedArtifactImport]string),
	}

	got := file.generatedArtifactLocalName(artifact, name)
	if got == "GoMap" || got == name || file.sourceNameExists(got) {
		t.Fatalf("collision-qualified map local name = %q", got)
	}
	if !strings.HasPrefix(got, "GoMap__from_") {
		t.Fatalf("collision-qualified map local name = %q, want semantic qualifier", got)
	}
}

func generatedImportMapArtifact(
	t *testing.T,
	key string,
	name string,
) *api.GeneratedArtifact {
	t.Helper()
	artifact, err := api.NewCompilationGeneratedArtifact(
		api.GeneratedArtifactMapSpecialization,
		types.NewMap(types.Typ[types.Int32], types.Typ[types.String]),
		key,
		name,
		output.MapSpecializationSupportPath,
	)
	if err != nil {
		t.Fatal(err)
	}
	return artifact
}
