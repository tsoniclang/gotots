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
	client, err := tsgo.StartClient(repository, provider)
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
		"provider-compress-gzip.ts",
	))
	if err != nil {
		t.Fatal(err)
	}
	var target tsgo.ProjectExport
	for _, selected := range exports {
		if selected.Name() == "CanonicalGzipReader" {
			target = selected
			break
		}
	}
	if target.Name() == "" {
		t.Fatal("canonical gzip stateful target is absent")
	}
	fields := []gostdlib.ProviderStatefulProfileFieldDocument{{Member: "Header"}}
	methods := []gostdlib.ProviderStatefulProfileMethodDocument{
		{Member: "Close"},
		{Member: "Read"},
	}
	if err := verifyStatefulProfileTargetMembers(target, fields, methods, nil); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		fields []gostdlib.ProviderStatefulProfileFieldDocument
		want   string
	}{
		{
			name: "omitted",
			want: "unowned public instance member",
		},
		{
			name: "extra",
			fields: []gostdlib.ProviderStatefulProfileFieldDocument{
				{Member: "Header"},
				{Member: "Invented"},
			},
			want: "omits an owned public instance member",
		},
		{
			name:   "renamed",
			fields: []gostdlib.ProviderStatefulProfileFieldDocument{{Member: "Renamed"}},
			want:   "unowned public instance member",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := verifyStatefulProfileTargetMembers(target, test.fields, methods, nil)
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
	client, err := tsgo.StartClient(repository, provider)
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
		if selected.Name() == "CanonicalPathError" {
			target = selected
			break
		}
	}
	if target.Name() == "" {
		t.Fatal("canonical path-error target is absent")
	}
	fields := []gostdlib.ProviderStatefulProfileFieldDocument{
		{Member: "Err"},
		{Member: "Op"},
		{Member: "Path"},
	}
	methods := []gostdlib.ProviderStatefulProfileMethodDocument{
		{Member: "Error"},
		{Member: "Unwrap"},
	}
	operations := []gostdlib.FacetCapability{
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
		operations[:1],
	)
	if err == nil || !strings.Contains(err.Error(), "unowned public static member") {
		t.Fatalf("dropped storage-operation error = %v", err)
	}
}
