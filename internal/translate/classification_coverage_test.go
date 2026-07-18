// Every unsupported site in the real corpus classifies into a disposition
// category — zero unclassified. The residual inventory is a total,
// closed classification; a newly added rejection reason with no reviewed
// disposition surfaces here as "unclassified" and fails. Requires the
// pinned corpus checkout, so it runs when GOTOTS_CORPUS_DIR is set (the
// gate's environment always sets it).
package translate_test

import (
	"os"
	"testing"

	"github.com/tsoniclang/gotots/internal/goenv"
	"github.com/tsoniclang/gotots/internal/pinning"
	"github.com/tsoniclang/gotots/internal/profile"
	"github.com/tsoniclang/gotots/internal/translate"
)

func TestCorpusHasZeroUnclassifiedSites(t *testing.T) {
	sourceDir := os.Getenv("GOTOTS_CORPUS_DIR")
	if sourceDir == "" {
		t.Skip("set GOTOTS_CORPUS_DIR to the pinned corpus checkout")
	}
	prof, err := profile.Load("../../profiles/tsts/project.json")
	if err != nil {
		t.Fatal(err)
	}
	build, err := prof.BuildProfileByName("linux-amd64")
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := pinning.VerifyToolchain(prof.Pin)
	if err != nil {
		t.Fatal(err)
	}
	env := resolved.Environ(goenv.EnvOptions{GOOS: build.GOOS, GOARCH: build.GOARCH, GOAMD64: build.GOAMD64, GOARM64: build.GOARM64})
	probe, err := translate.Probe(prof, env, sourceDir)
	if err != nil {
		t.Fatal(err)
	}
	unclassified := map[string]int{}
	for _, u := range probe.UnimplementedUnits {
		for _, s := range u.Sites {
			if s.Category == "unclassified" {
				unclassified[s.Class]++
			}
		}
	}
	if len(unclassified) > 0 {
		t.Errorf("%d unsupported classes have no reviewed disposition: %v", len(unclassified), unclassified)
	}
}
