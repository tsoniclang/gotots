package api

import (
	"fmt"
	"go/types"
)

type GeneratedArtifactPlacement uint8

const (
	GeneratedArtifactPlacementInvalid GeneratedArtifactPlacement = iota
	GeneratedArtifactPlacementCompilation
	GeneratedArtifactPlacementLexical
)

func (p GeneratedArtifactPlacement) Valid() bool {
	return p == GeneratedArtifactPlacementCompilation ||
		p == GeneratedArtifactPlacementLexical
}

type GeneratedArtifact struct {
	sourceType   *types.Struct
	artifact     string
	targetName   string
	placement    GeneratedArtifactPlacement
	outputPath   string
	lexicalOwner ArtifactOwner
	anchor       *types.TypeName
}

func NewCompilationGeneratedArtifact(
	sourceType *types.Struct,
	artifact string,
	targetName string,
	outputPath string,
) (*GeneratedArtifact, error) {
	if sourceType == nil ||
		artifact == "" ||
		targetName == "" ||
		outputPath == "" {
		return nil, &RootRequestError{
			Reason: "compilation generated artifact is invalid",
		}
	}
	return &GeneratedArtifact{
		sourceType: sourceType,
		artifact:   artifact,
		targetName: targetName,
		placement:  GeneratedArtifactPlacementCompilation,
		outputPath: outputPath,
	}, nil
}

func NewLexicalGeneratedArtifact(
	sourceType *types.Struct,
	artifact string,
	targetName string,
	lexicalOwner ArtifactOwner,
	anchor *types.TypeName,
) (*GeneratedArtifact, error) {
	sourceOwner, sourceOwned := lexicalOwner.Source()
	if sourceType == nil ||
		artifact == "" ||
		targetName == "" ||
		!sourceOwned ||
		anchor == nil ||
		sourceOwner.Pkg() == nil ||
		anchor.Pkg() != sourceOwner.Pkg() ||
		anchor.Parent() == nil ||
		anchor.Parent() == anchor.Pkg().Scope() {
		return nil, &RootRequestError{
			Reason: "lexical generated artifact is invalid",
		}
	}
	return &GeneratedArtifact{
		sourceType:   sourceType,
		artifact:     artifact,
		targetName:   targetName,
		placement:    GeneratedArtifactPlacementLexical,
		lexicalOwner: lexicalOwner,
		anchor:       anchor,
	}, nil
}

func (o *GeneratedArtifact) SourceType() *types.Struct {
	if o == nil {
		return nil
	}
	return o.sourceType
}

func (o *GeneratedArtifact) ArtifactKey() string {
	if o == nil {
		return ""
	}
	return o.artifact
}

func (o *GeneratedArtifact) TargetName() string {
	if o == nil {
		return ""
	}
	return o.targetName
}

func (o *GeneratedArtifact) Placement() GeneratedArtifactPlacement {
	if o == nil {
		return GeneratedArtifactPlacementInvalid
	}
	return o.placement
}

func (o *GeneratedArtifact) OutputPath() string {
	if o == nil {
		return ""
	}
	return o.outputPath
}

func (o *GeneratedArtifact) LexicalOwner() ArtifactOwner {
	if o == nil {
		return ArtifactOwner{}
	}
	return o.lexicalOwner
}

func (o *GeneratedArtifact) LexicalAnchor() *types.TypeName {
	if o == nil {
		return nil
	}
	return o.anchor
}

func (o *GeneratedArtifact) ReconstructionOwner() ArtifactOwner {
	if o == nil {
		return ArtifactOwner{}
	}
	if o.placement == GeneratedArtifactPlacementLexical {
		return o.lexicalOwner
	}
	return MustGeneratedArtifactOwner(o)
}

func (o *GeneratedArtifact) Valid() bool {
	if o == nil ||
		o.sourceType == nil ||
		o.artifact == "" ||
		o.targetName == "" ||
		!o.placement.Valid() {
		return false
	}
	switch o.placement {
	case GeneratedArtifactPlacementCompilation:
		return o.outputPath != "" &&
			!o.lexicalOwner.Valid() &&
			o.anchor == nil
	case GeneratedArtifactPlacementLexical:
		sourceOwner, sourceOwned := o.lexicalOwner.Source()
		return o.outputPath == "" &&
			sourceOwned &&
			o.anchor != nil &&
			sourceOwner.Pkg() != nil &&
			o.anchor.Pkg() == sourceOwner.Pkg() &&
			o.anchor.Parent() != nil &&
			o.anchor.Parent() != o.anchor.Pkg().Scope()
	default:
		return false
	}
}

type GeneratedArtifactPlacementError struct {
	TypeName string
	Reason   string
}

func (e *GeneratedArtifactPlacementError) Error() string {
	if e.TypeName == "" {
		return "place generated type: " + e.Reason
	}
	return fmt.Sprintf(
		"place generated type containing %q: %s",
		e.TypeName,
		e.Reason,
	)
}

type GeneratedArtifactShapeError struct {
	Artifact string
	Reason   string
}

func (e *GeneratedArtifactShapeError) Error() string {
	if e.Artifact == "" {
		return "emit generated type: " + e.Reason
	}
	return fmt.Sprintf(
		"emit generated type %q: %s",
		e.Artifact,
		e.Reason,
	)
}
