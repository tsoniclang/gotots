package sourcefact

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/api/sourceevidence"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type DeclarationOrigin struct {
	kind             string
	packagePath      string
	modulePath       string
	moduleVersion    string
	ownerKind        string
	contractKey      string
	outputPath       string
	sourceIdentity   string
	sourceFileDigest string
	programDigest    string
	sourceStart      int
	sourceEnd        int
	goVersion        string
	sourceLocation   string
	environmentBasis string
}

func (o DeclarationOrigin) WithEnvironmentBasis(
	identity string,
) (DeclarationOrigin, error) {
	if !o.Valid() || identity == "" || o.environmentBasis != "" {
		return DeclarationOrigin{}, &Error{Reason: "environment basis is invalid"}
	}
	o.environmentBasis = identity
	return o, nil
}

func (o DeclarationOrigin) EnvironmentBasis() (string, bool) {
	return o.environmentBasis, o.Valid() && o.environmentBasis != ""
}

func AuthoredOrigin(
	context api.Context,
	source ast.Node,
) (DeclarationOrigin, error) {
	occurrence, err := context.SourceOccurrence(source)
	if err != nil {
		return DeclarationOrigin{}, err
	}
	return NewDeclarationOriginFromOccurrence(occurrence)
}

func Origin(
	source *load.Package,
	file load.File,
	outputPath string,
	node ast.Node,
) (DeclarationOrigin, error) {
	evidence, err := FileEvidence(source, file, outputPath)
	if err != nil {
		return DeclarationOrigin{}, err
	}
	occurrence, err := evidence.Occurrence(node)
	if err != nil {
		return DeclarationOrigin{}, err
	}
	return NewDeclarationOriginFromOccurrence(occurrence)
}

func FileEvidence(
	source *load.Package,
	file load.File,
	outputPath string,
) (sourceevidence.File, error) {
	if source == nil || source.Program() == nil || file.Syntax() == nil {
		return sourceevidence.File{}, &Error{
			Reason: "source-file evidence owner is incomplete",
		}
	}
	ownerKind, ownerKey := PackageOwner(source)
	return sourceevidence.NewFile(
		source.FileSet(),
		file.Syntax(),
		source.Path(),
		source.ModulePath(),
		source.ModuleVersion(),
		ownerKind,
		ownerKey,
		outputPath,
		file.SourceIdentity(),
		file.SourceDigest(),
		source.Program().SourceDigest(),
		source.TypesInfo().FileVersions[file.Syntax()],
	)
}

func PackageEvidence(
	source *load.Package,
	outputPath string,
) (sourceevidence.Package, error) {
	if source == nil {
		return sourceevidence.Package{}, &Error{
			Reason: "source-package evidence owner is absent",
		}
	}
	files := source.Files()
	evidence := make([]sourceevidence.File, 0, len(files))
	for _, file := range files {
		selected, err := FileEvidence(source, file, outputPath)
		if err != nil {
			return sourceevidence.Package{}, err
		}
		evidence = append(evidence, selected)
	}
	return sourceevidence.NewPackage(evidence)
}

func WithPackageEvidence(
	context api.Context,
	source *load.Package,
	outputPath string,
) (api.Context, error) {
	evidence, err := PackageEvidence(source, outputPath)
	if err != nil {
		return api.Context{}, err
	}
	return context.WithSourceEvidence(evidence), nil
}

func PackageOwner(source *load.Package) (string, string) {
	if source == nil {
		return "invalid", ""
	}
	switch source.Owner() {
	case load.PackageOwnerModule:
		return "module", source.OwnerKey()
	case load.PackageOwnerWorkspace:
		return "workspace", source.OwnerKey()
	case load.PackageOwnerStandardLibrary:
		return "standard-library", source.OwnerKey()
	case load.PackageOwnerToolchain:
		return "toolchain", source.OwnerKey()
	case load.PackageOwnerLanguage:
		return "language", source.OwnerKey()
	case load.PackageOwnerExternal:
		return "external", source.OwnerKey()
	default:
		return "invalid", ""
	}
}

func NewDeclarationOriginFromOccurrence(
	occurrence sourceevidence.Occurrence,
) (DeclarationOrigin, error) {
	if !occurrence.Valid() {
		return DeclarationOrigin{}, &Error{Reason: "source occurrence is invalid"}
	}
	return NewDeclarationOrigin(
		occurrence.PackagePath(),
		occurrence.ModulePath(),
		occurrence.ModuleVersion(),
		occurrence.OwnerKind(),
		occurrence.OwnerKey(),
		occurrence.OutputPath(),
		occurrence.SourceIdentity(),
		occurrence.SourceFileDigest(),
		occurrence.ProgramDigest(),
		occurrence.Start(),
		occurrence.End(),
		occurrence.GoVersion(),
	)
}

