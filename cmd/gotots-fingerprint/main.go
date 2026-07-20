// Fingerprint runner: the permanent reproducible producer of
// fingerprints.json (numbered-order step 4). Every input and output
// class carries its sorted per-file path/hash manifest; every class
// exists even when empty; every walked file belongs to exactly one
// class; semantic identities are separate from machine environment.
//
// Usage:
//
//	GOTOTS_DUMP_DIR=<generated tree> GOTOTS_CORPUS_DIR=<pinned tsgo> \
//	  gotots-fingerprint > fingerprints.json
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"

	"github.com/tsoniclang/gotots/internal/fingerprint"
	"github.com/tsoniclang/gotots/internal/profile"
)

// dumpClasses is the complete generated-output class universe. Source
// maps and ledger reports are declared even while empty: absence and
// emptiness are different facts.
var dumpClasses = []string{
	"generated-core-modules",
	"generated-external-contracts",
	"generated-abi-modules",
	"analysis-body-artifacts",
	"generated-source-maps",
	"ledgers-reports",
}

var dumpRules = []fingerprint.PrefixRule{
	{Prefix: "core", Class: "generated-core-modules"},
	{Prefix: "external-stubs", Class: "generated-external-contracts"},
	{Prefix: "language-abi", Class: "generated-abi-modules"},
	{Prefix: "analysis", Class: "analysis-body-artifacts"},
	{Prefix: "sourcemaps", Class: "generated-source-maps"},
	{Prefix: "ledgers", Class: "ledgers-reports"},
}

var corpusClasses = []string{
	"selected-source",
	"test-only-source",
	"outside-universe-source",
	"tooling-source",
	"policy-excluded-source",
	"corpus-nonsource",
}

func corpusClassifier(prof *profile.Profile) fingerprint.Classifier {
	return func(filePath string) (string, bool) {
		if !strings.HasSuffix(filePath, ".go") || strings.Contains(filePath, "/testdata/") {
			return "corpus-nonsource", true
		}
		pkg := path.Dir(filePath)
		if pkg == "." {
			return "corpus-nonsource", true
		}
		rule, err := prof.SourceUniverse.Classify(pkg)
		if err != nil {
			// Total classification is Gate 03's contract over PACKAGES;
			// stray .go files outside any classified package (module
			// tooling at the root) are non-source for fingerprinting.
			return "corpus-nonsource", true
		}
		switch rule.Disposition {
		case profile.DispositionSelected:
			return "selected-source", true
		case profile.DispositionTestOnly:
			return "test-only-source", true
		case profile.DispositionOutside:
			return "outside-universe-source", true
		case profile.DispositionTooling:
			return "tooling-source", true
		case profile.DispositionPolicyExcluded:
			return "policy-excluded-source", true
		}
		return "", false
	}
}

func fileDigest(filePath string) string {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "absent"
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func main() {
	dumpDir := os.Getenv("GOTOTS_DUMP_DIR")
	corpusDir := os.Getenv("GOTOTS_CORPUS_DIR")
	if dumpDir == "" || corpusDir == "" {
		fmt.Fprintln(os.Stderr, "set GOTOTS_DUMP_DIR and GOTOTS_CORPUS_DIR")
		os.Exit(1)
	}
	prof, err := profile.Load("profiles/tsts/project.json")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	skip := func(dir string) bool {
		base := path.Base(dir)
		return base == ".git" || base == "node_modules" || base == ".temp" || base == ".tests"
	}

	semantic := map[string]string{
		"pinned-source-revision":   prof.Pin.Revision,
		"profile-file-sha256":      fileDigest("profiles/tsts/project.json"),
		"profile-schema-sha256":    fileDigest("schemas/project-profile.schema.json"),
		"spec-manifest-sha256":     fileDigest("docs/spec/manifest.json"),
		"decision-registry-sha256": fileDigest("docs/decisions/registry.json"),
		"toolchain-pin-sha256":     fileDigest("pins/product-toolchain.json"),
		"go-version":               runtime.Version(),
	}
	if head, err := exec.Command("git", "rev-parse", "HEAD").Output(); err == nil {
		semantic["gotots-revision"] = strings.TrimSpace(string(head))
	}
	hostname, _ := os.Hostname()
	environment := map[string]string{
		"hostname":    hostname,
		"dump-root":   dumpDir,
		"corpus-root": corpusDir,
	}

	dumpReport, err := fingerprint.Build(dumpDir, dumpClasses, fingerprint.PrefixClassifier(dumpRules), skip, nil, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dump fingerprint: %v\n", err)
		os.Exit(1)
	}
	corpusReport, err := fingerprint.Build(corpusDir, corpusClasses, corpusClassifier(prof), skip, nil, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "corpus fingerprint: %v\n", err)
		os.Exit(1)
	}

	combined := &fingerprint.Report{
		SchemaVersion: fingerprint.SchemaVersion,
		Semantic:      semantic,
		Environment:   environment,
	}
	combined.Classes = append(combined.Classes, corpusReport.Classes...)
	combined.Classes = append(combined.Classes, dumpReport.Classes...)

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", " ")
	if err := encoder.Encode(combined); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, class := range combined.Classes {
		fmt.Fprintf(os.Stderr, "%s: %d files %s\n", class.Name, len(class.Files), class.Sha256[:16])
	}
}
