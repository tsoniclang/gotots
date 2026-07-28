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
	first, err := registry.internMapSpecialization(
		artifactKey,
		firstType,
		placement,
	)
	if err != nil {
		t.Fatal(err)
	}
	identical, err := registry.internMapSpecialization(
		artifactKey,
		identicalType,
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
		placement,
	); err == nil {
		t.Fatal("artifact-key collision joined non-identical map types")
	}
}

func TestMapSpecializationTargetPrefixCollisionFailsClosed(t *testing.T) {
	registry := NewRegistry()
	placement := generatedArtifactPlacement{
		kind: api.GeneratedArtifactPlacementCompilation,
	}
	firstKey := strings.Repeat("d", 64)
	if _, err := registry.internMapSpecialization(
		firstKey,
		types.NewMap(types.Typ[types.Int32], types.Typ[types.String]),
		placement,
	); err != nil {
		t.Fatal(err)
	}
	secondKey := firstKey[:mapTargetNameHexLength] +
		strings.Repeat(
			"e",
			64-mapTargetNameHexLength,
		)
	if secondKey == firstKey {
		t.Fatal("target-prefix collision fixture is identical")
	}
	if _, err := registry.internMapSpecialization(
		secondKey,
		types.NewMap(types.Typ[types.Int64], types.Typ[types.String]),
		placement,
	); err == nil {
		t.Fatal("target-name prefix collision was accepted")
	}
}
