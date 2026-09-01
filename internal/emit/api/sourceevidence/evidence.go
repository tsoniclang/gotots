package sourceevidence

import (
	"fmt"
	"go/ast"
	"go/token"
)

type Error struct {
	Reason             string
	SelectedSource     string
	OccurrencePosition token.Position
}

func (e *Error) Error() string {
	if e.SelectedSource != "" || e.OccurrencePosition.IsValid() {
		return fmt.Sprintf(
			"%s: selected source %q, occurrence %s",
			e.Reason,
			e.SelectedSource,
			e.OccurrencePosition.String(),
		)
	}
	return e.Reason
}

type File struct {
	fileSet          *token.FileSet
	syntax           *ast.File
	packagePath      string
	modulePath       string
	moduleVersion    string
	ownerKind        string
	ownerKey         string
	outputPath       string
	sourceIdentity   string
	sourceFileDigest string
	programDigest    string
	goVersion        string
}

type Occurrence struct {
	packagePath      string
	modulePath       string
	moduleVersion    string
	ownerKind        string
	ownerKey         string
	outputPath       string
	sourceIdentity   string
	sourceFileDigest string
	programDigest    string
	start            int
	end              int
	goVersion        string
}

func NewFile(
	fileSet *token.FileSet,
	syntax *ast.File,
	packagePath string,
	modulePath string,
	moduleVersion string,
	ownerKind string,
	ownerKey string,
	outputPath string,
	sourceIdentity string,
	sourceFileDigest string,
	programDigest string,
	goVersion string,
) (File, error) {
	evidence := File{
		fileSet:          fileSet,
		syntax:           syntax,
		packagePath:      packagePath,
		modulePath:       modulePath,
		moduleVersion:    moduleVersion,
		ownerKind:        ownerKind,
		ownerKey:         ownerKey,
		outputPath:       outputPath,
		sourceIdentity:   sourceIdentity,
		sourceFileDigest: sourceFileDigest,
		programDigest:    programDigest,
		goVersion:        goVersion,
	}
	if !evidence.Valid() {
		return File{}, &Error{Reason: "source-file evidence is incomplete"}
	}
	return evidence, nil
}

func (e File) Valid() bool {
	if e.fileSet == nil || e.syntax == nil ||
		e.packagePath == "" || e.outputPath == "" ||
		e.sourceIdentity == "" || e.sourceFileDigest == "" ||
		e.programDigest == "" || e.goVersion == "" ||
		!validOwner(e.ownerKind, e.modulePath, e.moduleVersion, e.ownerKey) {
		return false
	}
	start := e.fileSet.File(e.syntax.Pos())
	end := e.fileSet.File(e.syntax.End())
	return start != nil && start == end
}

func (e File) Occurrence(source ast.Node) (Occurrence, error) {
	if !e.Valid() || source == nil ||
		source.Pos() < e.syntax.Pos() || source.End() > e.syntax.End() {
		position := token.Position{}
		if e.fileSet != nil && source != nil {
			position = e.fileSet.PositionFor(source.Pos(), false)
		}
		return Occurrence{}, &Error{
			Reason:             "source occurrence is outside the selected source file",
			SelectedSource:     e.sourceIdentity,
			OccurrencePosition: position,
		}
	}
	file := e.fileSet.File(source.Pos())
	if file == nil || file != e.fileSet.File(source.End()) ||
		file != e.fileSet.File(e.syntax.Pos()) {
		return Occurrence{}, &Error{
			Reason: "source occurrence is not file-bounded",
		}
	}
	occurrence := Occurrence{
		packagePath:      e.packagePath,
		modulePath:       e.modulePath,
		moduleVersion:    e.moduleVersion,
		ownerKind:        e.ownerKind,
		ownerKey:         e.ownerKey,
		outputPath:       e.outputPath,
		sourceIdentity:   e.sourceIdentity,
		sourceFileDigest: e.sourceFileDigest,
		programDigest:    e.programDigest,
		start:            file.Offset(source.Pos()),
		end:              file.Offset(source.End()),
		goVersion:        e.goVersion,
	}
	if !occurrence.Valid() {
		return Occurrence{}, &Error{
			Reason: "source occurrence evidence is incomplete",
		}
	}
	return occurrence, nil
}

func (o Occurrence) Valid() bool {
	return o.packagePath != "" && o.outputPath != "" &&
		o.sourceIdentity != "" && o.sourceFileDigest != "" &&
		o.programDigest != "" && o.start >= 0 && o.end > o.start &&
		o.goVersion != "" && validOwner(
		o.ownerKind,
		o.modulePath,
		o.moduleVersion,
		o.ownerKey,
	)
}

func (o Occurrence) PackagePath() string      { return o.packagePath }
func (o Occurrence) ModulePath() string       { return o.modulePath }
func (o Occurrence) ModuleVersion() string    { return o.moduleVersion }
func (o Occurrence) OwnerKind() string        { return o.ownerKind }
func (o Occurrence) OwnerKey() string         { return o.ownerKey }
func (o Occurrence) OutputPath() string       { return o.outputPath }
func (o Occurrence) SourceIdentity() string   { return o.sourceIdentity }
func (o Occurrence) SourceFileDigest() string { return o.sourceFileDigest }
func (o Occurrence) ProgramDigest() string    { return o.programDigest }
func (o Occurrence) Start() int               { return o.start }
func (o Occurrence) End() int                 { return o.end }
func (o Occurrence) GoVersion() string        { return o.goVersion }

func validOwner(kind string, modulePath string, moduleVersion string, key string) bool {
	switch kind {
	case "module":
		return modulePath != "" && key != ""
	case "workspace":
		return key != "" && (modulePath != "" || moduleVersion == "")
	case "standard-library", "toolchain", "language":
		return modulePath == "" && moduleVersion == "" && key != ""
	default:
		return false
	}
}
