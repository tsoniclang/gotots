package tsgo

import (
	"context"
	"debug/buildinfo"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"slices"
	"strings"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/toolchain"
)

type Tool struct {
	executable  toolchain.Executable
	selectedGo  toolchain.Go
	identity    ToolIdentity
	environment []string
}

type ToolForm string

const (
	ToolFormInvalid ToolForm = ""
	ToolFormModule  ToolForm = "module"
	ToolFormSource  ToolForm = "source"
)

func (f ToolForm) Valid() bool {
	return f == ToolFormModule || f == ToolFormSource
}

type ToolIdentity struct {
	form          ToolForm
	packagePath   string
	modulePath    string
	moduleVersion string
	moduleSum     string
	revision      string
	goVersion     string
	digest        string
	selectedGo    toolchain.GoIdentity
}

type ToolError struct {
	Path   string
	Reason string
}

func (e *ToolError) Error() string {
	if e.Path == "" {
		return "resolve pinned TS-Go tool: " + e.Reason
	}
	return fmt.Sprintf("resolve pinned TS-Go tool %s: %s", e.Path, e.Reason)
}

func ResolveTool(
	selectedGo toolchain.Go,
	moduleDirectory string,
	explicitPath string,
) (Tool, error) {
	if !selectedGo.Valid() {
		return Tool{}, &ToolError{Reason: "selected Go tool is invalid"}
	}
	moduleDirectory, err := filepath.Abs(moduleDirectory)
	if err != nil {
		return Tool{}, &ToolError{Path: moduleDirectory, Reason: err.Error()}
	}
	selectedPath := explicitPath
	if selectedPath == "" {
		if info, statErr := os.Stat(filepath.Join(moduleDirectory, "go.mod")); statErr != nil || info.IsDir() {
			if statErr == nil {
				statErr = fmt.Errorf("not a regular file")
			}
			return Tool{}, &ToolError{
				Path:   filepath.Join(moduleDirectory, "go.mod"),
				Reason: statErr.Error(),
			}
		}
		output, outputErr := selectedGo.HostOutput(
			context.Background(),
			moduleDirectory,
			"tool",
			"-n",
			filepath.Base(pinnedToolPackage),
		)
		if outputErr != nil {
			return Tool{}, &ToolError{Path: selectedGo.Path(), Reason: outputErr.Error()}
		}
		selectedPath = strings.TrimSpace(string(output))
		if selectedPath == "" || strings.ContainsAny(selectedPath, "\r\n") || !filepath.IsAbs(selectedPath) {
			return Tool{}, &ToolError{Path: selectedGo.Path(), Reason: fmt.Sprintf("invalid tool path %q", selectedPath)}
		}
	}
	executableName := "tsgo"
	if runtime.GOOS == "windows" {
		executableName += ".exe"
	}
	executable, err := toolchain.SelectExecutable(
		selectedPath,
		"",
		executableName,
		selectedGo.CacheRoot(),
	)
	if err != nil {
		return Tool{}, err
	}
	build, err := inspectPinnedTool(executable.SealedPath())
	if err != nil {
		return Tool{}, err
	}
	if build.info.GoVersion != selectedGo.Version() {
		return Tool{}, &ToolError{Path: executable.Path(), Reason: fmt.Sprintf(
			"built with %s, want selected %s",
			build.info.GoVersion,
			selectedGo.Version(),
		)}
	}
	profile, err := environmentcontractForTool(selectedGo)
	if err != nil {
		return Tool{}, err
	}
	resolved := Tool{
		executable: executable, selectedGo: selectedGo,
		identity: ToolIdentity{
			form: build.form, packagePath: build.info.Path,
			modulePath: build.info.Main.Path, moduleVersion: build.info.Main.Version,
			moduleSum: build.info.Main.Sum, revision: build.revision,
			goVersion: build.info.GoVersion, digest: executable.Digest(),
			selectedGo: selectedGo.Identity(),
		},
		environment: selectedGo.Environment(profile),
	}
	if err := resolved.Verify(); err != nil {
		return Tool{}, err
	}
	return resolved, nil
}

func ResolveDefaultTool(moduleDirectory string) (Tool, error) {
	selectedGo, err := toolchain.ResolveGo(
		"",
		filepath.Join(moduleDirectory, ".temp", "cache", "toolchain"),
	)
	if err != nil {
		return Tool{}, err
	}
	return ResolveTool(selectedGo, moduleDirectory, "")
}

func StartClient(moduleDirectory string, workingDirectory string) (*Client, error) {
	tool, err := ResolveDefaultTool(moduleDirectory)
	if err != nil {
		return nil, err
	}
	return StartClientWithTool(tool, workingDirectory)
}

func (t Tool) Path() string           { return t.executable.Path() }
func (t Tool) Identity() ToolIdentity { return t.identity }
func (t Tool) Valid() bool {
	return t.executable.Valid() && t.selectedGo.Valid() && t.identity.Valid() &&
		t.identity.SelectedGo() == t.selectedGo.Identity() && len(t.environment) != 0
}

