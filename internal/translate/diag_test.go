package translate_test

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/contracts"
	"github.com/tsoniclang/gotots/internal/goenv"
	"github.com/tsoniclang/gotots/internal/pinning"
	"github.com/tsoniclang/gotots/internal/profile"
	"github.com/tsoniclang/gotots/internal/translate"
)

func TestDiagBlockers(t *testing.T) {
	sourceDir := os.Getenv("GOTOTS_CORPUS_DIR")
	if sourceDir == "" {
		t.Skip("set GOTOTS_CORPUS_DIR")
	}
	prof, _ := profile.Load("../../profiles/tsts/project.json")
	build, _ := prof.BuildProfileByName("linux-amd64")
	resolved, err := pinning.VerifyToolchain(prof.Pin)
	if err != nil {
		t.Fatal(err)
	}
	env := resolved.Environ(goenv.EnvOptions{GOOS: build.GOOS, GOARCH: build.GOARCH, GOAMD64: build.GOAMD64, GOARM64: build.GOARM64})
	g, err := translate.Corpus(prof, env, sourceDir, translate.Options{SourceRevision: "diag", ProfileHash: "diag"})
	if err != nil {
		t.Fatal(err)
	}
	roots := map[string]string{}
	for pkg, reason := range g.NotMaterialized {
		roots[pkg] = reason
	}
	var lines []string
	for pkg, reason := range roots {
		short := pkg[strings.LastIndex(pkg, "typescript-go/")+len("typescript-go/"):]
		lines = append(lines, short+" :: "+reason)
	}
	sort.Strings(lines)
	for _, l := range lines {
		fmt.Println("NM " + l)
	}
	for _, s := range g.Support {
		if s.State != "unimplemented" {
			continue
		}
		parts := strings.SplitN(s.ID, "::", 3)
		if len(parts) >= 2 && parts[1] == "type" && strings.Contains(roots[s.Package], "declaration blockers") {
			for _, site := range s.Sites {
				fmt.Printf("TYPE %s :: %.240s @ %s:%d\n", s.ID[strings.LastIndex(s.ID, "internal/")+9:], site.Construct, site.Span.File, site.Span.Line)
			}
		}
	}
}

// TestDiagBodySites enumerates the remaining unsupported BODY sites by
// construct class (the A5 long tail): run with GOTOTS_CORPUS_DIR.
func TestDiagBodySites(t *testing.T) {
	sourceDir := os.Getenv("GOTOTS_CORPUS_DIR")
	if sourceDir == "" {
		t.Skip("set GOTOTS_CORPUS_DIR")
	}
	prof, err := profile.Load("../../profiles/tsts/project.json")
	if err != nil {
		t.Fatal(err)
	}
	build, _ := prof.BuildProfileByName("linux-amd64")
	resolved, err := pinning.VerifyToolchain(prof.Pin)
	if err != nil {
		t.Fatal(err)
	}
	env := resolved.Environ(goenv.EnvOptions{GOOS: build.GOOS, GOARCH: build.GOARCH, GOAMD64: build.GOAMD64, GOARM64: build.GOARM64})
	g, err := translate.Corpus(prof, env, sourceDir, translate.Options{SourceRevision: "diag", ProfileHash: "diag"})
	if err != nil {
		t.Fatal(err)
	}
	type classInfo struct {
		count   int
		samples []string
	}
	classes := map[string]*classInfo{}
	total := 0
	for _, s := range g.Support {
		if s.State != "unimplemented" {
			continue
		}
		for _, site := range s.Sites {
			total++
			key := site.Construct
			if len(key) > 90 {
				key = key[:90]
			}
			info := classes[key]
			if info == nil {
				info = &classInfo{}
				classes[key] = info
			}
			info.count++
			if len(info.samples) < 2 {
				info.samples = append(info.samples, fmt.Sprintf("%s:%d", site.Span.File, site.Span.Line))
			}
		}
	}
	type kv struct {
		k string
		v *classInfo
	}
	var list []kv
	for k, v := range classes {
		list = append(list, kv{k, v})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].v.count > list[j].v.count })
	fmt.Printf("TOTAL unsupported sites: %d in %d classes\n", total, len(list))
	for _, e := range list {
		fmt.Printf("%5d  %s   [%s]\n", e.v.count, e.k, strings.Join(e.v.samples, " "))
	}
}

