package tsgo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
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

func CompileWithTool(
	ctx context.Context,
	tool Tool,
	workingDirectory string,
	arguments []string,
) error {
	if ctx == nil {
		return &CompilerError{Reason: "context is nil"}
	}
	command, err := tool.command(ctx, arguments...)
	if err != nil {
		return err
	}
	command.Dir = workingDirectory
	output, runErr := command.CombinedOutput()
	verifyErr := tool.Verify()
	diagnostics := strings.TrimSpace(string(output))
	if runErr == nil && verifyErr == nil && diagnostics == "" {
		return nil
	}
	if verifyErr != nil {
		return errors.Join(runErr, verifyErr)
	}
	reason := "diagnostics reported with successful process status"
	if runErr != nil {
		reason = runErr.Error()
		if ctxErr := ctx.Err(); ctxErr != nil {
			reason = fmt.Sprintf("%v: %v", runErr, ctxErr)
		}
	}
	return &CompilerError{
		Path:        tool.Path(),
		Reason:      reason,
		Diagnostics: diagnostics,
	}
}

func Compile(
	ctx context.Context,
	moduleDirectory string,
	workingDirectory string,
	arguments []string,
) error {
	tool, err := ResolveDefaultTool(moduleDirectory)
	if err != nil {
		return err
	}
	return CompileWithTool(ctx, tool, workingDirectory, arguments)
}

func EncodeStrictProjectConfig(
	distributionRoot string,
	projectRoot string,
) ([]byte, error) {
	if distributionRoot == "" || projectRoot == "" {
		return nil, fmt.Errorf("encode TS-Go project config: root is absent")
	}
	providerPath := func(parts ...string) (string, error) {
		target := filepath.Join(append([]string{distributionRoot}, parts...)...)
		relative, err := filepath.Rel(projectRoot, target)
		if err != nil {
			return "", err
		}
		result := filepath.ToSlash(relative)
		if result != "." && len(result) > 0 && result[0] != '.' {
			result = "./" + result
		}
		return result, nil
	}
	standardLibrary, err := providerPath("gostdlib", "dist", "src", "*.d.ts")
	if err != nil {
		return nil, fmt.Errorf("encode TS-Go project config: %w", err)
	}
	externals, err := providerPath("externals", "dist", "src", "*.d.ts")
	if err != nil {
		return nil, fmt.Errorf("encode TS-Go project config: %w", err)
	}
	document := map[string]any{
		"compilerOptions": map[string]any{
			"target":           "ES2022",
			"module":           "NodeNext",
			"moduleResolution": "NodeNext",
			"paths": map[string][]string{
				"@gotots/runtime/*.js":   {"./runtime/*.ts"},
				"@gotots/gostdlib/*.js":  {standardLibrary},
				"@gotots/externals/*.js": {externals},
			},
			"strict":                           true,
			"exactOptionalPropertyTypes":       true,
			"noUncheckedIndexedAccess":         true,
			"noImplicitOverride":               true,
			"noFallthroughCasesInSwitch":       true,
			"forceConsistentCasingInFileNames": true,
			"skipLibCheck":                     false,
			"types":                            []string{},
			"noEmit":                           true,
		},
		"include": []string{"**/*.ts"},
		"exclude": []string{"node_modules", "out"},
	}
	payload, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode TS-Go project config: %w", err)
	}
	return append(payload, '\n'), nil
}
