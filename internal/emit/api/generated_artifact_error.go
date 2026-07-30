package api

import (
	"fmt"
	"go/types"
)

type GeneratedArtifactError struct {
	Artifact *GeneratedArtifact
	Cause    error
}

func (e *GeneratedArtifactError) Error() string {
	if e.Artifact == nil {
		return fmt.Sprintf("emit generated artifact: %v", e.Cause)
	}
	anchor := ""
	if selected := e.Artifact.LexicalAnchor(); selected != nil {
		anchor = ", lexical anchor " + selected.Name()
	}
	operation := ""
	if _, selected, ok := e.Artifact.GenericCapability(); ok {
		operation = ", operation " + selected.Operation().String()
	}
	sourceType := types.TypeString(
		e.Artifact.SourceType(),
		func(sourcePackage *types.Package) string {
			if sourcePackage == nil {
				return ""
			}
			return sourcePackage.Path()
		},
	)
	return fmt.Sprintf(
		"emit generated artifact %q (kind %d, key %s, type %s, placement %d%s%s): %v",
		e.Artifact.TargetName(),
		e.Artifact.Kind(),
		e.Artifact.ArtifactKey(),
		sourceType,
		e.Artifact.Placement(),
		anchor,
		operation,
		e.Cause,
	)
}

func (e *GeneratedArtifactError) Unwrap() error {
	return e.Cause
}

func WrapGeneratedArtifactError(
	artifact *GeneratedArtifact,
	err error,
) error {
	if err == nil {
		return nil
	}
	return &GeneratedArtifactError{Artifact: artifact, Cause: err}
}
