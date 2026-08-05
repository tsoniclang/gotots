package certify

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func validateProfileRootBounds(
	indexes []int,
	values *types.Tuple,
	name string,
	sourceIdentity string,
) error {
	length := 0
	if values != nil {
		length = values.Len()
	}
	for _, index := range indexes {
		if index < 0 || index >= length {
			return certifyError(
				"build provider callable profile",
				sourceIdentity,
				name+" root is outside the source callable signature",
			)
		}
	}
	return nil
}

func verifyProviderCapabilityViews(
	project *tsgo.ProjectInspection,
	target tsgo.ProjectExport,
	offset int,
	views []gostdlib.ProviderCallableProfileCapabilityViewDocument,
	interfaces []gostdlib.ProviderCallableProfileInterfaceDocument,
	targets map[string]tsgo.ProjectExport,
	source goSurface,
) error {
	if len(views) == 0 {
		return nil
	}
	parameters, err := project.CallableParameterNames(target)
	if err != nil {
		return err
	}
	byIdentity := make(
		map[string]gostdlib.ProviderCallableProfileInterfaceDocument,
		len(interfaces),
	)
	for _, selected := range interfaces {
		byIdentity[selected.SourceIdentity] = selected
	}
	for index, view := range views {
		parameter := offset + index
		if parameter >= len(parameters) ||
			parameters[parameter] != view.TargetParameter {
			return certifyError(
				"verify provider capability view",
				target.Name(),
				"capability-view target parameter does not match the certified slot",
			)
		}
		baseDocument, baseFound := byIdentity[view.BaseSourceIdentity]
		targetDocument, targetFound := byIdentity[view.TargetSourceIdentity]
		baseTarget, baseTargetFound := targets[baseDocument.Export]
		viewTarget, viewTargetFound := targets[targetDocument.Export]
		if !baseFound || !targetFound || !baseTargetFound || !viewTargetFound {
			return certifyError(
				"verify provider capability view",
				target.Name(),
				"capability-view interface evidence is absent",
			)
		}
		arguments, err := project.CallableParameterTypeArguments(target, parameter)
		if err != nil {
			return err
		}
		if len(arguments) != 2 || !arguments[0].Matches(baseTarget) ||
			!arguments[1].Matches(viewTarget) {
			return certifyError(
				"verify provider capability view",
				target.Name(),
				"capability-view generic arguments do not exact-join its interfaces",
			)
		}
		base, err := providerCapabilityNamedInterface(
			source,
			view.BaseSourceIdentity,
		)
		if err != nil {
			return err
		}
		selected, err := providerCapabilityNamedInterface(
			source,
			view.TargetSourceIdentity,
		)
		if err != nil {
			return err
		}
		baseContract := base.Underlying().(*types.Interface).Complete()
		targetContract := selected.Underlying().(*types.Interface).Complete()
		if !types.Implements(selected, baseContract) ||
			types.Implements(base, targetContract) {
			return certifyError(
				"verify provider capability view",
				view.TargetSourceIdentity,
				"target is not a strict Go interface capability of the base",
			)
		}
	}
	return nil
}

func providerCapabilityNamedInterface(
	source goSurface,
	identity string,
) (*types.Named, error) {
	var object types.Object
	if identity == gostdlib.LanguageErrorInterfaceIdentity {
		object = types.Universe.Lookup("error")
	} else if evidence, ok := source.objects[identity]; ok {
		object = evidence.object
	}
	typeName, ok := object.(*types.TypeName)
	if !ok || typeName.IsAlias() {
		return nil, certifyError(
			"verify provider capability view",
			identity,
			"source owner is not a named interface",
		)
	}
	named, ok := types.Unalias(typeName.Type()).(*types.Named)
	if !ok {
		return nil, certifyError(
			"verify provider capability view",
			identity,
			"source owner is not a named interface",
		)
	}
	contract, ok := named.Underlying().(*types.Interface)
	if !ok || !contract.Complete().IsMethodSet() {
		return nil, certifyError(
			"verify provider capability view",
			identity,
			"source owner is not a method-set interface",
		)
	}
	return named, nil
}
