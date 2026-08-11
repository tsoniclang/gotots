package load

import (
	"encoding/json"
	"fmt"
	"go/version"
	"io"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/tsoniclang/gotots/internal/toolchain"
)

func init() {
	if os.Getenv(toolchain.PackagesDriverModeEnvironment) != "v1" {
		return
	}
	if err := serveExactPackagesDriver(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}

func serveExactPackagesDriver(patterns []string) error {
	if len(patterns) == 0 {
		return fmt.Errorf("exact Go packages driver: patterns are absent")
	}
	request, err := decodeDriverRequest(os.Stdin)
	if err != nil {
		return err
	}
	minor, err := selectedGoMinorVersion()
	if err != nil {
		return err
	}
	arch := os.Getenv(toolchain.PackagesDriverArchEnvironment)
	if arch == "" {
		return fmt.Errorf("exact Go packages driver: selected architecture is absent")
	}
	mode := metadataMode(request.Mode)
	loaded, err := packages.Load(&packages.Config{
		Env:        nestedDriverEnvironment(request.Env),
		BuildFlags: slices.Clone(request.BuildFlags),
		Mode:       mode,
		Overlay:    cloneOverlay(request.Overlay),
		Tests:      request.Tests,
	}, patterns...)
	if err != nil {
		return fmt.Errorf("exact Go packages driver: %w", err)
	}
	all := make(map[string]*packages.Package)
	packages.Visit(loaded, nil, func(current *packages.Package) {
		all[current.ID] = current
	})
	ordered := make([]*packages.Package, 0, len(all))
	for _, current := range all {
		ordered = append(ordered, current)
	}
	sort.Slice(ordered, func(left int, right int) bool {
		return ordered[left].ID < ordered[right].ID
	})
	roots := make([]string, len(loaded))
	for index, root := range loaded {
		roots[index] = root.ID
	}
	if err := writeDriverEvidence(
		os.Getenv(toolchain.PackagesDriverEvidenceEnvironment),
		roots,
		ordered,
	); err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(packages.DriverResponse{
		Compiler:  "gc",
		Arch:      arch,
		Roots:     roots,
		Packages:  ordered,
		GoVersion: minor,
	})
}

func decodeDriverRequest(input io.Reader) (packages.DriverRequest, error) {
	var request packages.DriverRequest
	decoder := json.NewDecoder(input)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return packages.DriverRequest{}, fmt.Errorf("exact Go packages driver: decode request: %w", err)
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("multiple JSON values")
		}
		return packages.DriverRequest{}, fmt.Errorf("exact Go packages driver: decode request: %w", err)
	}
	return request, nil
}

func metadataMode(requested packages.LoadMode) packages.LoadMode {
	const metadata = packages.NeedName |
		packages.NeedFiles |
		packages.NeedCompiledGoFiles |
		packages.NeedImports |
		packages.NeedDeps |
		packages.NeedExportFile |
		packages.NeedModule |
		packages.NeedEmbedFiles |
		packages.NeedEmbedPatterns |
		packages.NeedForTest
	selected := requested & metadata
	return selected
}

func nestedDriverEnvironment(source []string) []string {
	result := make([]string, 0, len(source)+1)
	for _, entry := range source {
		name, _, _ := strings.Cut(entry, "=")
		if name == "GOPACKAGESDRIVER" ||
			name == toolchain.PackagesDriverModeEnvironment ||
			name == toolchain.PackagesDriverVersionEnvironment ||
			name == toolchain.PackagesDriverArchEnvironment ||
			name == toolchain.PackagesDriverEvidenceEnvironment {
			continue
		}
		result = append(result, entry)
	}
	return append(result, "GOPACKAGESDRIVER=off")
}

func selectedGoMinorVersion() (int, error) {
	selected := os.Getenv(toolchain.PackagesDriverVersionEnvironment)
	language := version.Lang(selected)
	minorText, ok := strings.CutPrefix(language, "go1.")
	if !ok {
		return 0, fmt.Errorf("exact Go packages driver: selected version %q is invalid", selected)
	}
	minor, err := strconv.Atoi(minorText)
	if err != nil || minor <= 0 {
		return 0, fmt.Errorf("exact Go packages driver: selected version %q is invalid", selected)
	}
	return minor, nil
}
