package naming

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestReflectionRawPointerDemandClosesEarlyAndLateDescriptors(t *testing.T) {
	for _, early := range []bool{false, true} {
		registry := NewRegistry()
		method := reflectionRawMethod("reflect")
		if early {
			if err := registry.observeReflectionRawPointerUse(method); err != nil {
				t.Fatal(err)
			}
		}
		bind := func(key string) {
			_, err := registry.internReflectionType(key, types.NewPointer(types.Typ[types.Uint32]), reflectionContractType(), key)
			if err != nil {
				t.Fatal(err)
			}
			registry.reflectionValueDemands[key] = struct{}{}
		}
		bind("First")
		if !early {
			requests, err := registry.FlushReflectionRawPointerDemands()
			if err != nil || len(requests) != 0 {
				t.Fatal("ordinary reflection acquired a raw-pointer facet")
			}
			if err := registry.observeReflectionRawPointerUse(method); err != nil {
				t.Fatal(err)
			}
		}
		for _, key := range []string{"First", "Later"} {
			if key == "Later" {
				bind(key)
			}
			requests, err := registry.FlushReflectionRawPointerDemands()
			if err != nil || len(requests) != 1 {
				t.Fatalf("raw demand batch: %d, %v", len(requests), err)
			}
			requirement, valid := requests[0].DeclarationRequirement()
			artifact, selected := requirement.GeneratedArtifact()
			if !valid || !selected || !requirement.Valid() || requirement.Kind() != api.DeclarationRequirementReflectionRawPointer || artifact.ArtifactKey() != key {
				t.Fatal("raw demand did not retain its exact descriptor and distinct facet")
			}
			requests, err = registry.FlushReflectionRawPointerDemands()
			if err != nil || len(requests) != 0 {
				t.Fatal("settled raw demand repeats")
			}
		}
	}
}

func TestSameSpelledForeignMethodDoesNotDemandReflectionRawPointers(t *testing.T) {
	registry := NewRegistry()
	if err := registry.observeReflectionRawPointerUse(reflectionRawMethod("example.com/reflect")); err != nil {
		t.Fatal(err)
	}
	if registry.reflectionRawPointerSelected {
		t.Fatal("foreign UnsafePointer spelling selected a provider facet")
	}
}

func reflectionRawMethod(path string) *types.Func {
	owner := types.NewPackage(path, "reflect")
	name := types.NewTypeName(token.NoPos, owner, "Value", nil)
	receiver := types.NewNamed(name, types.NewStruct(nil, nil), nil)
	signature := types.NewSignatureType(types.NewVar(token.NoPos, owner, "receiver", receiver), nil, nil, nil,
		types.NewTuple(types.NewVar(token.NoPos, owner, "", types.Typ[types.UnsafePointer])), false)
	return types.NewFunc(token.NoPos, owner, "UnsafePointer", signature)
}