func (t Tool) Verify() error {
	if !t.Valid() {
		return &ToolError{Path: t.Path(), Reason: "selection is invalid"}
	}
	return errors.Join(t.selectedGo.Verify(), t.executable.Verify())
}

func (i ToolIdentity) PackagePath() string              { return i.packagePath }
func (i ToolIdentity) Form() ToolForm                   { return i.form }
func (i ToolIdentity) ModulePath() string               { return i.modulePath }
func (i ToolIdentity) ModuleVersion() string            { return i.moduleVersion }
func (i ToolIdentity) ModuleSum() string                { return i.moduleSum }
func (i ToolIdentity) Revision() string                 { return i.revision }
func (i ToolIdentity) GoVersion() string                { return i.goVersion }
func (i ToolIdentity) Digest() string                   { return i.digest }
func (i ToolIdentity) SelectedGo() toolchain.GoIdentity { return i.selectedGo }
func (i ToolIdentity) Valid() bool {
	if !i.form.Valid() || i.packagePath != pinnedToolPackage ||
		i.modulePath != pinnedToolModule || i.revision != pinnedSchemaRevision ||
		i.goVersion == "" || len(i.digest) != 64 || !i.selectedGo.Valid() {
		return false
	}
	if i.form == ToolFormModule {
		return i.moduleVersion == pinnedToolVersion && i.moduleSum == pinnedToolSum
	}
	return i.moduleVersion == "(devel)" && i.moduleSum == ""
}
func (i ToolIdentity) String() string {
	if !i.Valid() {
		return ""
	}
	return "tsgo:" + string(i.form) + ":" + i.packagePath + "@" +
		i.moduleVersion + ":sum=" + i.moduleSum + ":revision=" + i.revision +
		":built=" + i.goVersion + ":selected=" + i.selectedGo.String() +
		":sha256=" + i.digest
}

func (t Tool) command(ctx context.Context, arguments ...string) (*exec.Cmd, error) {
	if err := t.Verify(); err != nil {
		return nil, err
	}
	command, err := t.executable.CommandContext(ctx, arguments...)
	if err != nil {
		return nil, err
	}
	command.Env = slices.Clone(t.environment)
	return command, nil
}

func environmentcontractForTool(selected toolchain.Go) (environmentcontract.BuildProfile, error) {
	return environmentcontract.NewBuildProfileForToolchain(
		selected.Version(),
		selected.DefaultGOOS(),
		selected.DefaultGOARCH(),
		false,
		nil,
	)
}

func verifyPinnedTool(toolPath string) error {
	_, err := inspectPinnedTool(toolPath)
	return err
}

type pinnedBuild struct {
	form     ToolForm
	revision string
	info     *buildinfo.BuildInfo
}

func inspectPinnedTool(toolPath string) (pinnedBuild, error) {
	info, err := buildinfo.ReadFile(toolPath)
	if err != nil {
		return pinnedBuild{}, &ToolError{Path: toolPath, Reason: err.Error()}
	}
	return classifyPinnedBuild(toolPath, info)
}

func classifyPinnedBuild(toolPath string, info *debug.BuildInfo) (pinnedBuild, error) {
	if info == nil {
		return pinnedBuild{}, &ToolError{Path: toolPath, Reason: "build information is absent"}
	}
	if info.Path != pinnedToolPackage {
		return pinnedBuild{}, &ToolError{Path: toolPath, Reason: fmt.Sprintf(
			"package %q, want %q",
			info.Path,
			pinnedToolPackage,
		)}
	}
	if info.Main.Path != pinnedToolModule {
		return pinnedBuild{}, &ToolError{Path: toolPath, Reason: fmt.Sprintf(
			"module %s, want %s", info.Main.Path, pinnedToolModule,
		)}
	}
	if info.Main.Replace != nil {
		return pinnedBuild{}, &ToolError{Path: toolPath, Reason: "tool module is replaced"}
	}
	if info.Main.Version == pinnedToolVersion && info.Main.Sum == pinnedToolSum {
		return pinnedBuild{
			form: ToolFormModule, revision: pinnedSchemaRevision, info: info,
		}, nil
	}
	if info.Main.Version != "(devel)" || info.Main.Sum != "" {
		return pinnedBuild{}, &ToolError{Path: toolPath, Reason: fmt.Sprintf(
			"module %s@%s %s is not a pinned module or source build",
			info.Main.Path, info.Main.Version, info.Main.Sum,
		)}
	}
	settings := make(map[string]string, len(info.Settings))
	for _, setting := range info.Settings {
		if _, duplicate := settings[setting.Key]; duplicate {
			return pinnedBuild{}, &ToolError{Path: toolPath, Reason: "build setting " + setting.Key + " is duplicated"}
		}
		settings[setting.Key] = setting.Value
	}
	if settings["vcs"] != "git" || settings["vcs.revision"] != pinnedSchemaRevision ||
		settings["vcs.modified"] != "false" {
		return pinnedBuild{}, &ToolError{
			Path:   toolPath,
			Reason: "development build lacks the exact clean pinned VCS identity",
		}
	}
	return pinnedBuild{form: ToolFormSource, revision: pinnedSchemaRevision, info: info}, nil
}
