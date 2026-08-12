package certify

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestCanonicalValueSlotsFollowProviderParameterOrder(t *testing.T) {
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
		"src/internal/facets/provider-compress-gzip.ts",
	))
	if err != nil {
		t.Fatal(err)
	}
	var target tsgo.ProjectExport
	for _, selected := range exports {
		if selected.Name() == "GzipNewReaderCanonical" {
			target = selected
			break
		}
	}
	if target.Name() == "" {
		t.Fatal("GzipNewReaderCanonical is absent")
	}
	values := []gostdlib.ProviderCallableProfileCanonicalValueDocument{
		{SourceIdentity: "io.EOF", TargetParameter: "eof"},
		{SourceIdentity: "io.ErrUnexpectedEOF", TargetParameter: "unexpectedEOF"},
		{SourceIdentity: "io.ErrNoProgress", TargetParameter: "noProgress"},
	}
	if err := validateCanonicalProfileValueParameters(
		project,
		target,
		1,
		values,
	); err != nil {
		t.Fatal(err)
	}
	values[1], values[2] = values[2], values[1]
	err = validateCanonicalProfileValueParameters(project, target, 1, values)
	if err == nil || !strings.Contains(err.Error(), "certified slot") {
		t.Fatalf("canonical-value order mutation error = %v", err)
	}
}
