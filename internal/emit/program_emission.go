package emit

import (
	"slices"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (e ProgramEmission) Files() []TargetFile {
	return slices.Clone(e.files)
}

func (f TargetFile) OutputPath() string {
	return f.outputPath
}

func (f TargetFile) PackageName() string {
	return f.packageName
}

func (f TargetFile) SourceFile() tsgo.SourceFile {
	return f.sourceFile
}

func (f TargetFile) Kind() TargetFileKind {
	return f.kind
}