func NewDeclarationOrigin(
	packagePath string,
	modulePath string,
	moduleVersion string,
	ownerKind string,
	contractKey string,
	outputPath string,
	sourceIdentity string,
	sourceFileDigest string,
	programDigest string,
	sourceStart int,
	sourceEnd int,
	goVersion string,
) (DeclarationOrigin, error) {
	origin := DeclarationOrigin{
		kind:             "authored",
		packagePath:      packagePath,
		modulePath:       modulePath,
		moduleVersion:    moduleVersion,
		ownerKind:        ownerKind,
		contractKey:      contractKey,
		outputPath:       outputPath,
		sourceIdentity:   sourceIdentity,
		sourceFileDigest: sourceFileDigest,
		programDigest:    programDigest,
		sourceStart:      sourceStart,
		sourceEnd:        sourceEnd,
		goVersion:        goVersion,
	}
	if packagePath == "" || outputPath == "" ||
		sourceIdentity == "" || sourceFileDigest == "" || programDigest == "" ||
		sourceStart < 0 || sourceEnd <= sourceStart || goVersion == "" ||
		!validAuthoredOwner(ownerKind, modulePath, moduleVersion, contractKey) {
		return DeclarationOrigin{}, &Error{Reason: "declaration origin is incomplete"}
	}
	return origin, nil
}

func NewEnvironmentDeclarationOrigin(
	packagePath string,
	modulePath string,
	moduleVersion string,
	ownerKind string,
	contractKey string,
	outputPath string,
	sourceIdentity string,
	programDigest string,
	sourceLocation string,
) (DeclarationOrigin, error) {
	origin := DeclarationOrigin{
		kind:           "environment-contract",
		packagePath:    packagePath,
		modulePath:     modulePath,
		moduleVersion:  moduleVersion,
		ownerKind:      ownerKind,
		contractKey:    contractKey,
		outputPath:     outputPath,
		sourceIdentity: sourceIdentity,
		programDigest:  programDigest,
		sourceStart:    -1,
		sourceEnd:      -1,
		sourceLocation: sourceLocation,
	}
	if packagePath == "" ||
		!validEnvironmentOwner(ownerKind, modulePath, moduleVersion, contractKey) ||
		contractKey == "" || outputPath == "" || sourceIdentity == "" ||
		programDigest == "" || sourceLocation == "" {
		return DeclarationOrigin{}, &Error{Reason: "environment declaration origin is incomplete"}
	}
	return origin, nil
}

func (o DeclarationOrigin) arguments(factory tsgo.Factory) []tsgo.Expression {
	return []tsgo.Expression{
		text(factory, o.kind),
		text(factory, o.packagePath),
		text(factory, o.modulePath),
		text(factory, o.moduleVersion),
		text(factory, o.ownerKind),
		text(factory, o.contractKey),
		text(factory, o.outputPath),
		text(factory, o.sourceIdentity),
		text(factory, o.sourceFileDigest),
		text(factory, o.programDigest),
		count(factory, o.sourceStart),
		count(factory, o.sourceEnd),
		text(factory, o.goVersion),
		text(factory, o.sourceLocation),
	}
}

func (o DeclarationOrigin) Valid() bool {
	switch o.kind {
	case "authored":
		return o.packagePath != "" && o.outputPath != "" &&
			o.sourceIdentity != "" && o.sourceFileDigest != "" && o.programDigest != "" &&
			o.sourceStart >= 0 && o.sourceEnd > o.sourceStart && o.goVersion != "" &&
			o.sourceLocation == "" && validAuthoredOwner(
			o.ownerKind,
			o.modulePath,
			o.moduleVersion,
			o.contractKey,
		)
	case "environment-contract":
		return o.packagePath != "" && validEnvironmentOwner(
			o.ownerKind,
			o.modulePath,
			o.moduleVersion,
			o.contractKey,
		) &&
			o.contractKey != "" && o.outputPath != "" && o.sourceIdentity != "" &&
			o.programDigest != "" && o.sourceStart == -1 && o.sourceEnd == -1 &&
			o.sourceLocation != ""
	default:
		return false
	}
}

func (o DeclarationOrigin) EnvironmentContract() bool {
	return o.Valid() && o.kind == "environment-contract"
}

func validAuthoredOwner(
	ownerKind string,
	modulePath string,
	moduleVersion string,
	contractKey string,
) bool {
	switch ownerKind {
	case "module":
		return modulePath != "" && contractKey != ""
	case "workspace":
		return contractKey != "" && (modulePath != "" || moduleVersion == "")
	case "standard-library", "toolchain", "language":
		return modulePath == "" && moduleVersion == "" && contractKey != ""
	default:
		return false
	}
}

func validEnvironmentOwner(
	ownerKind string,
	modulePath string,
	moduleVersion string,
	contractKey string,
) bool {
	switch ownerKind {
	case "external":
		return modulePath != "" && contractKey != ""
	case "standard-library", "toolchain", "language":
		return modulePath == "" && moduleVersion == "" && contractKey != ""
	default:
		return false
	}
}
