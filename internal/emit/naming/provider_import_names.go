package naming

import (
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func (r *Registry) indexProviderImportNames(modules []string) error {
	selected := slices.Clone(modules)
	slices.Sort(selected)
	seenModules := make(map[string]struct{}, len(selected))
	usedNames := make(map[string]struct{}, len(selected))
	for _, module := range selected {
		if module == "" {
			return &api.NameError{Reason: "provider module identity is empty"}
		}
		if _, duplicate := seenModules[module]; duplicate {
			return &api.NameError{
				Name:   module,
				Reason: "provider module identity is duplicated",
			}
		}
		seenModules[module] = struct{}{}
		base, err := providerModuleImportBase(module)
		if err != nil {
			return err
		}
		name := base
		for suffix := uint64(1); ; suffix++ {
			if _, duplicate := usedNames[name]; !duplicate {
				break
			}
			name = base + "__provider_" + strconv.FormatUint(suffix, 10)
		}
		usedNames[name] = struct{}{}
		r.providerImportNameByModule[module] = name
	}
	return nil
}

func providerModuleImportBase(module string) (string, error) {
	base := path.Base(module)
	if !strings.HasSuffix(base, ".js") {
		return "", &api.NameError{
			Name:   module,
			Reason: "provider module does not have an ESM JavaScript suffix",
		}
	}
	base = strings.TrimSuffix(base, ".js")
	base = strings.ReplaceAll(base, "-", "_")
	base = portableIdentifier(base)
	if base == "" {
		return "", &api.NameError{
			Name:   module,
			Reason: "provider module has no import-name stem",
		}
	}
	return base, nil
}
