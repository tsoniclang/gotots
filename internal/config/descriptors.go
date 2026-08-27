package config

import "slices"

type OptionID uint8

const (
	OptionInvalid OptionID = iota
	OptionDistributionRoot
	OptionCGO
	OptionGOARCH
	OptionGOOS
	OptionTags
	OptionPackageImplementations
	OptionCallableImplementations
	OptionOutputDirectory
	OptionExternals
	OptionStandardLibrary
	OptionEvaluationOrder
	OptionIntegers
	OptionRootMode
	OptionPackage
	OptionSourceRoot
	OptionToolCacheRoot
	OptionGoExecutable
	OptionTSGoExecutable
)

type Descriptor struct {
	id       OptionID
	jsonPath string
	flag     string
	repeated bool
}

func (d Descriptor) ID() OptionID     { return d.id }
func (d Descriptor) JSONPath() string { return d.jsonPath }
func (d Descriptor) Flag() string     { return d.flag }
func (d Descriptor) Repeated() bool   { return d.repeated }

var descriptors = []Descriptor{
	{OptionDistributionRoot, "distribution.root", "distribution-root", false},
	{OptionCGO, "go.cgo", "cgo", false},
	{OptionGOARCH, "go.goarch", "goarch", false},
	{OptionGOOS, "go.goos", "goos", false},
	{OptionTags, "go.tags", "tag", true},
	{OptionCallableImplementations, "implementations.callables", "callable-implementation", true},
	{OptionPackageImplementations, "implementations.packages", "package-implementation", true},
	{OptionOutputDirectory, "output.directory", "output", false},
	{OptionExternals, "providers.externals", "externals", false},
	{OptionStandardLibrary, "providers.standardLibrary", "standard-library", false},
	{OptionEvaluationOrder, "semantics.evaluationOrder", "evaluation-order", false},
	{OptionIntegers, "semantics.integers", "integer", false},
	{OptionRootMode, "source.mode", "root-mode", false},
	{OptionPackage, "source.package", "package", false},
	{OptionSourceRoot, "source.root", "project-root", false},
	{OptionToolCacheRoot, "tools.cache", "tool-cache", false},
	{OptionGoExecutable, "tools.go", "go", false},
	{OptionTSGoExecutable, "tools.tsgo", "tsgo", false},
}

func Descriptors() []Descriptor {
	return slices.Clone(descriptors)
}
