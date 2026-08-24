package runtime

import (
	"slices"

	"github.com/tsoniclang/gotots/internal/emit/api"
	targetoutput "github.com/tsoniclang/gotots/internal/output"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type PackageFile struct {
	outputPath string
	sourceFile tsgo.SourceFile
}

type Package struct {
	files               []PackageFile
	invocationContracts []InvocationContract
	manifest            []byte
	fingerprint         string
	scalar              api.ScalarABI
	concurrency         api.ConcurrencySemantics
	valid               bool
}

func (f PackageFile) OutputPath() string {
	return f.outputPath
}

func (f PackageFile) SourceFile() tsgo.SourceFile {
	return f.sourceFile
}

func (p Package) Valid() bool {
	return p.valid
}

func (p Package) Name() string {
	if !p.valid {
		return ""
	}
	return targetoutput.RuntimePackageName
}

func (p Package) Version() string {
	if !p.valid {
		return ""
	}
	return targetoutput.RuntimePackageVersion
}

func (p Package) RootPath() string {
	if !p.valid {
		return ""
	}
	return targetoutput.RuntimePackageRootPath
}

func (p Package) ManifestPath() string {
	if !p.valid {
		return ""
	}
	return targetoutput.RuntimePackageManifestPath
}

func (p Package) Profile() api.IntegerRepresentation {
	return p.scalar.IntegerRepresentation()
}

func (p Package) NativeIntegerWidth() api.NativeIntegerWidth {
	return p.scalar.NativeIntegerWidth()
}

func (p Package) Concurrency() api.ConcurrencySemantics {
	return p.concurrency
}

func (p Package) Files() []PackageFile {
	return slices.Clone(p.files)
}

func (p Package) Manifest() []byte {
	return slices.Clone(p.manifest)
}

func (p Package) Fingerprint() string {
	return p.fingerprint
}
