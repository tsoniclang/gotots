package command

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"

	"github.com/tsoniclang/gotots/internal/config"
	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const buildManifestName = "gotots-manifest.json"

const projectPackageName = "package.json"

func programDigest(program *load.Program) (string, error) {
	hash := sha256.New()
	for _, sourcePackage := range program.Packages() {
		hash.Write([]byte(sourcePackage.ModulePath()))
		hash.Write([]byte{0})
		hash.Write([]byte(sourcePackage.ModuleVersion()))
		hash.Write([]byte{0})
		hash.Write([]byte(sourcePackage.Path()))
		files := sourcePackage.Files()
		sort.Slice(files, func(left, right int) bool {
			return filepath.Base(files[left].Path()) < filepath.Base(files[right].Path())
		})
		for _, sourceFile := range files {
			payload, err := os.ReadFile(sourceFile.Path())
			if err != nil {
				return "", commandError("digest source", err.Error())
			}
			hash.Write([]byte{0})
			hash.Write([]byte(filepath.Base(sourceFile.Path())))
			hash.Write([]byte{0})
			hash.Write(payload)
		}
		hash.Write([]byte{0xff})
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func writeEmission(
	ctx context.Context,
	project config.Project,
	emission emit.ProgramEmission,
	semanticDigest string,
) (int, error) {
	outputDirectory := project.OutputDirectory()
	if err := os.MkdirAll(outputDirectory, 0o755); err != nil {
		return 0, commandError("create output", err.Error())
	}
	client, err := tsgo.StartClient(project.DistributionRoot(), outputDirectory)
	if err != nil {
		return 0, err
	}
	files := emission.Files()
	paths := make([]string, 0, len(files)+3)
	for _, file := range files {
		printed, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			_ = client.Close()
			return 0, err
		}
		if err := writeTargetFile(outputDirectory, file.OutputPath(), []byte(printed)); err != nil {
			_ = client.Close()
			return 0, err
		}
		paths = append(paths, file.OutputPath())
	}
	if err := client.Close(); err != nil {
		return 0, err
	}
	if runtimePackage, ok := emission.RuntimePackage(); ok {
		if err := writeTargetFile(
			outputDirectory,
			runtimePackage.ManifestPath(),
			runtimePackage.Manifest(),
		); err != nil {
			return 0, err
		}
		paths = append(paths, runtimePackage.ManifestPath())
	}
	sort.Strings(paths)
	packageDocument, err := encodeProjectPackage()
	if err != nil {
		return 0, err
	}
	if err := writeTargetFile(outputDirectory, projectPackageName, packageDocument); err != nil {
		return 0, err
	}
	paths = append(paths, projectPackageName)
	tsconfig, err := encodeTSConfig(project, outputDirectory)
	if err != nil {
		return 0, err
	}
	if err := writeTargetFile(outputDirectory, "tsconfig.json", tsconfig); err != nil {
		return 0, err
	}
	paths = append(paths, "tsconfig.json")
	sort.Strings(paths)
	if err := tsgo.Compile(
		ctx,
		project.DistributionRoot(),
		outputDirectory,
		[]string{"--noEmit", "-p", filepath.Join(outputDirectory, "tsconfig.json")},
	); err != nil {
		return 0, err
	}
	manifest, err := encodeBuildManifest(semanticDigest, paths)
	if err != nil {
		return 0, err
	}
	if err := writeTargetFile(outputDirectory, buildManifestName, manifest); err != nil {
		return 0, err
	}
	return len(paths), nil
}

func encodeProjectPackage() ([]byte, error) {
	document := struct {
		Private bool   `json:"private"`
		Type    string `json:"type"`
	}{
		Private: true,
		Type:    "module",
	}
	payload, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, commandError("encode package", err.Error())
	}
	return append(payload, '\n'), nil
}

func encodeTSConfig(project config.Project, outputDirectory string) ([]byte, error) {
	providerPath := func(parts ...string) (string, error) {
		target := filepath.Join(append([]string{project.DistributionRoot()}, parts...)...)
		relative, err := filepath.Rel(outputDirectory, target)
		if err != nil {
			return "", err
		}
		result := filepath.ToSlash(relative)
		if result != "." && len(result) > 0 && result[0] != '.' {
			result = "./" + result
		}
		return result, nil
	}
	standardLibrary, err := providerPath("gostdlib", "dist", "src", "*.d.ts")
	if err != nil {
		return nil, commandError("encode tsconfig", err.Error())
	}
	externals, err := providerPath("externals", "dist", "src", "*.d.ts")
	if err != nil {
		return nil, commandError("encode tsconfig", err.Error())
	}
	document := map[string]any{
		"compilerOptions": map[string]any{
			"target":           "ES2022",
			"module":           "NodeNext",
			"moduleResolution": "NodeNext",
			"paths": map[string][]string{
				"@gotots/runtime/*.js":   {"./runtime/*.ts"},
				"@gotots/gostdlib/*.js":  {standardLibrary},
				"@gotots/externals/*.js": {externals},
			},
			"strict":                           true,
			"exactOptionalPropertyTypes":       true,
			"noUncheckedIndexedAccess":         true,
			"noImplicitOverride":               true,
			"noFallthroughCasesInSwitch":       true,
			"forceConsistentCasingInFileNames": true,
			"skipLibCheck":                     false,
			"types":                            []string{},
			"noEmit":                           true,
		},
		"include": []string{"**/*.ts"},
		"exclude": []string{"node_modules", "out"},
	}
	payload, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, commandError("encode tsconfig", err.Error())
	}
	return append(payload, '\n'), nil
}

func writeTargetFile(root string, relative string, payload []byte) error {
	if filepath.IsAbs(relative) || filepath.ToSlash(filepath.Clean(relative)) != relative ||
		relative == "." || relative == ".." ||
		len(relative) >= 3 && relative[:3] == "../" {
		return commandError("write output", fmt.Sprintf("target path %q is invalid", relative))
	}
	target := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return commandError("write output", err.Error())
	}
	if err := os.WriteFile(target, payload, 0o644); err != nil {
		return commandError("write output", err.Error())
	}
	return nil
}

func encodeBuildManifest(semanticDigest string, files []string) ([]byte, error) {
	selected := slices.Clone(files)
	if !slices.IsSorted(selected) {
		return nil, commandError("encode build manifest", "files are not sorted")
	}
	document := struct {
		SchemaVersion  int      `json:"schemaVersion"`
		SemanticDigest string   `json:"semanticDigest"`
		Files          []string `json:"files"`
	}{
		SchemaVersion:  config.SchemaVersion,
		SemanticDigest: semanticDigest,
		Files:          selected,
	}
	payload, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, commandError("encode build manifest", err.Error())
	}
	return append(payload, '\n'), nil
}
