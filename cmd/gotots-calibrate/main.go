// Derives the calibration manifest mechanically from the seeds, the
// pinned corpus, and the baseline generation dump.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/tsoniclang/gotots/internal/calibration"
	"github.com/tsoniclang/gotots/internal/profile"
)

func main() {
	measure := flag.Bool("measure", false, "join the derived manifest to the authored hand ports and rewrite calibration/measurements.json")
	flag.Parse()
	if *measure {
		runMeasure()
		return
	}
	seeds, err := calibration.Load("calibration/seeds.json")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	prof, err := profile.Load("profiles/tsts/project.json")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if seeds.SourceRevision != prof.Pin.Revision {
		fmt.Fprintf(os.Stderr, "seeds sourceRevision %q disagrees with the pinned revision %q\n", seeds.SourceRevision, prof.Pin.Revision)
		os.Exit(1)
	}
	manifest, err := calibration.Derive(seeds,
		os.Getenv("GOTOTS_CORPUS_DIR"), ".", os.Getenv("GOTOTS_DUMP_DIR"),
		"implementing-agent", "jeswin (independent review pending)")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	data, _ := json.MarshalIndent(manifest, "", " ")
	if err := os.WriteFile("calibration/calibration-manifest.json", append(data, '\n'), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile("calibration/calibration-manifest.md", []byte(manifest.Render()), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("manifest: %d fixtures derived\n", len(manifest.Fixtures))
}

// runMeasure re-reads the persisted manifest (never re-deriving it) so
// measurement can run without the corpus or dump present.
func runMeasure() {
	data, err := os.ReadFile("calibration/calibration-manifest.json")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var manifest calibration.Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	report, err := calibration.Measure(&manifest, "calibration/handports")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := calibration.WriteMeasurements(report, "calibration/measurements.json"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("measurements: %d/%d authored, ordinary median %.2f, pending %v\n",
		report.Summary.FixturesAuthored, report.Summary.FixturesTotal,
		report.Summary.OrdinaryHandPortMedian, report.Summary.FixturesPending)
}
