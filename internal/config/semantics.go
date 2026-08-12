package config

import (
	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func loadBuildProfile(selected goDocument, toolchainVersion string) (load.BuildProfile, error) {
	profile, err := load.NewBuildProfileForToolchain(
		toolchainVersion,
		*selected.GOOS,
		*selected.GOARCH,
		*selected.CGO,
		selected.Tags,
	)
	if err != nil {
		return load.BuildProfile{}, projectError(
			"validate config",
			"go",
			err.Error(),
		)
	}
	return profile, nil
}

func parseInteger(selected string) (emit.IntegerRepresentation, error) {
	value, err := emit.ParseIntegerRepresentation(selected)
	if err != nil {
		return emit.IntegerRepresentationInvalid, projectError(
			"validate config",
			"semantics.integers",
			err.Error(),
		)
	}
	return value, nil
}

func parseEvaluation(selected string) (emit.EvaluationOrder, error) {
	value, err := emit.ParseEvaluationOrder(selected)
	if err != nil {
		return emit.EvaluationOrderInvalid, projectError(
			"validate config",
			"semantics.evaluationOrder",
			err.Error(),
		)
	}
	return value, nil
}

func parseConcurrency(selected string) (emit.ConcurrencySemantics, error) {
	value, err := emit.ParseConcurrencySemantics(selected)
	if err != nil {
		return emit.ConcurrencySemanticsInvalid, projectError(
			"validate config",
			"semantics.concurrency",
			err.Error(),
		)
	}
	return value, nil
}
