package environmentcontract

import (
	"go/types"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tsoniclang/gotots/internal/load"
)

func SourceLocation(sourcePackage *load.Package, object types.Object) string {
	if sourcePackage == nil || sourcePackage.FileSet() == nil ||
		object == nil || !object.Pos().IsValid() {
		return ""
	}
	position := sourcePackage.FileSet().PositionFor(object.Pos(), false)
	if !position.IsValid() {
		return ""
	}
	root := ""
	switch sourcePackage.Kind() {
	case load.PackageStandardLibraryContract:
		root = filepath.Join(sourcePackage.Program().GoTool().Root(), "src")
	case load.PackageExternalContract:
		root = sourcePackage.SourceRoot()
	}
	if root == "" {
		return ""
	}
	relative, err := filepath.Rel(root, position.Filename)
	if err != nil || relative == "." || relative == ".." ||
		strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return ""
	}
	return filepath.ToSlash(relative) + ":" +
		strconv.Itoa(position.Line) + ":" + strconv.Itoa(position.Column)
}
