package output

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/tsoniclang/gotots/internal/load"
)

type PathError struct {
	Source string
	Reason string
}

func (e *PathError) Error() string {
	if e.Source == "" {
		return "resolve target path: " + e.Reason
	}
	return fmt.Sprintf("resolve target path for %q: %s", e.Source, e.Reason)
}

func SourcePath(sourcePackage *load.Package, sourceFile load.File) (string, error) {
	if sourcePackage == nil {
		return "", &PathError{Reason: "source package is nil"}
	}
	if sourcePackage.ModulePath() == "" {
		return "", &PathError{
			Source: sourcePackage.Path(),
			Reason: "package has no source module identity",
		}
	}
	ownedFile, ok := sourcePackage.FileForSyntax(sourceFile.Syntax())
	if !ok || ownedFile.Path() != sourceFile.Path() {
		return "", &PathError{
			Source: sourceFile.Path(),
			Reason: "file does not belong to the source package",
		}
	}
	extension := filepath.Ext(sourceFile.Path())
	baseName := strings.TrimSuffix(filepath.Base(sourceFile.Path()), extension)
	if extension != ".go" || baseName == "" {
		return "", &PathError{
			Source: sourceFile.Path(),
			Reason: "source file is not a named Go file",
		}
	}
	packagePath, err := moduleRelativePackage(sourcePackage)
	if err != nil {
		return "", err
	}
	return path.Join(
		"modules",
		moduleKey(sourcePackage.ModulePath(), sourcePackage.ModuleVersion()),
		packagePath,
		baseName+".ts",
	), nil
}

func ModuleSpecifier(fromSourcePath string, toSourcePath string) (string, error) {
	if err := validateSourcePath(fromSourcePath); err != nil {
		return "", err
	}
	if err := validateSourcePath(toSourcePath); err != nil {
		return "", err
	}
	if fromSourcePath == toSourcePath {
		return "", &PathError{
			Source: fromSourcePath,
			Reason: "source module cannot import itself",
		}
	}
	relative, err := filepath.Rel(
		filepath.FromSlash(path.Dir(fromSourcePath)),
		filepath.FromSlash(toSourcePath),
	)
	if err != nil {
		return "", &PathError{
			Source: toSourcePath,
			Reason: err.Error(),
		}
	}
	relative = filepath.ToSlash(relative)
	relative = strings.TrimSuffix(relative, ".ts") + ".js"
	if !strings.HasPrefix(relative, ".") {
		relative = "./" + relative
	}
	return relative, nil
}

func moduleRelativePackage(sourcePackage *load.Package) (string, error) {
	if sourcePackage.Path() == sourcePackage.ModulePath() {
		return "_root", nil
	}
	relative, ok := strings.CutPrefix(
		sourcePackage.Path(),
		sourcePackage.ModulePath()+"/",
	)
	if !ok || relative == "" || path.Clean(relative) != relative ||
		strings.HasPrefix(relative, "../") {
		return "", &PathError{
			Source: sourcePackage.Path(),
			Reason: "package path is outside its semantic module",
		}
	}
	return relative, nil
}

func moduleKey(modulePath string, moduleVersion string) string {
	digest := sha256.Sum256([]byte(modulePath + "\x00" + moduleVersion))
	return hex.EncodeToString(digest[:])
}

func validateSourcePath(sourcePath string) error {
	if sourcePath == "" || path.IsAbs(sourcePath) ||
		path.Clean(sourcePath) != sourcePath ||
		path.Ext(sourcePath) != ".ts" ||
		strings.HasPrefix(sourcePath, "../") {
		return &PathError{
			Source: sourcePath,
			Reason: "target source path is not canonical",
		}
	}
	return nil
}
