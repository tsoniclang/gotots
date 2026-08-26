package certify

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestStatefulProfileTargetFieldsExactJoin(t *testing.T) {
	repository, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	provider := filepath.Join(repository, "gostdlib")
	_, selectedTSGo := resolveTestTools(t, repository)
	client, err := tsgo.StartClientWithTool(selectedTSGo, provider)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close TS-Go client: %v", err)
		}
	})
	project, err := client.OpenProject(filepath.Join(provider, "tsconfig.json"))
	if err != nil {
		t.Fatal(err)
	}
	exports, err := project.Exports(filepath.Join(
		provider,
		"src",
		"internal",
		"facets",
		"provider-compress-gzip-direct.ts",
	))
	if err != nil {
		t.Fatal(err)
	}
	var target tsgo.ProjectExport
	for _, selected := range exports {
		if selected.Name() == "DirectGzipReader" {
			target = selected
			break
		}
	}
	if target.Name() == "" {
		t.Fatal("direct gzip stateful target is absent")
	}
	fields := []gostdlib.ProviderStructFieldDocument{{Member: "Header"}}
	methods := []gostdlib.ProviderStatefulProfileMethodDocument{
		{Member: "Close"},
		{Member: "Read"},
	}
	operations := []gostdlib.FacetCapability{
		gostdlib.FacetCapabilityAssign,
		gostdlib.FacetCapabilityCopy,
	}
	if err := verifyStatefulProfileTargetMembers(
		target,
		fields,
		methods,
		operations,
	); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		fields []gostdlib.ProviderStructFieldDocument
		want   string
	}{
		{
			name: "omitted",
			want: "unowned public instance member",
		},
		{
			name: "extra",
			fields: []gostdlib.ProviderStructFieldDocument{
				{Member: "Header"},
				{Member: "Invented"},
			},
			want: "omits an owned public instance member",
		},
		{
			name:   "renamed",
			fields: []gostdlib.ProviderStructFieldDocument{{Member: "Renamed"}},
			want:   "unowned public instance member",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := verifyStatefulProfileTargetMembers(
				target,
				test.fields,
				methods,
				operations,
			)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("field mutation error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestStatefulProfileOperationsExactJoin(t *testing.T) {
	repository, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	provider := filepath.Join(repository, "gostdlib")
	_, selectedTSGo := resolveTestTools(t, repository)
	client, err := tsgo.StartClientWithTool(selectedTSGo, provider)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close TS-Go client: %v", err)
		}
	})
	project, err := client.OpenProject(filepath.Join(provider, "tsconfig.json"))
	if err != nil {
		t.Fatal(err)
	}
	exports, err := project.Exports(filepath.Join(
		provider,
		"src",
		"internal",
		"facets",
		"named-io-fs.ts",
	))
	if err != nil {
		t.Fatal(err)
	}
	var target tsgo.ProjectExport
	for _, selected := range exports {
		if selected.Name() == "DirectPathError" {
			target = selected
			break
		}
	}
	if target.Name() == "" {
		t.Fatal("direct path-error target is absent")
	}
	fields := []gostdlib.ProviderStructFieldDocument{
		{Member: "Err"},
		{Member: "Op"},
		{Member: "Path"},
	}
	methods := []gostdlib.ProviderStatefulProfileMethodDocument{
		{Member: "Error"},
		{Member: "Unwrap"},
	}
	operations := []gostdlib.FacetCapability{
		gostdlib.FacetCapabilityAssign,
		gostdlib.FacetCapabilityCopy,
		gostdlib.FacetCapabilityEqual,
		gostdlib.FacetCapabilityHash,
		gostdlib.FacetCapabilityMake,
		gostdlib.FacetCapabilityStorage,
	}
	if err := verifyStatefulProfileTargetMembers(
		target,
		fields,
		methods,
		operations,
	); err != nil {
		t.Fatal(err)
	}
	err = verifyStatefulProfileTargetMembers(
		target,
		fields,
		methods,
		operations[:len(operations)-1],
	)
	if err == nil || !strings.Contains(err.Error(), "unowned public static member") {
		t.Fatalf("dropped storage-operation error = %v", err)
	}
}
