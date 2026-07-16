// The probe/corpus classification join: both walks classify every
// production body by the same canonical identity, and their generated /
// unimplemented verdicts must agree one-to-one — no body may be
// explained away as "different tools". Requires the pinned corpus
// checkout, so it runs when GOTOTS_CORPUS_DIR is set (the gate's
// environment always sets it).
package translate_test

import (
	"os"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/goenv"
	"github.com/tsoniclang/gotots/internal/pinning"
	"github.com/tsoniclang/gotots/internal/profile"
	"github.com/tsoniclang/gotots/internal/translate"
)

func TestProbeCorpusClassificationsJoinExactly(t *testing.T) {
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
	generated, err := translate.Corpus(prof, env, sourceDir, translate.Options{SourceRevision: "join", ProfileHash: "join"})
	if err != nil {
		t.Fatal(err)
	}
	corpusState := map[string]string{}
	for _, support := range generated.Support {
		corpusState[support.ID] = string(support.State)
	}
	joined := 0
	for id, probeState := range probe.PerBodyState {
		corpus, has := corpusState[id]
		if !has {
			t.Errorf("probe body %s has no corpus support record", id)
			continue
		}
		if (probeState == "generated") != (corpus == "generated") {
			t.Errorf("classification disagreement at %s: probe=%s corpus=%s", id, probeState, corpus)
			continue
		}
		joined++
	}
	// The REVERSE join: every corpus body-support record (functions and
	// methods — identities the probe walks) must appear in the probe's
	// ledger; a corpus-only body identity is a walk divergence.
	reverse := 0
	for id, state := range corpusState {
		if _, has := probe.PerBodyState[id]; has {
			continue
		}
		if strings.Contains(id, "::func::") || strings.Contains(id, "::method::") {
			t.Errorf("corpus body %s (state %s) missing from the probe ledger", id, state)
			continue
		}
		reverse++ // declaration-level units (types, vars) — not probe-walked
	}
	if joined != probe.Bodies {
		t.Errorf("joined %d of %d probe bodies", joined, probe.Bodies)
	}
	t.Logf("exact join: %d bodies, probe %d ir-admitted / %d blocked", joined, probe.IRAdmitted, probe.Blocked)
}
