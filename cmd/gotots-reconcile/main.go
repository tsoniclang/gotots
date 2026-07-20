// Reconciliation runner: census + generation → the canonical
// identity-join report (JSON to stdout).
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/tsoniclang/gotots/internal/census"
	"github.com/tsoniclang/gotots/internal/goenv"
	"github.com/tsoniclang/gotots/internal/pinning"
	"github.com/tsoniclang/gotots/internal/profile"
	"github.com/tsoniclang/gotots/internal/reconcile"
	"github.com/tsoniclang/gotots/internal/translate"
)

func main() {
	sourceDir := os.Getenv("GOTOTS_CORPUS_DIR")
	head := os.Getenv("GOTOTS_HEAD")
	if sourceDir == "" || head == "" {
		fmt.Fprintln(os.Stderr, "set GOTOTS_CORPUS_DIR and GOTOTS_HEAD")
		os.Exit(1)
	}
	prof, err := profile.Load("profiles/tsts/project.json")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	build, err := prof.BuildProfileByName("linux-amd64")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	resolved, err := pinning.VerifyToolchain(prof.Pin)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	env := resolved.Environ(goenv.EnvOptions{GOOS: build.GOOS, GOARCH: build.GOARCH, GOAMD64: build.GOAMD64, GOARM64: build.GOARM64})
	run, err := census.Run(prof, sourceDir, "linux-amd64")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	generated, err := translate.Corpus(prof, env, sourceDir, translate.Options{SourceRevision: head, ProfileHash: "reconcile"})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var history *reconcile.History
	if priorPath := os.Getenv("GOTOTS_PRIOR_BODIES"); priorPath != "" {
		data, err := os.ReadFile(priorPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		var priorBodies []string
		for _, line := range strings.Split(string(data), "\n") {
			if line != "" {
				priorBodies = append(priorBodies, line)
			}
		}
		history = &reconcile.History{
			Source:          priorPath,
			PriorBodies:     priorBodies,
			PriorBodiesFrom: os.Getenv("GOTOTS_PRIOR_BODIES_FROM"),
		}
	}
	if pubPath := os.Getenv("GOTOTS_PRIOR_PUBLISHED"); pubPath != "" && history != nil {
		data, err := os.ReadFile(pubPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		for _, line := range strings.Split(string(data), "\n") {
			if line != "" {
				history.PriorPublished = append(history.PriorPublished, line)
			}
		}
		history.PriorPublishedAt = os.Getenv("GOTOTS_PRIOR_BODIES_FROM")
	}
	inputs := map[string]string{}
	for name, env := range map[string]string{"prior-bodies": "GOTOTS_PRIOR_BODIES", "prior-published": "GOTOTS_PRIOR_PUBLISHED"} {
		if path := os.Getenv(env); path != "" {
			data, _ := os.ReadFile(path)
			digest := sha256.Sum256(data)
			inputs[name] = hex.EncodeToString(digest[:])
		}
	}
	report := reconcile.BuildWithHistory(head, run, generated, history, inputs)
	if mdPath := os.Getenv("GOTOTS_RENDER_MD"); mdPath != "" {
		if err := os.WriteFile(mdPath, []byte(report.Render()), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", " ")
	if err := encoder.Encode(report); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
