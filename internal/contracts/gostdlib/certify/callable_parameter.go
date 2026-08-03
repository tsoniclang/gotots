package certify

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	gostdlibsource "github.com/tsoniclang/gotots/internal/contracts/gostdlib/sourcecontract"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type callableParameterEffect func(int) (gostdlib.EffectKind, error)

func exportCallableParameters(
	evidence goObject,
	target tsgo.ProjectExport,
	project *tsgo.ProjectInspection,
	effectMarker tsgo.ProjectExport,
) ([]gostdlib.ProviderCallableParameterDocument, error) {
	return callableParameterDocuments(
		evidence,
		gostdlib.AccessExport,
		func(parameter int) (gostdlib.EffectKind, error) {
			return parameterCallableEffect(
				project,
				target,
				parameter,
				effectMarker,
			)
		},
	)
}

func memberCallableParameters(
	evidence goObject,
	target tsgo.ProjectMember,
	access gostdlib.AccessKind,
	project *tsgo.ProjectInspection,
	effectMarker tsgo.ProjectExport,
) ([]gostdlib.ProviderCallableParameterDocument, error) {
	return callableParameterDocuments(
		evidence,
		access,
		func(parameter int) (gostdlib.EffectKind, error) {
			selected, err := project.CallableParameterEffect(
				target,
				parameter,
				effectMarker,
			)
			if err != nil {
				return gostdlib.EffectInvalid, err
			}
			return providerEffect(selected)
		},
	)
}

func callableParameterDocuments(
	evidence goObject,
	access gostdlib.AccessKind,
	effectAt callableParameterEffect,
) ([]gostdlib.ProviderCallableParameterDocument, error) {
	signature, ok := evidence.object.Type().(*types.Signature)
	if !ok {
		return nil, certifyError(
			"inspect callable parameters",
			evidence.contract.Identity(),
			"selected Go callable signature is absent",
		)
	}
	offset := 0
	if access == gostdlib.AccessStaticMethod {
		offset = 1
	}
	var result []gostdlib.ProviderCallableParameterDocument
	for parameter := range signature.Params().Len() {
		if _, callable := gostdlibsource.DirectCallableParameterSignature(
			signature.Params().At(parameter).Type(),
		); !callable {
			continue
		}
		effect, err := effectAt(parameter + offset)
		if err != nil {
			return nil, err
		}
		result = append(result, gostdlib.ProviderCallableParameterDocument{
			Parameter: parameter,
			Effect:    effect,
		})
	}
	return result, nil
}