// TestDiagDispositionLedger prints every remaining unsupported site with
// its §C routing class (run with GOTOTS_CORPUS_DIR; output feeds the
// manual-completion contract).
func TestDiagDispositionLedger(t *testing.T) {
	sourceDir := os.Getenv("GOTOTS_CORPUS_DIR")
	if sourceDir == "" {
		t.Skip("set GOTOTS_CORPUS_DIR")
	}
	prof, err := profile.Load("../../profiles/tsts/project.json")
	if err != nil {
		t.Fatal(err)
	}
	build, _ := prof.BuildProfileByName("linux-amd64")
	resolved, err := pinning.VerifyToolchain(prof.Pin)
	if err != nil {
		t.Fatal(err)
	}
	env := resolved.Environ(goenv.EnvOptions{GOOS: build.GOOS, GOARCH: build.GOARCH, GOAMD64: build.GOAMD64, GOARM64: build.GOARM64})
	g, err := translate.Corpus(prof, env, sourceDir, translate.Options{SourceRevision: "diag", ProfileHash: "diag"})
	if err != nil {
		t.Fatal(err)
	}
	route := func(construct, file string) string {
		switch {
		case strings.Contains(construct, "chan") || strings.Contains(construct, "goroutine") ||
			strings.Contains(construct, "select") || strings.Contains(construct, "close") ||
			strings.Contains(construct, "channel send"):
			return "policy-concurrency"
		case strings.Contains(construct, "unsafe"):
			return "out-of-scope-unsafe"
		case strings.Contains(construct, "reflect"):
			return "out-of-scope-reflection"
		case strings.Contains(file, "fswatch"):
			return "policy-platform"
		case strings.Contains(construct, "type-parameter boxing") ||
			strings.Contains(construct, "runtime type identity of") ||
			strings.Contains(construct, "address of T") ||
			strings.Contains(construct, "compound assignment on T") ||
			strings.Contains(construct, "conversion from T") ||
			strings.Contains(construct, "conversion from float64 to T") ||
			strings.Contains(construct, "capturing type parameters") ||
			strings.Contains(construct, "typeid: no exact"):
			return "deferred-protocol"
		}
		return "manual-candidate"
	}
	counts := map[string]int{}
	for _, s := range g.Support {
		if s.State != "unimplemented" {
			continue
		}
		for _, site := range s.Sites {
			r := route(site.Construct, site.Span.File)
			counts[r]++
			fmt.Printf("LEDGER %-24s %s :: %.100s @ %s:%d\n", r, s.ID[strings.LastIndex(s.ID, "internal/")+9:], site.Construct, site.Span.File, site.Span.Line)
		}
	}
	fmt.Println("LEDGER-COUNTS:", counts)
}

func TestDiagUnimplementedBodies(t *testing.T) {
	sourceDir := os.Getenv("GOTOTS_CORPUS_DIR")
	if sourceDir == "" {
		t.Skip("set GOTOTS_CORPUS_DIR")
	}
	prof, err := profile.Load("../../profiles/tsts/project.json")
	if err != nil {
		t.Fatal(err)
	}
	build, _ := prof.BuildProfileByName("linux-amd64")
	resolved, err := pinning.VerifyToolchain(prof.Pin)
	if err != nil {
		t.Fatal(err)
	}
	env := resolved.Environ(goenv.EnvOptions{GOOS: build.GOOS, GOARCH: build.GOARCH, GOAMD64: build.GOAMD64, GOARM64: build.GOARM64})
	g, err := translate.Corpus(prof, env, sourceDir, translate.Options{SourceRevision: "diag", ProfileHash: "diag"})
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, s := range g.Support {
		if s.State != "unimplemented" {
			continue
		}
		count++
		constructs := map[string]bool{}
		for _, site := range s.Sites {
			constructs[site.Construct] = true
		}
		keys := make([]string, 0, len(constructs))
		for k := range constructs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		fmt.Printf("BODY\t%s\t%s\n", s.ID, strings.Join(keys, " | "))
	}
	fmt.Printf("UNIMPLEMENTED BODIES: %d\n", count)
}

// TestManualCompletionContract verifies the §C contract against the
// live corpus: every unimplemented body has exactly one reviewed
// disposition with its exact blocker set (drift fails), no entry is
// stale.
func TestManualCompletionContract(t *testing.T) {
	sourceDir := os.Getenv("GOTOTS_CORPUS_DIR")
	if sourceDir == "" {
		t.Skip("set GOTOTS_CORPUS_DIR")
	}
	prof, err := profile.Load("../../profiles/tsts/project.json")
	if err != nil {
		t.Fatal(err)
	}
	build, _ := prof.BuildProfileByName("linux-amd64")
	resolved, err := pinning.VerifyToolchain(prof.Pin)
	if err != nil {
		t.Fatal(err)
	}
	env := resolved.Environ(goenv.EnvOptions{GOOS: build.GOOS, GOARCH: build.GOARCH, GOAMD64: build.GOAMD64, GOARM64: build.GOARM64})
	g, err := translate.Corpus(prof, env, sourceDir, translate.Options{SourceRevision: "diag", ProfileHash: "diag"})
	if err != nil {
		t.Fatal(err)
	}
	completions, err := contracts.LoadManualCompletions()
	if err != nil {
		t.Fatal(err)
	}
	unimplemented := map[string][]string{}
	for _, s := range g.Support {
		if s.State != "unimplemented" {
			continue
		}
		set := map[string]bool{}
		for _, site := range s.Sites {
			set[site.Construct] = true
		}
		var constructs []string
		for construct := range set {
			constructs = append(constructs, construct)
		}
		sort.Strings(constructs)
		unimplemented[s.ID] = constructs
	}
	if err := completions.VerifyDispositions(unimplemented); err != nil {
		t.Fatal(err)
	}
}
