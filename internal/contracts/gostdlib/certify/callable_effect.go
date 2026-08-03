package certify

import (
	"path/filepath"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const (
	callableEffectMarkerPath = "src/internal/certify/callable-effect.ts"
	callableEffectMarkerName = "AsyncEffectMarker"
)

func loadCallableEffectMarker(
	config resolvedConfig,
	project *tsgo.ProjectInspection,
) (tsgo.ProjectExport, error) {
	exports, err := project.Exports(filepath.Join(
		config.providerRoot,
		filepath.FromSlash(callableEffectMarkerPath),
	))
	if err != nil {
		return tsgo.ProjectExport{}, err
	}
	if len(exports) != 1 || exports[0].Name() != callableEffectMarkerName {
		return tsgo.ProjectExport{}, certifyError(
			"inspect callable effect",
			callableEffectMarkerPath,
			"marker export is not exact",
		)
	}
	return exports[0], nil
}

func exportCallableEffect(
	project *tsgo.ProjectInspection,
	target tsgo.ProjectExport,
	marker tsgo.ProjectExport,
) (gostdlib.EffectKind, error) {
	selected, err := project.CallableEffect(target, marker)
	if err != nil {
		return gostdlib.EffectInvalid, err
	}
	return providerEffect(selected)
}

func memberCallableEffect(
	project *tsgo.ProjectInspection,
	target tsgo.ProjectMember,
	marker tsgo.ProjectExport,
) (gostdlib.EffectKind, error) {
	selected, err := project.CallableEffect(target, marker)
	if err != nil {
		return gostdlib.EffectInvalid, err
	}
	return providerEffect(selected)
}

func parameterCallableEffect(
	project *tsgo.ProjectInspection,
	target tsgo.ProjectExport,
	parameter int,
	marker tsgo.ProjectExport,
) (gostdlib.EffectKind, error) {
	selected, err := project.CallableParameterEffect(target, parameter, marker)
	if err != nil {
		return gostdlib.EffectInvalid, err
	}
	return providerEffect(selected)
}

func providerEffect(source tsgo.CallableEffect) (gostdlib.EffectKind, error) {
	switch source {
	case tsgo.CallableEffectSynchronous:
		return gostdlib.EffectSynchronous, nil
	case tsgo.CallableEffectAsynchronous:
		return gostdlib.EffectAsynchronous, nil
	case tsgo.CallableEffectAwaitable:
		return gostdlib.EffectAwaitable, nil
	default:
		return gostdlib.EffectInvalid, certifyError(
			"inspect callable effect",
			"target",
			"effect is invalid",
		)
	}
}
