package config

import (
	"slices"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
	"github.com/tsoniclang/gotots/internal/toolchain"
)

const SchemaVersion = 1

type RootMode string

const (
	RootModeInvalid  RootMode = ""
	RootModeMain     RootMode = "main"
	RootModeExported RootMode = "exported"
	RootModePackage  RootMode = "package"
	RootModeAll      RootMode = "all"
)

func (m RootMode) Valid() bool {
	return m == RootModeMain || m == RootModeExported ||
		m == RootModePackage || m == RootModeAll
}

type Overrides struct {
	DistributionRoot      *string
	GoExecutable          *string
	TSGoExecutable        *string
	ToolCacheRoot         *string
	SourceRoot            *string
	PackagePattern        *string
	RootMode              *string
	GOOS                  *string
	GOARCH                *string
	CGOEnabled            *bool
	BuildTags             []string
	BuildTagsSet          bool
	IntegerRepresentation *string
	EvaluationOrder       *string
	ConcurrencySemantics  *string
	StandardLibrary       *bool
	Externals             *bool
	ImplementationBundles []string
	ImplementationSet     bool
	OutputDirectory       *string
}

type Request struct {
	ConfigPath string
	Overrides  Overrides
}

type Project struct {
	configPath            string
	distributionRoot      string
	sourceRoot            string
	packagePattern        string
	rootMode              RootMode
	buildProfile          load.BuildProfile
	goTool                toolchain.Go
	tsgoTool              tsgo.Tool
	toolCacheRoot         string
	integer               emit.IntegerRepresentation
	evaluation            emit.EvaluationOrder
	concurrency           emit.ConcurrencySemantics
	standardLibrary       bool
	externals             bool
	implementationBundles []string
	outputDirectory       string
}

func (p Project) ConfigPath() string                                { return p.configPath }
func (p Project) DistributionRoot() string                          { return p.distributionRoot }
func (p Project) SourceRoot() string                                { return p.sourceRoot }
func (p Project) PackagePattern() string                            { return p.packagePattern }
func (p Project) RootMode() RootMode                                { return p.rootMode }
func (p Project) BuildProfile() load.BuildProfile                   { return p.buildProfile }
func (p Project) GoTool() toolchain.Go                              { return p.goTool }
func (p Project) TSGoTool() tsgo.Tool                               { return p.tsgoTool }
func (p Project) ToolCacheRoot() string                             { return p.toolCacheRoot }
func (p Project) IntegerRepresentation() emit.IntegerRepresentation { return p.integer }
func (p Project) EvaluationOrder() emit.EvaluationOrder             { return p.evaluation }
func (p Project) ConcurrencySemantics() emit.ConcurrencySemantics   { return p.concurrency }
func (p Project) StandardLibraryEnabled() bool                      { return p.standardLibrary }
func (p Project) ExternalsEnabled() bool                            { return p.externals }
func (p Project) ImplementationBundles() []string {
	return slices.Clone(p.implementationBundles)
}
func (p Project) OutputDirectory() string { return p.outputDirectory }

type EvidenceDigests struct {
	Source                string
	SourceImplementations string
	StandardLibrary       string
	Externals             string
}

type Error struct {
	Operation string
	Subject   string
	Reason    string
}

func (e *Error) Error() string {
	if e.Subject == "" {
		return "resolve GoToTS project " + e.Operation + ": " + e.Reason
	}
	return "resolve GoToTS project " + e.Operation + " " +
		quote(e.Subject) + ": " + e.Reason
}

func quote(value string) string {
	return `"` + value + `"`
}
