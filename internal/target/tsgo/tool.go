package tsgo

import (
	"debug/buildinfo"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
)

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

func resolvePinnedTool(moduleDirectory string) (string, error) {
	moduleDirectory, err := filepath.Abs(moduleDirectory)
	if err != nil {
		return "", &ToolError{Path: moduleDirectory, Reason: err.Error()}
	}
	if info, err := os.Stat(filepath.Join(moduleDirectory, "go.mod")); err != nil || info.IsDir() {
		if err == nil {
			err = fmt.Errorf("not a regular file")
		}
		return "", &ToolError{Path: filepath.Join(moduleDirectory, "go.mod"), Reason: err.Error()}
	}
	goExecutable := filepath.Join(runtime.GOROOT(), "bin", "go")
	if runtime.GOOS == "windows" {
		goExecutable += ".exe"
	}
	command := exec.Command(goExecutable, "tool", "-n", filepath.Base(pinnedToolPackage))
	command.Dir = moduleDirectory
	command.Env = nativeToolEnvironment()
	output, err := command.CombinedOutput()
	if err != nil {
		return "", &ToolError{Path: goExecutable, Reason: fmt.Sprintf("%v: %s", err, strings.TrimSpace(string(output)))}
	}
	toolPath := strings.TrimSpace(string(output))
	if toolPath == "" || strings.ContainsAny(toolPath, "\r\n") || !filepath.IsAbs(toolPath) {
		return "", &ToolError{Path: goExecutable, Reason: fmt.Sprintf("invalid tool path %q", toolPath)}
	}
	if err := verifyPinnedTool(toolPath); err != nil {
		return "", err
	}
	return toolPath, nil
}

func nativeToolEnvironment() []string {
	return environmentcontract.HostEnvironment()
}

func verifyPinnedTool(toolPath string) error {
	info, err := buildinfo.ReadFile(toolPath)
	if err != nil {
		return &ToolError{Path: toolPath, Reason: err.Error()}
	}
	if info.Path != pinnedToolPackage {
		return &ToolError{Path: toolPath, Reason: fmt.Sprintf(
			"package %q, want %q",
			info.Path,
			pinnedToolPackage,
		)}
	}
	if info.Main.Path != pinnedToolModule || info.Main.Version != pinnedToolVersion {
		return &ToolError{Path: toolPath, Reason: fmt.Sprintf(
			"module %s@%s, want %s@%s",
			info.Main.Path,
			info.Main.Version,
			pinnedToolModule,
			pinnedToolVersion,
		)}
	}
	if info.Main.Replace != nil {
		return &ToolError{Path: toolPath, Reason: "tool module is replaced"}
	}
	return nil
}
