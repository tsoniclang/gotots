// Package goenv constructs the hermetic environment used for every Go
// toolchain invocation. Ambient environment variables must not silently
// influence extraction, and no invocation may mutate the pinned source or
// fetch modules.
//
// Trust ordering matters: Locate finds the candidate go executable without
// running it, so callers can verify its digest against a pin before any
// invocation. Only Bootstrap executes the binary.
package goenv

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Locate finds the candidate go executable on PATH without executing it.
func Locate() (string, error) {
	goExecutable, err := exec.LookPath("go")
	if err != nil {
		return "", fmt.Errorf("locate go executable: %w", err)
	}
	return goExecutable, nil
}

// Resolved records the machine-specific toolchain locations discovered once
// from a verified executable and then frozen for all child invocations.
// These values are evidence, not deterministic report content.
type Resolved struct {
	GoExecutable string `json:"goExecutable"`
	GOROOT       string `json:"goroot"`
	GOCACHE      string `json:"gocache"`
	GOMODCACHE   string `json:"gomodcache"`
	GOPATH       string `json:"gopath"`
}

// BootstrapEnviron is the environment for interrogating the go binary
// itself: ambient toolchain switching and workspace state are disabled so
// the verified binary cannot re-exec a different toolchain, while the
// user's cache/path configuration is still honored (the resolved values are
// recorded as machine evidence).
func BootstrapEnviron() []string {
	env := []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
		"GOTOOLCHAIN=local",
		"GOWORK=off",
	}
	if tmp := os.Getenv("TMPDIR"); tmp != "" {
		env = append(env, "TMPDIR="+tmp)
	}
	return env
}

// Bootstrap executes the given go binary to resolve its root and cache
// locations. Callers must have verified the executable's identity first
// when a pin is available; the unverified path exists only for the
// toolchain-id bootstrap command.
func Bootstrap(goExecutable string) (*Resolved, error) {
	command := exec.Command(goExecutable, "env", "-json", "GOROOT", "GOCACHE", "GOMODCACHE", "GOPATH")
	command.Env = BootstrapEnviron()
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
	decoder := json.NewDecoder(bytes.NewReader(out.Bytes()))
	if err := decoder.Decode(&values); err != nil {
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

// EnvOptions are the complete source-selection inputs for one build
// profile. Every field is explicit; there are no ambient defaults.
type EnvOptions struct {
	GOOS       string
	GOARCH     string
	GOAMD64    string // required when GOARCH is amd64
	CgoEnabled bool
}

// Environ builds the complete, closed child environment for one build
// profile. Nothing from the ambient environment leaks through except the
// values resolved and frozen in r plus PATH/HOME/TMPDIR.
//
// The policy is fail-closed and read-only:
//   - GOFLAGS=-mod=readonly: the loader may never rewrite go.mod/go.sum;
//   - GOPROXY=off and GOSUMDB=off: no network;
//   - GOWORK=off, GOENV=off: no workspace or user configuration input;
//   - GOTOOLCHAIN=local: the pinned local toolchain only, no auto-switch.
func (r *Resolved) Environ(options EnvOptions) []string {
	cgo := "0"
	if options.CgoEnabled {
		cgo = "1"
	}
	env := []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
		"GOROOT=" + r.GOROOT,
		"GOCACHE=" + r.GOCACHE,
		"GOMODCACHE=" + r.GOMODCACHE,
		"GOPATH=" + r.GOPATH,
		"GOOS=" + options.GOOS,
		"GOARCH=" + options.GOARCH,
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
	if options.GOAMD64 != "" {
		env = append(env, "GOAMD64="+options.GOAMD64)
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
