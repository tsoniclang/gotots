// Package goenv constructs the hermetic environment used for every Go
// toolchain invocation. Ambient environment variables must not silently
// influence extraction, and no invocation may mutate the pinned source or
// fetch modules.
package goenv

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Resolved records the machine-specific toolchain locations discovered once
// from the ambient environment and then frozen for all child invocations.
// These values are evidence, not deterministic report content.
type Resolved struct {
	GoExecutable string `json:"goExecutable"`
	GOROOT       string `json:"goroot"`
	GOCACHE      string `json:"gocache"`
	GOMODCACHE   string `json:"gomodcache"`
	GOPATH       string `json:"gopath"`
}

// Resolve locates the go executable and its cache/root directories using the
// ambient environment exactly once.
func Resolve() (*Resolved, error) {
	goExecutable, err := exec.LookPath("go")
	if err != nil {
		return nil, fmt.Errorf("locate go executable: %w", err)
	}
	command := exec.Command(goExecutable, "env", "-json", "GOROOT", "GOCACHE", "GOMODCACHE", "GOPATH")
	var out bytes.Buffer
	command.Stdout = &out
	command.Stderr = &out
	if err := command.Run(); err != nil {
		return nil, fmt.Errorf("go env: %v: %s", err, out.String())
	}
	var values struct {
		GOROOT     string
		GOCACHE    string
		GOMODCACHE string
		GOPATH     string
	}
	if err := json.Unmarshal(out.Bytes(), &values); err != nil {
		return nil, fmt.Errorf("parse go env output: %w", err)
	}
	return &Resolved{
		GoExecutable: goExecutable,
		GOROOT:       values.GOROOT,
		GOCACHE:      values.GOCACHE,
		GOMODCACHE:   values.GOMODCACHE,
		GOPATH:       values.GOPATH,
	}, nil
}

// Environ builds the complete, closed child environment for one build
// profile. Nothing from the ambient environment leaks through except the
// values resolved and frozen in r.
//
// The policy is fail-closed and read-only:
//   - GOFLAGS=-mod=readonly: the loader may never rewrite go.mod/go.sum;
//   - GOPROXY=off and GONOSUMDB unset with GOSUMDB=off: no network;
//   - GOWORK=off, GOENV=off: no workspace or user configuration input;
//   - GOTOOLCHAIN=local: the pinned local toolchain only, no auto-switch.
func (r *Resolved) Environ(goos, goarch string, cgoEnabled bool) []string {
	cgo := "0"
	if cgoEnabled {
		cgo = "1"
	}
	env := []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
		"GOROOT=" + r.GOROOT,
		"GOCACHE=" + r.GOCACHE,
		"GOMODCACHE=" + r.GOMODCACHE,
		"GOPATH=" + r.GOPATH,
		"GOOS=" + goos,
		"GOARCH=" + goarch,
		"CGO_ENABLED=" + cgo,
		"GOFLAGS=-mod=readonly",
		"GOPROXY=off",
		"GOSUMDB=off",
		"GOWORK=off",
		"GOENV=off",
		"GOTOOLCHAIN=local",
		"LANG=C",
		"LC_ALL=C",
	}
	if tmp := os.Getenv("TMPDIR"); tmp != "" {
		env = append(env, "TMPDIR="+tmp)
	}
	return env
}

// Run executes the go tool with the hermetic environment in dir and returns
// stdout. Stderr is captured for error reporting only.
func (r *Resolved) Run(dir string, env []string, args ...string) ([]byte, error) {
	command := exec.Command(r.GoExecutable, args...)
	command.Dir = dir
	command.Env = env
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return nil, fmt.Errorf("go %s: %v: %s", strings.Join(args, " "), err, stderr.String())
	}
	return stdout.Bytes(), nil
}
