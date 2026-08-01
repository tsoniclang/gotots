package certify

import (
	"fmt"
	"go/types"
	"slices"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func certifiedGenericCallableProjection(
	project *tsgo.ProjectInspection,
	evidence goObject,
	target tsgo.ProjectExport,
	configured map[string][]gostdlib.GenericTypeArgumentDocument,
) ([]gostdlib.GenericTypeArgumentDocument, error) {
	function, callable := evidence.object.(*types.Func)
	if !callable {
		if len(configured[evidence.contract.Identity()]) != 0 {
			return nil, certifyError(
				"build generic callable projection",
				evidence.contract.Identity(),
				"projection owner is not a function",
			)
		}
		return nil, nil
	}
	signature, _ := function.Type().(*types.Signature)
	sourceCount := 0
	if signature != nil {
		sourceCount = signature.TypeParams().Len()
	}
	projection, configuredProjection := configured[evidence.contract.Identity()]
	if sourceCount == 0 {
		if configuredProjection {
			return nil, certifyError(
				"build generic callable projection",
				evidence.contract.Identity(),
				"projection owner has no function type parameters",
			)
		}
		return nil, nil
	}
	if !configuredProjection {
		return nil, certifyError(
			"build generic callable projection",
			evidence.contract.Identity(),
			"generic provider function has no target type-argument projection",
		)
	}
	targetCount, err := project.CallableTypeParameterCount(target)
	if err != nil {
		return nil, err
	}
	if len(projection) != targetCount {
		return nil, certifyError(
			"build generic callable projection",
			evidence.contract.Identity(),
			fmt.Sprintf(
				"projection has %d target parameters, provider callable has %d",
				len(projection),
				targetCount,
			),
		)
	}
	for _, argument := range projection {
		if argument.TypeParameter >= sourceCount || !argument.Facet.Valid() {
			return nil, certifyError(
				"build generic callable projection",
				evidence.contract.Identity(),
				"projection index is outside the Go declaration",
			)
		}
	}
	return slices.Clone(projection), nil
}

func verifyGenericCallableProjectionBindings(
	modules []gostdlib.ModuleDocument,
	configured map[string][]gostdlib.GenericTypeArgumentDocument,
) error {
	bound := make(
		map[string][]gostdlib.GenericTypeArgumentDocument,
		len(configured),
	)
	for _, module := range modules {
		for _, binding := range module.Bindings {
			if len(binding.GenericTypeArguments) == 0 {
				continue
			}
			if _, duplicate := bound[binding.Identity]; duplicate {
				return certifyError(
					"verify generic callable projections",
					binding.Identity,
					"provider binding is duplicated",
				)
			}
			bound[binding.Identity] = slices.Clone(
				binding.GenericTypeArguments,
			)
		}
	}
	for identity, projection := range configured {
		if !slices.Equal(bound[identity], projection) {
			return certifyError(
				"verify generic callable projections",
				identity,
				"configured projection does not exact-join one provider binding",
			)
		}
	}
	return nil
}
