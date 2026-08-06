package environment

import (
	"fmt"
	"os"
	"runtime"
	"slices"
	"sort"
	"strings"
)

type BuildProfile struct {
	toolchainVersion string
	goos             string
	goarch           string
	cgoEnabled       bool
	tags             []string
}

type BuildProfileError struct {
	Field  string
	Reason string
}

func (e *BuildProfileError) Error() string {
	if e.Field == "" {
		return "resolve Go build profile: " + e.Reason
	}
	return fmt.Sprintf(
		"resolve Go build profile field %q: %s",
		e.Field,
		e.Reason,
	)
}

func NewBuildProfile(
	goos string,
	goarch string,
	cgoEnabled bool,
	tags []string,
) (BuildProfile, error) {
	return NewBuildProfileForToolchain(
		runtime.Version(),
		goos,
		goarch,
		cgoEnabled,
		tags,
	)
}

func NewBuildProfileForToolchain(
	toolchainVersion string,
	goos string,
	goarch string,
	cgoEnabled bool,
	tags []string,
) (BuildProfile, error) {
	if !strings.HasPrefix(toolchainVersion, "go") ||
		!validBuildWord(toolchainVersion) {
		return BuildProfile{}, &BuildProfileError{
			Field:  "toolchain version",
			Reason: "value is empty or non-canonical",
		}
	}
	if !validBuildWord(goos) {
		return BuildProfile{}, &BuildProfileError{
			Field:  "GOOS",
			Reason: "value is empty or non-canonical",
		}
	}
	if !validBuildWord(goarch) {
		return BuildProfile{}, &BuildProfileError{
			Field:  "GOARCH",
			Reason: "value is empty or non-canonical",
		}
	}
	selectedTags := slices.Clone(tags)
	if selectedTags == nil {
		selectedTags = []string{}
	}
	for _, tag := range selectedTags {
		if !validBuildWord(tag) {
			return BuildProfile{}, &BuildProfileError{
				Field:  "build tags",
				Reason: fmt.Sprintf("tag %q is non-canonical", tag),
			}
		}
	}
	sort.Strings(selectedTags)
	for index := 1; index < len(selectedTags); index++ {
		if selectedTags[index] == selectedTags[index-1] {
			return BuildProfile{}, &BuildProfileError{
				Field:  "build tags",
				Reason: fmt.Sprintf("tag %q is duplicated", selectedTags[index]),
			}
		}
	}
	return BuildProfile{
		toolchainVersion: toolchainVersion,
		goos:             goos,
		goarch:           goarch,
		cgoEnabled:       cgoEnabled,
		tags:             selectedTags,
	}, nil
}

func DefaultBuildProfile() BuildProfile {
	profile, err := NewBuildProfile(
		runtime.GOOS,
		runtime.GOARCH,
		false,
		nil,
	)
	if err != nil {
		panic(err)
	}
	return profile
}

func (p BuildProfile) ToolchainVersion() string {
	return p.toolchainVersion
}

func (p BuildProfile) GOOS() string {
	return p.goos
}

func (p BuildProfile) GOARCH() string {
	return p.goarch
}

func (p BuildProfile) CgoEnabled() bool {
	return p.cgoEnabled
}

func (p BuildProfile) Tags() []string {
	return slices.Clone(p.tags)
}

func (p BuildProfile) Valid() bool {
	if !strings.HasPrefix(p.toolchainVersion, "go") ||
		!validBuildWord(p.toolchainVersion) ||
		!validBuildWord(p.goos) || !validBuildWord(p.goarch) ||
		!sort.StringsAreSorted(p.tags) {
		return false
	}
	for index, tag := range p.tags {
		if !validBuildWord(tag) ||
			index > 0 && tag == p.tags[index-1] {
			return false
		}
	}
	return true
}

func (p BuildProfile) IsZero() bool {
	return p.toolchainVersion == "" &&
		p.goos == "" && p.goarch == "" &&
		!p.cgoEnabled && len(p.tags) == 0
}

func (p BuildProfile) Environment(base []string) []string {
	result := make([]string, 0, len(base)+5)
	for _, entry := range base {
		if buildEnvironmentKey(entry) {
			continue
		}
		result = append(result, entry)
	}
	cgo := "0"
	if p.cgoEnabled {
		cgo = "1"
	}
	return append(
		result,
		"GOOS="+p.goos,
		"GOARCH="+p.goarch,
		"CGO_ENABLED="+cgo,
		"GOFLAGS=",
		"GOTOOLCHAIN=local",
	)
}

func (p BuildProfile) BuildFlags() []string {
	if len(p.tags) == 0 {
		return nil
	}
	return []string{"-tags=" + strings.Join(p.tags, ",")}
}

func HostEnvironment() []string {
	result := make([]string, 0, len(os.Environ())+5)
	for _, entry := range os.Environ() {
		if buildEnvironmentKey(entry) {
			continue
		}
		result = append(result, entry)
	}
	return append(
		result,
		"GOOS="+runtime.GOOS,
		"GOARCH="+runtime.GOARCH,
		"CGO_ENABLED=0",
		"GOFLAGS=",
		"GOTOOLCHAIN=local",
	)
}

func buildEnvironmentKey(entry string) bool {
	for _, name := range []string{
		"GOOS",
		"GOARCH",
		"CGO_ENABLED",
		"GOFLAGS",
		"GOTOOLCHAIN",
	} {
		if strings.HasPrefix(entry, name+"=") {
			return true
		}
	}
	return false
}

func validBuildWord(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if character >= 'a' && character <= 'z' ||
			character >= 'A' && character <= 'Z' ||
			character >= '0' && character <= '9' ||
			character == '_' || character == '.' {
			continue
		}
		return false
	}
	return true
}
