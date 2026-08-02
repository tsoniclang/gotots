package certify

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
)

func validateSeeds(source []moduleSeed) ([]moduleSeed, error) {
	if len(source) == 0 {
		return nil, certifyError("configure modules", "", "module set is empty")
	}
	result := append([]moduleSeed(nil), source...)
	specifiers := make(map[string]struct{}, len(result))
	sources := make(map[string]struct{}, len(result))
	for index, seed := range result {
		if seed.GoImportPath == "" || seed.GoImportPath == "." ||
			path.Clean(seed.GoImportPath) != seed.GoImportPath ||
			strings.HasPrefix(seed.GoImportPath, "../") ||
			strings.HasPrefix(seed.GoImportPath, "/") {
			return nil, certifyError(
				"configure modules",
				seed.GoImportPath,
				"Go import path is not canonical",
			)
		}
		if _, ok := providerSubpath(seed.Specifier); !ok ||
			path.Clean(seed.SourcePath) != seed.SourcePath ||
			!strings.HasPrefix(seed.SourcePath, "src/") ||
			!strings.HasSuffix(seed.SourcePath, ".ts") {
			return nil, certifyError("configure modules", seed.GoImportPath, "identity is incomplete")
		}
		if index != 0 && result[index-1].GoImportPath >= seed.GoImportPath {
			return nil, certifyError(
				"configure modules",
				seed.GoImportPath,
				"modules are not strictly ordered",
			)
		}
		if _, duplicate := specifiers[seed.Specifier]; duplicate {
			return nil, certifyError("configure modules", seed.Specifier, "specifier is duplicated")
		}
		if _, duplicate := sources[seed.SourcePath]; duplicate {
			return nil, certifyError("configure modules", seed.SourcePath, "source is duplicated")
		}
		specifiers[seed.Specifier] = struct{}{}
		sources[seed.SourcePath] = struct{}{}
	}
	return result, nil
}

func verifyPublicName(name string, targetType string) error {
	if name == "" || targetType == "" {
		return fmt.Errorf("public symbol identity is incomplete")
	}
	for _, forbidden := range []string{
		"$argument",
		"__from_",
		"$cooperative_",
		"$contract",
		"$state",
	} {
		if strings.Contains(name, forbidden) || strings.Contains(targetType, forbidden) {
			return fmt.Errorf("public symbol contains encoded ABI spelling %q", forbidden)
		}
	}
	return nil
}

func compareCanonical(left []byte, right []byte) error {
	if bytes.Equal(left, right) {
		return nil
	}
	return certifyError(
		"verify manifest",
		"canonical bytes",
		"checked manifest differs from independently regenerated evidence",
	)
}

func readManifest(path string) ([]byte, gostdlib.Manifest, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return nil, gostdlib.Manifest{}, certifyError("read manifest", path, err.Error())
	}
	manifest, err := gostdlib.Parse(payload)
	if err != nil {
		return nil, gostdlib.Manifest{}, err
	}
	canonical, err := gostdlib.Encode(manifest)
	if err != nil {
		return nil, gostdlib.Manifest{}, err
	}
	return canonical, manifest, nil
}
