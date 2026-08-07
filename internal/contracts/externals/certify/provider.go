package certify

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tsoniclang/gotots/internal/contracts/externals"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func verifyProviderTypecheck(config resolvedConfig) error {
	err := tsgo.Compile(
		context.Background(),
		config.repositoryRoot,
		config.providerRoot,
		[]string{"--noEmit", "-p", config.tsConfigPath},
	)
	if err != nil {
		return certifyError(
			"typecheck provider",
			config.tsConfigPath,
			err.Error(),
		)
	}
	return nil
}

type packageDocument struct {
	Name    string                     `json:"name"`
	Version string                     `json:"version"`
	Exports map[string]json.RawMessage `json:"exports"`
}

type packageExport struct {
	Types   string `json:"types"`
	Default string `json:"default"`
}

type providerTarget struct {
	fingerprint string
	owner       string
}

const (
	providerEffectMarkerPath = "src/internal/effect.ts"
	providerEffectMarkerName = "AsyncEffectMarker"
)

func readProviderPackage(
	config resolvedConfig,
	seeds []bindingSeed,
) (packageDocument, error) {
	packagePath := filepath.Join(config.providerRoot, "package.json")
	payload, err := os.ReadFile(packagePath)
	if err != nil {
		return packageDocument{}, certifyError("read provider package", packagePath, err.Error())
	}
	var document packageDocument
	if err := json.Unmarshal(payload, &document); err != nil {
		return packageDocument{}, certifyError("read provider package", packagePath, err.Error())
	}
	if document.Name != externals.PackageName ||
		document.Version != externals.PackageVersion {
		return packageDocument{}, certifyError(
			"read provider package",
			packagePath,
			"package identity is invalid",
		)
	}
	expected := make(map[string]packageExport)
	for _, seed := range seeds {
		if seed.TargetKind != externals.TargetModule {
			continue
		}
		subpath := "./" + strings.TrimPrefix(
			seed.ModuleSpecifier,
			externals.PackageName+"/",
		)
		base := strings.TrimSuffix(seed.SourcePath, ".ts")
		want := packageExport{
			Types:   "./dist/" + base + ".d.ts",
			Default: "./dist/" + base + ".js",
		}
		if existing, duplicate := expected[subpath]; duplicate && existing != want {
			return packageDocument{}, certifyError(
				"verify provider exports",
				subpath,
				"module target is inconsistent",
			)
		}
		expected[subpath] = want
	}
	for subpath, want := range expected {
		encoded, ok := document.Exports[subpath]
		if !ok {
			return packageDocument{}, certifyError(
				"verify provider exports",
				subpath,
				"module is absent",
			)
		}
		var got packageExport
		if err := json.Unmarshal(encoded, &got); err != nil || got != want {
			return packageDocument{}, certifyError(
				"verify provider exports",
				subpath,
				fmt.Sprintf("target is %#v, want %#v", got, want),
			)
		}
	}
	for subpath := range document.Exports {
		if subpath == "./package.json" {
			continue
		}
		if _, ok := expected[subpath]; !ok {
			return packageDocument{}, certifyError(
				"verify provider exports",
				subpath,
				"module has no binding owner",
			)
		}
	}
	return document, nil
}

