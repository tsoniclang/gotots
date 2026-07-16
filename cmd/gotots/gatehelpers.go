// Gate helper utilities: running commands inside the staging repo,
// output splitting, file digests, and census package enumeration.
package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/tsoniclang/gotots/internal/census"
)

func runInRepo(dir, name string, args ...string) (string, error) {
	command := exec.Command(name, args...)
	command.Dir = dir
	var out bytes.Buffer
	command.Stdout = &out
	command.Stderr = &out
	err := command.Run()
	return out.String(), err
}

func splitLines(out string) []string {
	var lines []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	if len(lines) > 40 {
		lines = lines[len(lines)-40:]
	}
	return lines
}

func digestFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read attested input %s: %w", path, err)
	}
	digest := sha256.Sum256(data)
	return fmt.Sprintf("%x", digest), nil
}

// ownedProductionPackages lists the owned production package paths in the
// census, for the module-retained denominator.
func ownedProductionPackages(run *census.Result) []string {
	seen := map[string]bool{}
	var out []string
	for _, decl := range run.Report.Declarations {
		if decl.Scope == "production" && !seen[decl.Package] {
			seen[decl.Package] = true
			out = append(out, decl.Package)
		}
	}
	return out
}
