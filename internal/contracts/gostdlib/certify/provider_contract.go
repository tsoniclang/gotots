package certify

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const providerSupportPath = "src/internal/facets/provider-support.ts"

type providerSupportMarkers struct {
	view         tsgo.ProjectExport
	guard        tsgo.ProjectExport
	contract     tsgo.ProjectExport
	fromProvider tsgo.ProjectExport
}

func loadProviderSupportMarkers(
	config resolvedConfig,
	project *tsgo.ProjectInspection,
) (providerSupportMarkers, error) {
	exports, err := project.Exports(filepath.Join(
		config.providerRoot,
		filepath.FromSlash(providerSupportPath),
	))
	if err != nil {
		return providerSupportMarkers{}, err
	}
	byName := make(map[string]tsgo.ProjectExport, len(exports))
	for _, selected := range exports {
		byName[selected.Name()] = selected
	}
	if len(exports) != 4 {
		return providerSupportMarkers{}, certifyError(
			"inspect provider support",
			providerSupportPath,
			fmt.Sprintf("marker export count is %d, want 4", len(exports)),
		)
	}
	result := providerSupportMarkers{
		view:         byName["InterfaceView"],
		guard:        byName["InterfaceGuard"],
		contract:     byName["InterfaceContract"],
		fromProvider: byName["FromProviderBridge"],
	}
	if result.view.Name() == "" || result.guard.Name() == "" ||
		result.contract.Name() == "" ||
		result.fromProvider.Name() == "" {
		return providerSupportMarkers{}, certifyError(
			"inspect provider support",
			providerSupportPath,
			"marker export set is not exact",
		)
	}
	return result, nil
}

