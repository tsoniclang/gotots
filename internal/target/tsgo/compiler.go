package tsgo

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type CompilerError struct {
	Path        string
	Reason      string
	Diagnostics string
}

func (e *CompilerError) Error() string {
	message := "TS-Go compiler"
	if e.Path != "" {
		message += " " + e.Path
	}
	if e.Reason != "" {
		message += ": " + e.Reason
	}
	if e.Diagnostics != "" {
		message += "\n" + e.Diagnostics
	}
	return message
}

func Compile(
	ctx context.Context,
	moduleDirectory string,
	workingDirectory string,
	arguments []string,
) error {
	if ctx == nil {
		return &CompilerError{Reason: "context is nil"}
	}
	toolPath, err := resolvePinnedTool(moduleDirectory)
	if err != nil {
		return err
	}
	command := exec.CommandContext(ctx, toolPath, arguments...)
	command.Dir = workingDirectory
	command.Env = nativeToolEnvironment()
	output, runErr := command.CombinedOutput()
	diagnostics := strings.TrimSpace(string(output))
	if runErr == nil && diagnostics == "" {
		return nil
	}
	reason := "diagnostics reported with successful process status"
	if runErr != nil {
		reason = runErr.Error()
		if ctxErr := ctx.Err(); ctxErr != nil {
			reason = fmt.Sprintf("%v: %v", runErr, ctxErr)
		}
	}
	return &CompilerError{
		Path:        toolPath,
		Reason:      reason,
		Diagnostics: diagnostics,
	}
}
