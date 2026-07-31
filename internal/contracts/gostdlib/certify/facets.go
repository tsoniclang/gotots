package certify

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
)

const facetMapSchemaVersion = 1

type facetMapDocument struct {
	SchemaVersion int         `json:"schemaVersion"`
	Facets        []facetSeed `json:"facets"`
}

type facetSeed struct {
	Kind           gostdlib.FacetKind         `json:"kind"`
	SourceIdentity string                     `json:"sourceIdentity"`
	Capabilities   []gostdlib.FacetCapability `json:"capabilities,omitempty"`
	ProfileKey     string                     `json:"profileKey,omitempty"`
	Specifier      string                     `json:"specifier"`
	SourcePath     string                     `json:"sourcePath"`
	Export         string                     `json:"export"`
	StorageExport  string                     `json:"storageExport,omitempty"`
	Effect         gostdlib.EffectKind        `json:"effect,omitempty"`
}

func readFacetSeeds(sourcePath string) ([]facetSeed, error) {
	file, err := os.Open(sourcePath)
	if err != nil {
		return nil, certifyError("read facet map", sourcePath, err.Error())
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var document facetMapDocument
	if err := decoder.Decode(&document); err != nil {
		return nil, certifyError("read facet map", sourcePath, err.Error())
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("multiple JSON values")
		}
		return nil, certifyError("read facet map", sourcePath, err.Error())
	}
	if document.SchemaVersion != facetMapSchemaVersion {
		return nil, certifyError("read facet map", sourcePath, "schema is unsupported")
	}
	return validateFacetSeeds(document.Facets)
}

func validateFacetSeeds(source []facetSeed) ([]facetSeed, error) {
	result := append([]facetSeed(nil), source...)
	for index := range result {
		result[index].Capabilities = append(
			[]gostdlib.FacetCapability(nil),
			result[index].Capabilities...,
		)
	}
	sort.Slice(result, func(left, right int) bool {
		return facetSeedKey(result[left]) < facetSeedKey(result[right])
	})
	lookups := make(map[string]struct{})
	targets := make(map[string]struct{})
	previous := ""
	for index, seed := range result {
		key := facetSeedKey(seed)
		if key == "" || index != 0 && key == previous {
			return nil, certifyError("configure facets", key, "facet identity is invalid or duplicated")
		}
		previous = key
		if !seed.Kind.Valid() || seed.SourceIdentity == "" ||
			seed.Specifier == "" || seed.SourcePath == "" || seed.Export == "" {
			return nil, certifyError("configure facets", key, "facet identity is incomplete")
		}
		if subpath, ok := providerSubpath(seed.Specifier); !ok ||
			!strings.HasPrefix(subpath, "./internal/facets/") ||
			!strings.HasPrefix(seed.SourcePath, "src/internal/facets/") ||
			!strings.HasSuffix(seed.SourcePath, ".ts") {
			return nil, certifyError("configure facets", key, "facet module is invalid")
		}
		capabilities := make([]string, 0, len(seed.Capabilities)+1)
		for _, capability := range seed.Capabilities {
			capabilities = append(capabilities, string(capability))
		}
		if seed.ProfileKey != "" {
			capabilities = append(capabilities, seed.ProfileKey)
		}
		if len(capabilities) == 0 {
			return nil, certifyError("configure facets", key, "capability set is empty")
		}
		for _, capability := range capabilities {
			lookup := seed.SourceIdentity + "\x00" + string(seed.Kind) + "\x00" + capability
			if _, duplicate := lookups[lookup]; duplicate {
				return nil, certifyError("configure facets", lookup, "capability owner is duplicated")
			}
			lookups[lookup] = struct{}{}
		}
		for _, export := range []string{seed.Export, seed.StorageExport} {
			if export == "" {
				continue
			}
			target := seed.Specifier + "\x00" + export
			if _, duplicate := targets[target]; duplicate {
				return nil, certifyError("configure facets", target, "target owner is duplicated")
			}
			targets[target] = struct{}{}
		}
	}
	return result, nil
}

func facetSeedKey(seed facetSeed) string {
	return seed.Specifier + "\x00" + seed.SourceIdentity + "\x00" +
		string(seed.Kind) + "\x00" + seed.Export
}
