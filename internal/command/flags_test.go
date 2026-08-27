package command

import (
	"flag"
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/config"
)

func TestParseArgumentsSelectsOnlyExactDefaultConfig(t *testing.T) {
	workingDirectory := filepath.Join(t.TempDir(), "child")
	invocation, err := ParseArguments(workingDirectory, []string{"build"})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(workingDirectory, "gotots.json")
	if invocation.ConfigPath() != want {
		t.Fatalf("config path = %q, want %q", invocation.ConfigPath(), want)
	}
}

func TestParseArgumentsSupportsBothConfigAliases(t *testing.T) {
	for _, flag := range []string{"-c", "--config"} {
		t.Run(flag, func(t *testing.T) {
			workingDirectory := t.TempDir()
			invocation, err := ParseArguments(
				workingDirectory,
				[]string{"build", flag, "project/gotots.json"},
			)
			if err != nil {
				t.Fatal(err)
			}
			want := filepath.Join(workingDirectory, "project", "gotots.json")
			if invocation.ConfigPath() != want {
				t.Fatalf("config path = %q, want %q", invocation.ConfigPath(), want)
			}
		})
	}
}

func TestParseArgumentsBindsSemanticAndRepeatedOverrides(t *testing.T) {
	invocation, err := ParseArguments(t.TempDir(), []string{
		"build",
		"--integer", "fixed64-bigint",
		"--tag", "noasm",
		"--tag", "purego",
		"--package-implementation", "package-one.json",
		"--package-implementation", "package-two.json",
		"--callable-implementation", "callable-one.json",
		"--callable-implementation", "callable-two.json",
		"--go", "/tools/go-selected",
		"--tsgo", "/tools/tsgo-selected",
		"--tool-cache", "/project/.temp/cache/tools",
		"--standard-library=false",
		"--externals",
		"--print-resolved-config",
	})
	if err != nil {
		t.Fatal(err)
	}
	overrides := invocation.Overrides()
	if overrides.IntegerRepresentation == nil ||
		*overrides.IntegerRepresentation != "fixed64-bigint" ||
		!overrides.BuildTagsSet || len(overrides.BuildTags) != 2 ||
		!overrides.PackageImplementationsSet || len(overrides.PackageImplementations) != 2 ||
		!overrides.CallableImplementationsSet || len(overrides.CallableImplementations) != 2 ||
		overrides.StandardLibrary == nil || *overrides.StandardLibrary ||
		overrides.Externals == nil || !*overrides.Externals ||
		overrides.GoExecutable == nil || *overrides.GoExecutable != "/tools/go-selected" ||
		overrides.TSGoExecutable == nil || *overrides.TSGoExecutable != "/tools/tsgo-selected" ||
		overrides.ToolCacheRoot == nil || *overrides.ToolCacheRoot != "/project/.temp/cache/tools" ||
		!invocation.PrintResolvedConfig() {
		t.Fatalf("parsed invocation = %#v", invocation)
	}
}

func TestParseArgumentsRejectsAmbiguousConfigSelection(t *testing.T) {
	if _, err := ParseArguments(t.TempDir(), []string{
		"build", "-c", "one.json", "--config", "two.json",
	}); err == nil {
		t.Fatal("ambiguous config selection was accepted")
	}
}

func TestEveryRegisteredOptionHasACommandBinding(t *testing.T) {
	flags := flag.NewFlagSet("test", flag.ContinueOnError)
	var overrides config.Overrides
	for _, descriptor := range config.Descriptors() {
		if err := bindDescriptor(flags, descriptor, &overrides); err != nil {
			t.Fatal(err)
		}
		if flags.Lookup(descriptor.Flag()) == nil {
			t.Fatalf("%s has no registered command flag", descriptor.JSONPath())
		}
	}
}
