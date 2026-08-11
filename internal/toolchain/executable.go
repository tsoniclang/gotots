package toolchain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

type Executable struct {
	selectedPath string
	sealedPath   string
	digest       string
}

type ExecutableError struct {
	Operation string
	Path      string
	Reason    string
}

func (e *ExecutableError) Error() string {
	if e.Path == "" {
		return "resolve executable " + e.Operation + ": " + e.Reason
	}
	return fmt.Sprintf("resolve executable %s %q: %s", e.Operation, e.Path, e.Reason)
}

func SelectExecutable(
	selection string,
	defaultName string,
	sealedName string,
	cacheRoot string,
) (Executable, error) {
	if cacheRoot == "" {
		return Executable{}, executableError("select cache", "", "path is empty")
	}
	cacheRoot, err := filepath.Abs(cacheRoot)
	if err != nil {
		return Executable{}, executableError("select cache", cacheRoot, err.Error())
	}
	selectedPath := selection
	if selectedPath == "" {
		if defaultName == "" {
			return Executable{}, executableError("select", "", "path is empty")
		}
		resolved, err := exec.LookPath(defaultName)
		if err != nil {
			return Executable{}, executableError("select", defaultName, err.Error())
		}
		selectedPath = resolved
	}
	absolute, err := filepath.Abs(selectedPath)
	if err != nil {
		return Executable{}, executableError("select", selectedPath, err.Error())
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return Executable{}, executableError("inspect", absolute, err.Error())
	}
	if !info.Mode().IsRegular() || runtime.GOOS != "windows" && info.Mode().Perm()&0o111 == 0 {
		return Executable{}, executableError("inspect", absolute, "path is not an executable regular file")
	}
	digest, err := fileDigest(absolute)
	if err != nil {
		return Executable{}, executableError("fingerprint", absolute, err.Error())
	}
	sealedPath, err := sealExecutable(cacheRoot, absolute, digest, sealedName)
	if err != nil {
		return Executable{}, err
	}
	return Executable{selectedPath: absolute, sealedPath: sealedPath, digest: digest}, nil
}

func (e Executable) Path() string   { return e.selectedPath }
func (e Executable) Digest() string { return e.digest }

func (e Executable) Valid() bool {
	return e.selectedPath != "" && e.sealedPath != "" && len(e.digest) == sha256.Size*2
}

func (e Executable) Verify() error {
	if !e.Valid() {
		return executableError("verify", e.selectedPath, "selection is invalid")
	}
	digest, err := fileDigest(e.sealedPath)
	if err != nil {
		return executableError("verify", e.sealedPath, err.Error())
	}
	if digest != e.digest {
		return executableError("verify", e.sealedPath, "sealed bytes drifted")
	}
	return nil
}

func (e Executable) CommandContext(ctx context.Context, arguments ...string) (*exec.Cmd, error) {
	if ctx == nil {
		return nil, executableError("execute", e.selectedPath, "context is nil")
	}
	if err := e.Verify(); err != nil {
		return nil, err
	}
	return exec.CommandContext(ctx, e.sealedPath, arguments...), nil
}

func (e Executable) SelectedOutput(
	ctx context.Context,
	environment []string,
	directory string,
	arguments ...string,
) ([]byte, error) {
	if ctx == nil {
		return nil, executableError("execute selected", e.selectedPath, "context is nil")
	}
	if err := e.verifySelected(); err != nil {
		return nil, err
	}
	command := exec.CommandContext(ctx, e.selectedPath, arguments...)
	command.Env = append([]string(nil), environment...)
	command.Dir = directory
	payload, runErr := command.Output()
	verifyErr := e.verifySelected()
	if runErr != nil || verifyErr != nil {
		var executionErr error
		if runErr != nil {
			executionErr = executableError("execute selected", e.selectedPath, runErr.Error())
		}
		return payload, errors.Join(executionErr, verifyErr)
	}
	return payload, nil
}

func (e Executable) verifySelected() error {
	if !e.Valid() {
		return executableError("verify selected", e.selectedPath, "selection is invalid")
	}
	digest, err := fileDigest(e.selectedPath)
	if err != nil {
		return executableError("verify selected", e.selectedPath, err.Error())
	}
	if digest != e.digest {
		return executableError("verify selected", e.selectedPath, "selected bytes drifted")
	}
	return nil
}

func (e Executable) Directory() string {
	if !e.Valid() {
		return ""
	}
	return filepath.Dir(e.sealedPath)
}

func (e Executable) SealedPath() string {
	if !e.Valid() {
		return ""
	}
	return e.sealedPath
}

func sealExecutable(cacheRoot string, source string, digest string, name string) (string, error) {
	if name == "" {
		name = filepath.Base(source)
	}
	if filepath.Base(name) != name || name == "." || name == string(filepath.Separator) {
		return "", executableError("seal", name, "name is invalid")
	}
	root := filepath.Join(cacheRoot, "executables", digest)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", executableError("seal", root, err.Error())
	}
	target := filepath.Join(root, name)
	if _, err := os.Stat(target); err == nil {
		if got, digestErr := fileDigest(target); digestErr != nil || got != digest {
			if digestErr == nil {
				digestErr = fmt.Errorf("content digest differs")
			}
			return "", executableError("seal", target, digestErr.Error())
		}
		return target, nil
	} else if !os.IsNotExist(err) {
		return "", executableError("seal", target, err.Error())
	}
	temporary, err := os.CreateTemp(root, ".candidate-")
	if err != nil {
		return "", executableError("seal", root, err.Error())
	}
	temporaryPath := temporary.Name()
	failed := true
	defer func() {
		_ = temporary.Close()
		if failed {
			_ = os.Remove(temporaryPath)
		}
	}()
	input, err := os.Open(source)
	if err != nil {
		return "", executableError("seal", source, err.Error())
	}
	_, copyErr := io.Copy(temporary, input)
	closeInputErr := input.Close()
	if copyErr != nil {
		return "", executableError("seal", source, copyErr.Error())
	}
	if closeInputErr != nil {
		return "", executableError("seal", source, closeInputErr.Error())
	}
	if err := temporary.Sync(); err != nil {
		return "", executableError("seal", temporaryPath, err.Error())
	}
	if err := temporary.Chmod(0o555); err != nil {
		return "", executableError("seal", temporaryPath, err.Error())
	}
	if err := temporary.Close(); err != nil {
		return "", executableError("seal", temporaryPath, err.Error())
	}
	sealedDigest, err := fileDigest(temporaryPath)
	if err != nil {
		return "", executableError("seal", temporaryPath, err.Error())
	}
	if sealedDigest != digest {
		return "", executableError("seal", source, "source bytes changed while sealing")
	}
	if err := os.Rename(temporaryPath, target); err != nil {
		return "", executableError("seal", target, err.Error())
	}
	publishedDigest, err := fileDigest(target)
	if err != nil {
		_ = os.Remove(target)
		return "", executableError("seal", target, err.Error())
	}
	if publishedDigest != digest {
		_ = os.Remove(target)
		return "", executableError("seal", target, "published bytes differ")
	}
	failed = false
	return target, nil
}

func fileDigest(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	_, copyErr := io.Copy(hash, file)
	closeErr := file.Close()
	if copyErr != nil {
		return "", copyErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func executableError(operation string, path string, reason string) error {
	return &ExecutableError{Operation: operation, Path: path, Reason: reason}
}
