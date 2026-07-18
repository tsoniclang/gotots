package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/tsoniclang/gotots/internal/goenv"
	"github.com/tsoniclang/gotots/internal/pinning"
	"github.com/tsoniclang/gotots/internal/profile"
	"github.com/tsoniclang/gotots/internal/translate"
)

// runTranslateProbe measures the current reviewed subset against the real
// pinned corpus: how many bodies translate today and exactly which
// semantic families block the rest, ranked. Diagnostic evidence for
// roadmap ordering, not authoritative generation.
func runTranslateProbe(args []string) error {
	flags := flag.NewFlagSet("translate-probe", flag.ExitOnError)
	profilePath := flags.String("profile", "profiles/tsts/project.json", "project profile path")
	sourceDir := flags.String("source", "", "pinned source checkout (required)")
	buildProfile := flags.String("build-profile", "linux-amd64", "build profile name")
	reportPath := flags.String("report", "", "probe report output path (required)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *sourceDir == "" || *reportPath == "" {
		return fmt.Errorf("--source and --report are required")
	}

	prof, err := profile.Load(*profilePath)
	if err != nil {
		return err
	}
	build, err := prof.BuildProfileByName(*buildProfile)
	if err != nil {
		return err
	}
	resolved, err := pinning.VerifyToolchain(prof.Pin)
	if err != nil {
		return err
	}
	env := resolved.Environ(goenv.EnvOptions{
		GOOS: build.GOOS, GOARCH: build.GOARCH,
		GOAMD64: build.GOAMD64, GOARM64: build.GOARM64,
	})

	result, err := translate.Probe(prof, env, *sourceDir)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(*reportPath, append(data, '\n'), 0o644); err != nil {
		return err
	}

	fmt.Printf("bodies %d ir-admitted %d (%.1f%%) blocked %d across %d packages\n",
		result.Bodies, result.IRAdmitted,
		100*float64(result.IRAdmitted)/float64(result.Bodies), result.Blocked, result.Packages)
	fmt.Println("NOTE: ir-admitted is not module-retained coverage; package withholding removes bodies from runnable output (see gate census).")
	fmt.Printf("IR/declaration-complete candidate packages (analysis only, deferred emitters NOT executed): %d %v\n", len(result.PackagesIRDeclComplete), result.PackagesIRDeclComplete)
	type entry struct {
		key   string
		count int
	}
	var ranked []entry
	for key, count := range result.BlockerHistogram {
		ranked = append(ranked, entry{key, count})
	}
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].count > ranked[j].count })
	fmt.Println("top blockers:")
	for i, e := range ranked {
		if i >= 25 {
			break
		}
		fmt.Printf("  %6d  %s\n", e.count, e.key)
	}
	return nil
}
