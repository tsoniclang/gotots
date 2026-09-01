package command

import (
	"fmt"
	"os"
	"path/filepath"
)

type outputFailure struct {
	cause    error
	artifact string
}

func (e *outputFailure) Error() string {
	return fmt.Sprintf("%v; failure artifact preserved at %q", e.cause, e.artifact)
}

func (e *outputFailure) Unwrap() error {
	return e.cause
}

func writeOutputTransaction(
	target string,
	write func(string) (int, error),
) (int, error) {
	parent := filepath.Dir(target)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return 0, commandError("create output parent", err.Error())
	}
	staging, err := os.MkdirTemp(
		parent,
		"."+filepath.Base(target)+".gotots-stage-",
	)
	if err != nil {
		return 0, commandError("create output staging", err.Error())
	}
	files, err := write(staging)
	if err != nil {
		return 0, preserveOutputFailure(err, staging)
	}
	if err := installOutputDirectory(staging, target); err != nil {
		return 0, preserveOutputFailure(err, staging)
	}
	return files, nil
}

func preserveOutputFailure(cause error, staging string) error {
	if _, err := os.Lstat(staging); err != nil {
		return cause
	}
	return &outputFailure{cause: cause, artifact: staging}
}

func installOutputDirectory(staging string, target string) error {
	if _, err := os.Lstat(target); os.IsNotExist(err) {
		if err := os.Rename(staging, target); err != nil {
			return commandError("install output", err.Error())
		}
		return nil
	} else if err != nil {
		return commandError("inspect output", err.Error())
	}
	backup := staging + "-previous"
	if err := os.Rename(target, backup); err != nil {
		return commandError("stage previous output", err.Error())
	}
	if err := os.Rename(staging, target); err != nil {
		if rollbackErr := os.Rename(backup, target); rollbackErr != nil {
			return commandError(
				"install output",
				fmt.Sprintf("%v; restoring previous output: %v", err, rollbackErr),
			)
		}
		return commandError("install output", err.Error())
	}
	if err := os.RemoveAll(backup); err != nil {
		return commandError("remove previous output", err.Error())
	}
	return nil
}
