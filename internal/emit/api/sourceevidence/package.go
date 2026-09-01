package sourceevidence

import (
	"go/ast"
	"go/token"
)

type Package struct {
	fileSet       *token.FileSet
	byTokenFile   map[*token.File]File
	packagePath   string
	modulePath    string
	moduleVersion string
	ownerKind     string
	ownerKey      string
	outputPath    string
	programDigest string
}

func NewPackage(files []File) (Package, error) {
	if len(files) == 0 || !files[0].Valid() {
		return Package{}, &Error{Reason: "source-package evidence is incomplete"}
	}
	first := files[0]
	evidence := Package{
		fileSet:       first.fileSet,
		byTokenFile:   make(map[*token.File]File, len(files)),
		packagePath:   first.packagePath,
		modulePath:    first.modulePath,
		moduleVersion: first.moduleVersion,
		ownerKind:     first.ownerKind,
		ownerKey:      first.ownerKey,
		outputPath:    first.outputPath,
		programDigest: first.programDigest,
	}
	for _, file := range files {
		if !file.Valid() ||
			file.fileSet != evidence.fileSet ||
			file.packagePath != evidence.packagePath ||
			file.modulePath != evidence.modulePath ||
			file.moduleVersion != evidence.moduleVersion ||
			file.ownerKind != evidence.ownerKind ||
			file.ownerKey != evidence.ownerKey ||
			file.outputPath != evidence.outputPath ||
			file.programDigest != evidence.programDigest {
			return Package{}, &Error{Reason: "source-package file evidence is inconsistent"}
		}
		tokenFile := file.fileSet.File(file.syntax.Pos())
		if tokenFile == nil {
			return Package{}, &Error{Reason: "source-package token file is absent"}
		}
		if _, duplicate := evidence.byTokenFile[tokenFile]; duplicate {
			return Package{}, &Error{Reason: "source-package token file is duplicated"}
		}
		evidence.byTokenFile[tokenFile] = file
	}
	if !evidence.Valid() {
		return Package{}, &Error{Reason: "source-package evidence is invalid"}
	}
	return evidence, nil
}

func (e Package) Valid() bool {
	return e.fileSet != nil && len(e.byTokenFile) != 0 &&
		e.packagePath != "" && e.outputPath != "" && e.programDigest != "" &&
		validOwner(e.ownerKind, e.modulePath, e.moduleVersion, e.ownerKey)
}

func (e Package) Occurrence(source ast.Node) (Occurrence, error) {
	if !e.Valid() || source == nil {
		return Occurrence{}, &Error{Reason: "source-package occurrence is invalid"}
	}
	start := e.fileSet.File(source.Pos())
	end := e.fileSet.File(source.End())
	file, ok := e.byTokenFile[start]
	if start == nil || start != end || !ok {
		return Occurrence{}, &Error{
			Reason:             "source occurrence is outside the selected source package",
			SelectedSource:     e.packagePath,
			OccurrencePosition: e.fileSet.PositionFor(source.Pos(), false),
		}
	}
	return file.Occurrence(source)
}
