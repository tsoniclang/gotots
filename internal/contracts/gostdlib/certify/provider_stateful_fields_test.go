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
		if selected.Name() == "CanonicalGzipReaderReadAsync" {
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
	if err := verifyStatefulProfileTargetMembers(target, fields, methods); err != nil {
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
			err := verifyStatefulProfileTargetMembers(target, test.fields, methods)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("field mutation error = %v, want %q", err, test.want)
			}
		})
	}
}
