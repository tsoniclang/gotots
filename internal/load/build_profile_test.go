package load

import (
	"context"
	"go/types"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
)

func TestLoadUsesResolvedBuildProfileInsteadOfAmbientShell(t *testing.T) {
	project := t.TempDir()
	writeBuildProfileFixture(t, project)
	profile, err := NewBuildProfile(
		runtime.GOOS,
		runtime.GOARCH,
		false,
		[]string{"selectedprofile", "zeta"},
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOOS", "plan9")
	t.Setenv("GOARCH", "386")
	t.Setenv("CGO_ENABLED", "1")
	t.Setenv("GOFLAGS", "-tags=ambientprofile")

	program, err := Load(context.Background(), Request{
		Directory:    project,
		Pattern:      ".",
		BuildProfile: profile,
	})
	if err != nil {
		t.Fatal(err)
	}
	selected := program.BuildProfile()
	if selected.ToolchainVersion() != runtime.Version() ||
		selected.GOOS() != runtime.GOOS ||
		selected.GOARCH() != runtime.GOARCH ||
		selected.CgoEnabled() ||
		!slices.Equal(selected.Tags(), []string{"selectedprofile", "zeta"}) {
		t.Fatalf("resolved profile = %#v", selected)
	}
	root := program.Roots()[0]
	variant, ok := root.Types().Scope().Lookup("Selected").(*types.Const)
	if !ok || variant.Val().ExactString() != `"selected"` {
		t.Fatalf("selected variant = %#v", variant)
	}
	if root.Types().Scope().Lookup("Ambient") != nil {
		t.Fatal("ambient GOFLAGS build tag changed the selected source universe")
	}
	tags := selected.Tags()
	tags[0] = "mutated"
	if slices.Equal(tags, program.BuildProfile().Tags()) {
		t.Fatal("build-profile tags expose mutable backing storage")
	}
}

func TestLoadResolvesZeroProfileToExactHostProfile(t *testing.T) {
	project := t.TempDir()
	writeBuildProfileFixture(t, project)
	t.Setenv("GOOS", "plan9")
	t.Setenv("GOARCH", "386")
	t.Setenv("CGO_ENABLED", "1")
	t.Setenv("GOFLAGS", "-tags=ambientprofile")

	program, err := Load(context.Background(), Request{
		Directory: project,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	selected := program.BuildProfile()
	if selected.ToolchainVersion() != runtime.Version() ||
		selected.GOOS() != runtime.GOOS ||
		selected.GOARCH() != runtime.GOARCH ||
		selected.CgoEnabled() ||
		len(selected.Tags()) != 0 {
		t.Fatalf("default profile = %#v", selected)
	}
	if program.Roots()[0].Types().Scope().Lookup("Ambient") != nil {
		t.Fatal("default profile inherited ambient GOFLAGS")
	}
}

func TestBuildProfileRejectsNonCanonicalTags(t *testing.T) {
	for _, testCase := range []struct {
		name string
		tags []string
	}{
		{name: "empty", tags: []string{""}},
		{name: "duplicate", tags: []string{"same", "same"}},
		{name: "comma", tags: []string{"left,right"}},
		{name: "space", tags: []string{"not valid"}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := NewBuildProfile(
				runtime.GOOS,
				runtime.GOARCH,
				false,
				testCase.tags,
			); err == nil {
				t.Fatal("invalid build profile was accepted")
			}
		})
	}
}

func writeBuildProfileFixture(t *testing.T, directory string) {
	t.Helper()
	files := map[string]string{
		"go.mod": "module example.com/buildprofile\n\ngo 1.26.4\n",
		"default.go": `//go:build !selectedprofile

package buildprofile

const Selected = "default"
`,
		"selected.go": `//go:build selectedprofile

package buildprofile

const Selected = "selected"
`,
		"ambient.go": `//go:build ambientprofile

package buildprofile

const Ambient = true
`,
	}
	for name, content := range files {
		if err := os.WriteFile(
			filepath.Join(directory, name),
			[]byte(content),
			0o644,
		); err != nil {
			t.Fatal(err)
		}
	}
}