func verifyProviderSupportParameters(
	project *tsgo.ProjectInspection,
	target tsgo.ProjectExport,
	baseParameters int,
	viewCount int,
	guardCount int,
	contractCount int,
	fromProviderCount int,
	markers providerSupportMarkers,
) error {
	type expectedParameter struct {
		kind   string
		marker tsgo.ProjectExport
	}
	expected := make([]expectedParameter, 0,
		viewCount+guardCount+contractCount+fromProviderCount)
	for range viewCount {
		expected = append(expected, expectedParameter{"capability-view", markers.view})
	}
	for range guardCount {
		expected = append(expected, expectedParameter{"guard", markers.guard})
	}
	for range contractCount {
		expected = append(expected, expectedParameter{"contract", markers.contract})
	}
	for range fromProviderCount {
		expected = append(expected, expectedParameter{"from-provider", markers.fromProvider})
	}
	actualCount, err := project.CallableParameterCount(target)
	if err != nil {
		return err
	}
	wantCount := baseParameters + len(expected)
	if actualCount != wantCount {
		return certifyError(
			"verify provider support parameters",
			target.Name(),
			fmt.Sprintf("target has %d parameters, contract requires %d", actualCount, wantCount),
		)
	}
	for offset, selected := range expected {
		parameter := baseParameters + offset
		identity, err := project.CallableParameterTypeIdentity(target, parameter)
		if err != nil {
			return certifyError(
				"verify provider support parameters",
				target.Name(),
				fmt.Sprintf("inspect parameter %d: %v", parameter, err),
			)
		}
		if !identity.Matches(selected.marker) {
			return certifyError(
				"verify provider support parameters",
				target.Name(),
				fmt.Sprintf("parameter %d is not the %s support contract", parameter, selected.kind),
			)
		}
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

func readProviderPackage(config resolvedConfig) (packageDocument, error) {
	packagePath := filepath.Join(config.providerRoot, "package.json")
	payload, err := os.ReadFile(packagePath)
	if err != nil {
		return packageDocument{}, certifyError("read package", packagePath, err.Error())
	}
	var document packageDocument
	if err := json.Unmarshal(payload, &document); err != nil {
		return packageDocument{}, certifyError("read package", packagePath, err.Error())
	}
	if document.Name != gostdlib.PackageName ||
		document.Version != gostdlib.PackageVersion ||
		len(document.Exports) == 0 {
		return packageDocument{}, certifyError(
			"read package",
			packagePath,
			"provider package identity is invalid",
		)
	}
	return document, nil
}

func verifyPackageModules(
	document packageDocument,
	providerScalarSubpath string,
	providerPointerSubpath string,
	seeds []moduleSeed,
	facets []facetSeed,
	profiles []providerCallableProfileSeed,
	statefulProfiles []providerStatefulProfileSeed,
	providerInterfaces []providerInterfaceSeed,
	providerCapabilities []providerInterfaceCapabilitySeed,
) error {
	expected := make(
		map[string]packageExport,
		len(seeds)+len(facets)+len(profiles)+len(statefulProfiles)+
			len(providerInterfaces)+len(providerCapabilities)+2,
	)
	for _, support := range []string{
		providerScalarSubpath,
		providerPointerSubpath,
	} {
		base := strings.TrimSuffix(strings.TrimPrefix(support, "./"), ".js")
		if support == "" {
			return certifyError(
				"verify package exports",
				support,
				"provider support module is absent",
			)
		}
		if _, duplicate := expected[support]; duplicate {
			return certifyError(
				"verify package exports",
				support,
				"provider support module is duplicated",
			)
		}
		expected[support] = packageExport{
			Types:   "./dist/src/" + base + ".d.ts",
			Default: "./dist/src/" + base + ".js",
		}
	}
	for _, seed := range seeds {
		subpath, ok := providerSubpath(seed.Specifier)
		if !ok {
			return certifyError("verify package exports", seed.Specifier, "specifier is invalid")
		}
		if _, duplicate := expected[subpath]; duplicate {
			return certifyError("verify package exports", subpath, "module is duplicated")
		}
		base := strings.TrimSuffix(seed.SourcePath, ".ts")
		expected[subpath] = packageExport{
			Types:   "./dist/" + base + ".d.ts",
			Default: "./dist/" + base + ".js",
		}
	}
	for _, seed := range facets {
		subpath, ok := providerSubpath(seed.Specifier)
		if !ok {
			return certifyError("verify package exports", seed.Specifier, "specifier is invalid")
		}
		base := strings.TrimSuffix(seed.SourcePath, ".ts")
		want := packageExport{
			Types:   "./dist/" + base + ".d.ts",
			Default: "./dist/" + base + ".js",
		}
		if existing, duplicate := expected[subpath]; duplicate && existing != want {
			return certifyError("verify package exports", subpath, "facet module is inconsistent")
		}
		expected[subpath] = want
	}
	for _, seed := range profiles {
		subpath, ok := providerSubpath(seed.Specifier)
		if !ok {
			return certifyError("verify package exports", seed.Specifier, "specifier is invalid")
		}
		base := strings.TrimSuffix(seed.SourcePath, ".ts")
		want := packageExport{
			Types:   "./dist/" + base + ".d.ts",
			Default: "./dist/" + base + ".js",
		}
		if existing, duplicate := expected[subpath]; duplicate && existing != want {
			return certifyError("verify package exports", subpath, "profile module is inconsistent")
		}
		expected[subpath] = want
	}
	for _, seed := range statefulProfiles {
		subpath, ok := providerSubpath(seed.Specifier)
		if !ok {
			return certifyError("verify package exports", seed.Specifier, "specifier is invalid")
		}
		base := strings.TrimSuffix(seed.SourcePath, ".ts")
		want := packageExport{
			Types:   "./dist/" + base + ".d.ts",
			Default: "./dist/" + base + ".js",
		}
		if existing, duplicate := expected[subpath]; duplicate && existing != want {
			return certifyError(
				"verify package exports",
				subpath,
				"stateful-profile module is inconsistent",
			)
		}
		expected[subpath] = want
	}
	for _, seed := range providerInterfaces {
		subpath, ok := providerSubpath(seed.Specifier)
		if !ok {
			return certifyError("verify package exports", seed.Specifier, "specifier is invalid")
		}
		base := strings.TrimSuffix(seed.SourcePath, ".ts")
		want := packageExport{
			Types:   "./dist/" + base + ".d.ts",
			Default: "./dist/" + base + ".js",
		}
		if existing, duplicate := expected[subpath]; duplicate && existing != want {
			return certifyError(
				"verify package exports",
				subpath,
				"provider-interface module is inconsistent",
			)
		}
		expected[subpath] = want
	}
	for _, seed := range providerCapabilities {
		subpath, ok := providerSubpath(seed.Specifier)
		if !ok {
			return certifyError("verify package exports", seed.Specifier, "specifier is invalid")
		}
		base := strings.TrimSuffix(seed.SourcePath, ".ts")
		want := packageExport{
			Types:   "./dist/" + base + ".d.ts",
			Default: "./dist/" + base + ".js",
		}
		if existing, duplicate := expected[subpath]; duplicate && existing != want {
			return certifyError(
				"verify package exports",
				subpath,
				"provider-capability module is inconsistent",
			)
		}
		expected[subpath] = want
	}
	for subpath, want := range expected {
		encoded, ok := document.Exports[subpath]
		if !ok {
			return certifyError("verify package exports", subpath, "module is absent")
		}
		var actual packageExport
		if err := json.Unmarshal(encoded, &actual); err != nil {
			return certifyError("verify package exports", subpath, err.Error())
		}
		if actual != want {
			return certifyError(
				"verify package exports",
				subpath,
				fmt.Sprintf("target is %#v, want %#v", actual, want),
			)
		}
	}
	for subpath := range document.Exports {
		if subpath == "./package.json" {
			continue
		}
		if _, ok := expected[subpath]; !ok {
			return certifyError(
				"verify package exports",
				subpath,
				"module has no manifest owner",
			)
		}
	}
	return nil
}

func providerSubpath(specifier string) (string, bool) {
	prefix := gostdlib.PackageName + "/"
	if !strings.HasPrefix(specifier, prefix) || !strings.HasSuffix(specifier, ".js") {
		return "", false
	}
	value := strings.TrimPrefix(specifier, prefix)
	if value == "" || strings.Contains(value, "..") {
		return "", false
	}
	return "./" + value, true
}

func providerDigest(config resolvedConfig) (string, error) {
	paths := []string{
		"package.json",
		"tsconfig.json",
		filepath.ToSlash(relativeProviderPath(config, config.moduleMapPath)),
		filepath.ToSlash(relativeProviderPath(config, config.facetMapPath)),
		filepath.ToSlash(relativeProviderPath(config, config.runtimeContractPath)),
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
		if filepath.Ext(sourcePath) != ".ts" {
			return nil
		}
		paths = append(paths, filepath.ToSlash(relativeProviderPath(config, sourcePath)))
		return nil
	})
	if err != nil {
		return "", certifyError("hash provider", sourceRoot, err.Error())
	}
	sort.Strings(paths)
	paths = compactStrings(paths)
	hash := sha256.New()
	for _, relative := range paths {
		if relative == "" || strings.HasPrefix(relative, "../") {
			return "", certifyError("hash provider", relative, "path is outside provider")
		}
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

func compactStrings(values []string) []string {
	if len(values) < 2 {
		return values
	}
	write := 1
	for read := 1; read < len(values); read++ {
		if values[read] == values[write-1] {
			continue
		}
		values[write] = values[read]
		write++
	}
	return values[:write]
}
