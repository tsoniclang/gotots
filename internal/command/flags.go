package command

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strconv"

	"github.com/tsoniclang/gotots/internal/config"
)

type Invocation struct {
	configPath    string
	overrides     config.Overrides
	printResolved bool
}

func (i Invocation) ConfigPath() string { return i.configPath }
func (i Invocation) Overrides() config.Overrides {
	result := i.overrides
	result.BuildTags = slices.Clone(result.BuildTags)
	result.ImplementationBundles = slices.Clone(result.ImplementationBundles)
	return result
}
func (i Invocation) PrintResolvedConfig() bool { return i.printResolved }

type Error struct {
	Operation string
	Reason    string
}

func (e *Error) Error() string {
	return "run GoToTS command " + e.Operation + ": " + e.Reason
}

func ParseArguments(workingDirectory string, arguments []string) (Invocation, error) {
	if workingDirectory == "" {
		return Invocation{}, commandError("parse arguments", "working directory is empty")
	}
	absoluteWorkingDirectory, err := filepath.Abs(workingDirectory)
	if err != nil {
		return Invocation{}, commandError("parse arguments", err.Error())
	}
	if len(arguments) == 0 || arguments[0] != "build" {
		return Invocation{}, commandError("parse arguments", "expected build subcommand")
	}
	invocation := Invocation{
		configPath: filepath.Join(absoluteWorkingDirectory, "gotots.json"),
	}
	flags := flag.NewFlagSet("gotots build", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	configSelection := singlePathValue{
		base:   absoluteWorkingDirectory,
		target: &invocation.configPath,
	}
	flags.Var(&configSelection, "c", "project configuration path")
	flags.Var(&configSelection, "config", "project configuration path")
	for _, descriptor := range config.Descriptors() {
		if err := bindDescriptor(flags, descriptor, &invocation.overrides); err != nil {
			return Invocation{}, err
		}
	}
	flags.BoolVar(
		&invocation.printResolved,
		"print-resolved-config",
		false,
		"print the canonical resolved project",
	)
	if err := flags.Parse(arguments[1:]); err != nil {
		return Invocation{}, commandError("parse arguments", err.Error())
	}
	if flags.NArg() != 0 {
		return Invocation{}, commandError(
			"parse arguments",
			fmt.Sprintf("unexpected arguments %v", flags.Args()),
		)
	}
	return invocation, nil
}

func bindDescriptor(
	flags *flag.FlagSet,
	descriptor config.Descriptor,
	overrides *config.Overrides,
) error {
	description := "override " + descriptor.JSONPath()
	switch descriptor.ID() {
	case config.OptionDistributionRoot:
		flags.Var(newStringValue(&overrides.DistributionRoot), descriptor.Flag(), description)
	case config.OptionCGO:
		flags.Var(newBoolValue(&overrides.CGOEnabled), descriptor.Flag(), description)
	case config.OptionGOARCH:
		flags.Var(newStringValue(&overrides.GOARCH), descriptor.Flag(), description)
	case config.OptionGOOS:
		flags.Var(newStringValue(&overrides.GOOS), descriptor.Flag(), description)
	case config.OptionTags:
		flags.Var(&listValue{target: &overrides.BuildTags, selected: &overrides.BuildTagsSet}, descriptor.Flag(), description)
	case config.OptionImplementationBundles:
		flags.Var(&listValue{target: &overrides.ImplementationBundles, selected: &overrides.ImplementationSet}, descriptor.Flag(), description)
	case config.OptionOutputDirectory:
		flags.Var(newStringValue(&overrides.OutputDirectory), descriptor.Flag(), description)
	case config.OptionExternals:
		flags.Var(newBoolValue(&overrides.Externals), descriptor.Flag(), description)
	case config.OptionStandardLibrary:
		flags.Var(newBoolValue(&overrides.StandardLibrary), descriptor.Flag(), description)
	case config.OptionConcurrency:
		flags.Var(newStringValue(&overrides.ConcurrencySemantics), descriptor.Flag(), description)
	case config.OptionEvaluationOrder:
		flags.Var(newStringValue(&overrides.EvaluationOrder), descriptor.Flag(), description)
	case config.OptionIntegers:
		flags.Var(newStringValue(&overrides.IntegerRepresentation), descriptor.Flag(), description)
	case config.OptionRootMode:
		flags.Var(newStringValue(&overrides.RootMode), descriptor.Flag(), description)
	case config.OptionPackage:
		flags.Var(newStringValue(&overrides.PackagePattern), descriptor.Flag(), description)
	case config.OptionSourceRoot:
		flags.Var(newStringValue(&overrides.SourceRoot), descriptor.Flag(), description)
	case config.OptionGoExecutable:
		flags.Var(newStringValue(&overrides.GoExecutable), descriptor.Flag(), description)
	case config.OptionTSGoExecutable:
		flags.Var(newStringValue(&overrides.TSGoExecutable), descriptor.Flag(), description)
	case config.OptionToolCacheRoot:
		flags.Var(newStringValue(&overrides.ToolCacheRoot), descriptor.Flag(), description)
	default:
		return commandError(
			"bind flags",
			fmt.Sprintf("option %d has no binding", descriptor.ID()),
		)
	}
	return nil
}

type singlePathValue struct {
	base     string
	target   *string
	selected bool
}

func (v *singlePathValue) String() string { return "" }
func (v *singlePathValue) Set(value string) error {
	if v.selected {
		return fmt.Errorf("configuration path was selected more than once")
	}
	if value == "" {
		return fmt.Errorf("configuration path is empty")
	}
	v.selected = true
	if !filepath.IsAbs(value) {
		value = filepath.Join(v.base, value)
	}
	*v.target = filepath.Clean(value)
	return nil
}

type stringValue struct {
	target   **string
	selected bool
}

func newStringValue(target **string) *stringValue { return &stringValue{target: target} }
func (v *stringValue) String() string             { return "" }
func (v *stringValue) Set(value string) error {
	if v.selected {
		return fmt.Errorf("value was selected more than once")
	}
	v.selected = true
	*v.target = &value
	return nil
}

type boolValue struct {
	target   **bool
	selected bool
}

func newBoolValue(target **bool) *boolValue { return &boolValue{target: target} }
func (v *boolValue) String() string         { return "" }
func (*boolValue) IsBoolFlag() bool         { return true }
func (v *boolValue) Set(value string) error {
	if v.selected {
		return fmt.Errorf("value was selected more than once")
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return err
	}
	v.selected = true
	*v.target = &parsed
	return nil
}

type listValue struct {
	target   *[]string
	selected *bool
}

func (v *listValue) String() string { return "" }
func (v *listValue) Set(value string) error {
	if value == "" {
		return fmt.Errorf("list value is empty")
	}
	*v.selected = true
	*v.target = append(*v.target, value)
	return nil
}

func commandError(operation string, reason string) error {
	return &Error{Operation: operation, Reason: reason}
}
