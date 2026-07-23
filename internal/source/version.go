package source

import (
	"fmt"
	"go/build/constraint"
	"os"
	"strings"
)

func readGoDirective(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		if version, found := strings.CutPrefix(trimmed, "go "); found {
			return "go" + strings.TrimSpace(version)
		}
	}
	return ""
}

func effectiveGoVersion(raw []byte, base string) (string, error) {
	for _, line := range strings.Split(string(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "package ") {
			break
		}
		if !constraint.IsGoBuild(trimmed) {
			continue
		}
		expression, err := constraint.Parse(trimmed)
		if err != nil {
			return "", fmt.Errorf(
				"invalid //go:build constraint: %w", err,
			)
		}
		if version := constraint.GoVersion(expression); version != "" {
			return version, nil
		}
	}
	return base, nil
}
