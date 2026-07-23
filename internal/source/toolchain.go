package source

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// resolveToolchain resolves the exact selected go binary and its fingerprint,
// and produces the environment shared by every loader and verifier.
func resolveToolchain(req Request) (Toolchain, []string, func(), error) {
	noop := func() {}
	binary := req.GoBinary
	if binary == "" {
		resolved, err := exec.LookPath("go")
		if err != nil {
			return Toolchain{}, nil, noop, &LoadError{
				Dir:    req.Dir,
				Reason: "no go binary selected or on PATH: " + err.Error(),
			}
		}
		binary = resolved
	}
	absolute, err := filepath.Abs(binary)
	if err != nil {
		return Toolchain{}, nil, noop, &LoadError{
			Dir:    req.Dir,
			Reason: "toolchain path unresolvable: " + err.Error(),
		}
	}
	shimDir, err := os.MkdirTemp("", "gotots-toolchain-*")
	if err != nil {
		return Toolchain{}, nil, noop, &LoadError{
			Dir: req.Dir, Reason: "toolchain shim: " + err.Error(),
		}
	}
	cleanup := func() { _ = os.RemoveAll(shimDir) }
	if err := os.Symlink(absolute, filepath.Join(shimDir, "go")); err != nil {
		cleanup()
		return Toolchain{}, nil, noop, &LoadError{
			Dir: req.Dir, Reason: "toolchain shim: " + err.Error(),
		}
	}
	env := append(os.Environ(), req.Env...)
	env = append(
		env,
		"PATH="+shimDir+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	keys := []string{
		"GOROOT", "GOVERSION", "GOOS", "GOARCH", "GOEXPERIMENT",
		"GOFLAGS", "CGO_ENABLED", "CC", "CXX", "CGO_CFLAGS",
		"CGO_CPPFLAGS", "CGO_CXXFLAGS", "CGO_FFLAGS", "CGO_LDFLAGS",
		"PKG_CONFIG", "GOAMD64", "GOARM", "GOARM64", "GOMIPS",
		"GOMIPS64", "GOPPC64", "GORISCV64", "GOWASM",
	}
	args := append([]string{"env"}, keys...)
	out, err := runGo(absolute, env, req.Dir, args...)
	if err != nil {
		cleanup()
		return Toolchain{}, nil, noop, &LoadError{
			Dir:    req.Dir,
			Reason: "toolchain fingerprint failed: " + err.Error(),
		}
	}
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(lines) != len(keys) {
		cleanup()
		return Toolchain{}, nil, noop, &LoadError{
			Dir:    req.Dir,
			Reason: "toolchain fingerprint output malformed",
		}
	}
	binaryBytes, err := os.ReadFile(absolute)
	if err != nil {
		cleanup()
		return Toolchain{}, nil, noop, &LoadError{
			Dir:    req.Dir,
			Reason: "toolchain binary unreadable: " + err.Error(),
		}
	}
	values := map[string]string{}
	configHash := sha256.New()
	for index, key := range keys {
		values[key] = strings.TrimSpace(lines[index])
		fmt.Fprintf(configHash, "%s=%s\n", key, values[key])
	}
	return Toolchain{
		binary: absolute,
		binaryDigest: fmt.Sprintf(
			"%x", sha256.Sum256(binaryBytes),
		),
		goroot: values["GOROOT"], version: values["GOVERSION"],
		goos: values["GOOS"], goarch: values["GOARCH"],
		experiments: values["GOEXPERIMENT"], goflags: values["GOFLAGS"],
		cgoEnabled:        values["CGO_ENABLED"],
		buildConfigDigest: fmt.Sprintf("%x", configHash.Sum(nil)),
	}, env, cleanup, nil
}

func listPatternSet(
	toolchain Toolchain,
	env []string,
	dir string,
	pattern string,
) (map[string]bool, error) {
	out, err := runGo(toolchain.binary, env, dir, "list", pattern)
	if err != nil {
		return nil, &LoadError{
			Dir:    dir,
			Reason: "go list " + pattern + " failed: " + err.Error(),
		}
	}
	set := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line != "" {
			set[line] = true
		}
	}
	return set, nil
}

func runGo(
	binary string,
	env []string,
	dir string,
	args ...string,
) (string, error) {
	command := exec.Command(binary, args...)
	command.Dir = dir
	command.Env = env
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err != nil {
		return "", &LoadError{
			Dir: dir,
			Reason: strings.Join(args, " ") + ": " +
				err.Error() + ": " + stderr.String(),
		}
	}
	return stdout.String(), nil
}
