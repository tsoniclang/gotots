package certify

import (
	"fmt"
	"go/types"
	"slices"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func certifiedSourceGenericCallableProjection(
	project *tsgo.ProjectInspection,
	evidence goObject,
	target tsgo.ProjectExport,
) ([]gostdlib.GenericTypeArgumentDocument, error) {
	projection, err := sourceGenericCallableProjection(evidence)
	if err != nil {
		return nil, err
	}
	targetCount, err := project.CallableTypeParameterCount(target)
	if err != nil {
		return nil, err
	}
	if len(projection) != targetCount {
		return nil, certifyError(
			"build source generic callable projection",
			evidence.contract.Identity(),
			fmt.Sprintf(
				"source has %d type parameters, provider callable has %d",
				len(projection),
				targetCount,
			),
		)
	}
	return projection, nil
}

func sourceGenericCallableProjection(
	evidence goObject,
) ([]gostdlib.GenericTypeArgumentDocument, error) {
	function, ok := evidence.object.(*types.Func)
	if !ok {
		return nil, nil
	}
	signature, ok := function.Type().(*types.Signature)
	if !ok {
		return nil, certifyError(
			"derive source generic callable projection",
			evidence.contract.Identity(),
			"binding owner has no signature",
		)
	}
	result := make(
		[]gostdlib.GenericTypeArgumentDocument,
		signature.TypeParams().Len(),
	)
	for index := range result {
		result[index] = gostdlib.GenericTypeArgumentDocument{
			TypeParameter: index,
			Facet:         gostdlib.GenericTypeArgumentLogical,
		}
	}
	return result, nil
}

func verifyGenericKernelProjection(
	project *tsgo.ProjectInspection,
	evidence goObject,
	target tsgo.ProjectExport,
	projection []gostdlib.GenericTypeArgumentDocument,
	operations []gostdlib.GenericOperationDocument,
) error {
	function, ok := evidence.object.(*types.Func)
	if !ok {
		return certifyError(
			"build generic kernel",
			evidence.contract.Identity(),
			"kernel owner is not a function",
		)
	}
	signature, ok := function.Type().(*types.Signature)
	if !ok || signature.TypeParams() == nil || signature.TypeParams().Len() == 0 {
		return certifyError(
			"build generic kernel",
			evidence.contract.Identity(),
			"kernel owner is not generic",
		)
	}
	targetCount, err := project.CallableTypeParameterCount(target)
	if err != nil {
		return err
	}
	targetValues, err := project.CallableParameterCount(target)
	if err != nil {
		return err
	}
	return verifyGenericKernelShape(
		evidence.contract.Identity(),
		signature,
		projection,
		operations,
		targetCount,
		targetValues,
	)
}

func verifyGenericKernelShape(
	identity string,
	signature *types.Signature,
	projection []gostdlib.GenericTypeArgumentDocument,
	operations []gostdlib.GenericOperationDocument,
	targetTypes int,
	targetValues int,
) error {
	if signature == nil || signature.Recv() != nil ||
		signature.TypeParams() == nil || signature.TypeParams().Len() == 0 {
		return certifyError(
			"build generic kernel",
			identity,
			"kernel source contract is invalid",
		)
	}
	if targetTypes != len(projection) {
		return certifyError(
			"build generic kernel",
			identity,
			fmt.Sprintf(
				"kernel has %d type parameters, projection has %d",
				targetTypes,
				len(projection),
			),
		)
	}
	sourceValues, err := sourceCallableParameterCount(
		signature,
		gostdlib.AccessExport,
	)
	if err != nil {
		return certifyError("build generic kernel", identity, err.Error())
	}
	expectedValues := sourceValues + len(operations)
	if targetValues != expectedValues {
		return certifyError(
			"build generic kernel",
			identity,
			fmt.Sprintf(
				"kernel has %d value parameters, capability and source contract requires %d",
				targetValues,
				expectedValues,
			),
		)
	}
	for _, argument := range projection {
		if argument.TypeParameter < 0 ||
			argument.TypeParameter >= signature.TypeParams().Len() ||
			!argument.Facet.Valid() {
			return certifyError(
				"build generic kernel",
				identity,
				"kernel projection is outside the Go declaration",
			)
		}
	}
	return nil
}

func verifySourceGenericCallableProjectionBindings(
	source goSurface,
	modules []gostdlib.ModuleDocument,
) error {
	for _, module := range modules {
		for _, binding := range module.Bindings {
			if binding.Kind != gostdlib.BindingFunction ||
				binding.Access != gostdlib.AccessExport {
				continue
			}
			evidence, ok := source.objects[binding.Identity]
			if !ok {
				return certifyError(
					"verify source generic callable projections",
					binding.Identity,
					"selected-Go function evidence is absent",
				)
			}
			expected, err := sourceGenericCallableProjection(evidence)
			if err != nil {
				return err
			}
			if !slices.Equal(expected, binding.GenericTypeArguments) {
				return certifyError(
					"verify source generic callable projections",
					binding.Identity,
					"binding projection does not exact-join source type parameters",
				)
			}
		}
	}
	return nil
}
