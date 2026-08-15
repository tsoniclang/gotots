package providerinterfacebridge

import (
	"reflect"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestImplementedContractNamesAreExactOrderedAndUnique(t *testing.T) {
	first, err := api.NewInterfaceContractReference(
		"First",
		"FirstContract",
		"isFirst",
	)
	if err != nil {
		t.Fatal(err)
	}
	second, err := api.NewInterfaceContractReference(
		"Second",
		"SecondContract",
		"isSecond",
	)
	if err != nil {
		t.Fatal(err)
	}
	selected := implementedContractNames(
		"Base",
		[]capabilitySelection{
			{canonical: first},
			{canonical: second},
			{canonical: first},
		},
	)
	want := []string{"Base", "First", "Second"}
	if !reflect.DeepEqual(selected, want) {
		t.Fatalf("implemented contracts = %q, want %q", selected, want)
	}
}