func inspectProviderTargets(
	config resolvedConfig,
	seeds []bindingSeed,
	sources map[string]sourcePackage,
) (map[string]providerTarget, error) {
	client, err := tsgo.StartClient(config.repositoryRoot, config.providerRoot)
	if err != nil {
		return nil, err
	}
	project, err := client.OpenProject(config.tsConfigPath)
	if err != nil {
		client.Close()
		return nil, err
	}
	result, inspectErr := inspectProviderProject(config, project, seeds, sources)
	closeErr := client.Close()
	if inspectErr != nil {
		return nil, inspectErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	return result, nil
}

func inspectProviderProject(
	config resolvedConfig,
	project *tsgo.ProjectInspection,
	seeds []bindingSeed,
	sources map[string]sourcePackage,
) (map[string]providerTarget, error) {
	effectMarker, err := providerEffectMarker(config, project)
	if err != nil {
		return nil, err
	}
	byPath := make(map[string][]bindingSeed)
	for _, seed := range seeds {
		if seed.TargetKind == externals.TargetModule {
			byPath[seed.SourcePath] = append(byPath[seed.SourcePath], seed)
		}
	}
	result := make(map[string]providerTarget)
	for sourcePath, selectedSeeds := range byPath {
		exports, err := project.Exports(filepath.Join(
			config.providerRoot,
			filepath.FromSlash(sourcePath),
		))
		if err != nil {
			return nil, err
		}
		byName := make(map[string]tsgo.ProjectExport, len(exports))
		for _, target := range exports {
			byName[target.Name()] = target
		}
		if len(byName) != len(selectedSeeds) {
			return nil, certifyError(
				"inspect provider module",
				sourcePath,
				fmt.Sprintf("export count is %d, want %d", len(byName), len(selectedSeeds)),
			)
		}
		for _, seed := range selectedSeeds {
			target, ok := byName[seed.Export]
			source, sourceOK := sources[seed.SourcePackage].functions[seed.SourceName]
			if !ok || target.Fingerprint() == "" || !sourceOK ||
				source.function == nil || source.signature == nil || source.body {
				return nil, certifyError(
					"inspect provider export",
					seed.ModuleSpecifier+"#"+seed.Export,
					"target or source evidence is absent",
				)
			}
			parameters, err := project.CallableParameterCount(target)
			if err != nil {
				return nil, err
			}
			typeParameters, err := project.CallableTypeParameterCount(target)
			if err != nil {
				return nil, err
			}
			if parameters != source.signature.Params().Len() || typeParameters != 0 {
				return nil, certifyError(
					"inspect provider export",
					seed.ModuleSpecifier+"#"+seed.Export,
					"source-facing parameter or generic arity differs",
				)
			}
			effect, err := project.CallableEffect(target, effectMarker)
			if err != nil {
				return nil, err
			}
			if effect != tsgo.CallableEffectSynchronous {
				return nil, certifyError(
					"inspect provider export",
					seed.ModuleSpecifier+"#"+seed.Export,
					"target adds asynchronous behavior to a synchronous Go boundary",
				)
			}
			owners := target.ImplementationOwners()
			if len(owners) != 1 || owners[0] != sourcePath {
				return nil, certifyError(
					"inspect provider export",
					seed.ModuleSpecifier+"#"+seed.Export,
					"target does not have one canonical source owner",
				)
			}
			result[seed.SourcePackage+"\x00"+seed.SourceName] = providerTarget{
				fingerprint: target.Fingerprint(),
				owner:       owners[0],
			}
		}
	}
	return result, nil
}

func providerEffectMarker(
	config resolvedConfig,
	project *tsgo.ProjectInspection,
) (tsgo.ProjectExport, error) {
	exports, err := project.Exports(filepath.Join(
		config.providerRoot,
		filepath.FromSlash(providerEffectMarkerPath),
	))
	if err != nil {
		return tsgo.ProjectExport{}, err
	}
	if len(exports) != 1 || exports[0].Name() != providerEffectMarkerName {
		return tsgo.ProjectExport{}, certifyError(
			"inspect provider effect marker",
			providerEffectMarkerPath,
			"marker export is not exact",
		)
	}
	return exports[0], nil
}

func providerDigest(config resolvedConfig) (string, error) {
	paths := []string{
		"package.json",
		"tsconfig.json",
		filepath.ToSlash(relativeProviderPath(config, config.bindingMapPath)),
	}
	sourceRoot := filepath.Join(config.providerRoot, "src")
	err := filepath.WalkDir(sourceRoot, func(sourcePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("provider source symlink is forbidden: %s", sourcePath)
		}
		if filepath.Ext(sourcePath) == ".ts" {
			paths = append(paths, filepath.ToSlash(relativeProviderPath(config, sourcePath)))
		}
		return nil
	})
	if err != nil {
		return "", certifyError("hash provider", sourceRoot, err.Error())
	}
	sort.Strings(paths)
	hash := sha256.New()
	previous := ""
	for _, relative := range paths {
		if relative == previous || relative == "" || strings.HasPrefix(relative, "../") {
			return "", certifyError("hash provider", relative, "path is duplicate or outside provider")
		}
		previous = relative
		payload, err := os.ReadFile(filepath.Join(config.providerRoot, filepath.FromSlash(relative)))
		if err != nil {
			return "", certifyError("hash provider", relative, err.Error())
		}
		fmt.Fprintf(hash, "%d:%s:%d:", len(relative), relative, len(payload))
		hash.Write(payload)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func relativeProviderPath(config resolvedConfig, sourcePath string) string {
	relative, err := filepath.Rel(config.providerRoot, sourcePath)
	if err != nil {
		return ""
	}
	return relative
}
