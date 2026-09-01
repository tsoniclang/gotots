package gototsruntime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	corefixture "github.com/tsoniclang/gotots/internal/testfixture/tsoniccore"
)

func InstallResolution(root string, outputDirectory string) error {
	if root == "" || outputDirectory == "" {
		return fmt.Errorf("install GoToTS runtime resolution fixture: path is absent")
	}
	if err := corefixture.InstallResolutionOnly(root); err != nil {
		return fmt.Errorf("install GoToTS runtime resolution fixture dependency: %w", err)
	}
	runtimeRoot := filepath.Join(root, "runtime")
	entries, err := os.ReadDir(runtimeRoot)
	if err != nil {
		return fmt.Errorf("install GoToTS runtime resolution fixture: %w", err)
	}
	modules := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".ts" {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".ts")
		if name == "" || strings.ContainsAny(name, `/\\`) {
			return fmt.Errorf(
				"install GoToTS runtime resolution fixture: module %q is invalid",
				entry.Name(),
			)
		}
		modules = append(modules, name)
	}
	sort.Strings(modules)
	if len(modules) == 0 {
		return fmt.Errorf("install GoToTS runtime resolution fixture: runtime package is empty")
	}
	packageRoot := filepath.Join(root, "node_modules", "@gotots", "runtime")
	if err := os.MkdirAll(packageRoot, 0o755); err != nil {
		return fmt.Errorf("install GoToTS runtime resolution fixture: %w", err)
	}
	manifest, err := json.MarshalIndent(map[string]any{
		"name":    "@gotots/runtime",
		"version": "0.0.0",
		"type":    "module",
		"exports": map[string]any{
			"./*.js": map[string]string{
				"types":   "./*.d.ts",
				"default": "./*.js",
			},
		},
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("install GoToTS runtime resolution fixture: %w", err)
	}
	if err := os.WriteFile(
		filepath.Join(packageRoot, "package.json"),
		append(manifest, '\n'),
		0o600,
	); err != nil {
		return fmt.Errorf("install GoToTS runtime resolution fixture: %w", err)
	}
	for _, module := range modules {
		typeSource := fmt.Sprintf(
			"export * from %q;\n",
			"../../../runtime/"+module+".js",
		)
		runtimeTarget := filepath.Join(outputDirectory, "runtime", module+".js")
		relativeTarget, err := filepath.Rel(packageRoot, runtimeTarget)
		if err != nil {
			return fmt.Errorf(
				"install GoToTS runtime resolution fixture %s: %w",
				module,
				err,
			)
		}
		relativeTarget = filepath.ToSlash(relativeTarget)
		if !strings.HasPrefix(relativeTarget, ".") {
			relativeTarget = "./" + relativeTarget
		}
		valueSource := fmt.Sprintf("export * from %q;\n", relativeTarget)
		for name, source := range map[string]string{
			module + ".d.ts": typeSource,
			module + ".js":   valueSource,
		} {
			if err := os.WriteFile(
				filepath.Join(packageRoot, name),
				[]byte(source),
				0o600,
			); err != nil {
				return fmt.Errorf(
					"install GoToTS runtime resolution fixture %s: %w",
					module,
					err,
				)
			}
		}
	}
	return nil
}
