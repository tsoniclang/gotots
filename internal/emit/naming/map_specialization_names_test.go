package naming

import (
	"go/types"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestMapSpecializationCanonicalizationJoinsOnlyIdenticalGoTypes(
	t *testing.T,
) {
	registry := NewRegistry()
	placement := generatedArtifactPlacement{
		kind: api.GeneratedArtifactPlacementCompilation,
	}
	firstType := types.NewMap(types.Typ[types.Int32], types.Typ[types.String])
	identicalType := types.NewMap(types.Typ[types.Int32], types.Typ[types.String])
	artifactKey := strings.Repeat("a", 64)
	name, err := semanticGeneratedTypeName("$goMap$", firstType, nil)
	if err != nil {
		t.Fatal(err)
	}
	first, err := registry.internMapSpecialization(
		artifactKey,
		firstType,
		name,
		placement,
	)
	if err != nil {
		t.Fatal(err)
	}
	identical, err := registry.internMapSpecialization(
		artifactKey,
		identicalType,
		name,
		placement,
	)
	if err != nil {
		t.Fatal(err)
	}
	if first.owner != identical.owner {
		t.Fatal("identical map types did not join one generated artifact")
	}

	collidingType := types.NewMap(
		types.Typ[types.Int64],
		types.Typ[types.String],
	)
	if _, err := registry.internMapSpecialization(
		artifactKey,
		collidingType,
		name,
		placement,
	); err == nil {
		t.Fatal("artifact-key collision joined non-identical map types")
	}
}

func TestMapSpecializationSemanticNameCollisionFailsClosed(t *testing.T) {
	registry := NewRegistry()
	placement := generatedArtifactPlacement{
		kind: api.GeneratedArtifactPlacementCompilation,
	}
	firstKey := strings.Repeat("d", 64)
	firstType := types.NewMap(types.Typ[types.Int32], types.Typ[types.String])
	firstName, err := semanticGeneratedTypeName("$goMap$", firstType, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := registry.internMapSpecialization(
		firstKey,
		firstType,
		firstName,
		placement,
	); err != nil {
		t.Fatal(err)
	}
	secondKey := strings.Repeat("e", 64)
	if _, err := registry.internMapSpecialization(
		secondKey,
		types.NewMap(types.Typ[types.Int64], types.Typ[types.String]),
		firstName,
		placement,
	); err == nil {
		t.Fatal("semantic target-name collision was accepted")
	}
}
