package certify

import (
	"encoding/json"
	"fmt"
	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
	"go/types"
	"io"
	"os"
	"slices"
)

func buildStateBindings(
	config resolvedConfig,
	project *tsgo.ProjectInspection,
	source *goPackageSurface,
	target tsgo.ProjectExport,
	behaviorEvidence *implementationEvidence,
) ([]gostdlib.BindingDocument, error) {
	var result []gostdlib.BindingDocument
	for _, member := range target.ValueMembers() {
		if !member.Visible() {
			continue
		}
		evidence, ok := source.objectsByName[member.Name()]
		if !ok {
			return nil, certifyError(
				"build state",
				member.Name(),
				"state member has no selected-GOROOT declaration",
			)
		}
		if _, ok := evidence.object.(*types.Var); !ok {
			return nil, certifyError(
				"build state",
				member.Name(),
				"state member does not own a Go variable",
			)
		}
		binding, err := bindingDocument(
			evidence,
			"state",
			member.Name(),
			gostdlib.AccessStateMember,
			member.Fingerprint(),
			member.ImplementationOwners(),
		)
		if err != nil {
			return nil, err
		}
		sites, behavior, behaviorErr := certifyBindingBehavior(
			config,
			project,
			binding.Identity,
			member.Name(),
			member.DeclarationNodeHandles(),
			behaviorEvidence,
		)
		if behaviorErr != nil {
			return nil, behaviorErr
		}
		binding.ImplementationSites = sites
		binding.Dependencies = behavior.dependencies
		binding.Disposition = behavior.disposition
		result = append(result, binding)
	}
	if len(result) == 0 {
		return nil, certifyError("build state", target.Name(), "state has no members")
	}
	return result, nil
}

func addTargetOwner(
	owners map[string]struct{},
	binding gostdlib.BindingDocument,
) error {
	key := string(binding.Access) + "\x00" + binding.Export + "\x00" + binding.Member
	if _, duplicate := owners[key]; duplicate {
		return certifyError("build binding", key, "target owner is duplicated")
	}
	owners[key] = struct{}{}
	return nil
}

func bindingDocument(
	evidence goObject,
	export string,
	member string,
	access gostdlib.AccessKind,
	fingerprint string,
	owners []string,
) (gostdlib.BindingDocument, error) {
	if fingerprint == "" {
		return gostdlib.BindingDocument{}, certifyError(
			"build binding",
			evidence.contract.Identity(),
			"target fingerprint is absent",
		)
	}
	if len(owners) != 1 {
		return gostdlib.BindingDocument{}, certifyError(
			"build binding",
			evidence.contract.Identity(),
			fmt.Sprintf("target has %d implementation owners, want one", len(owners)),
		)
	}
	kind, err := bindingKind(evidence.contract.Kind())
	if err != nil {
		return gostdlib.BindingDocument{}, err
	}
	representation := gostdlib.RepresentationInvalid
	if kind == gostdlib.BindingType {
		representation = gostdlib.RepresentationDirect
	}
	return gostdlib.BindingDocument{
		Identity:            evidence.contract.Identity(),
		Kind:                kind,
		Access:              access,
		Representation:      representation,
		Export:              export,
		Member:              member,
		SourceSignature:     evidence.contract.Signature(),
		SourceValue:         evidence.contract.Value(),
		SourceLocation:      evidence.location,
		ImplementationOwner: owners[0],
		TargetFingerprint:   fingerprint,
	}, nil
}

func bindingKind(source environmentcontract.ObjectKind) (gostdlib.BindingKind, error) {
	switch source {
	case environmentcontract.ObjectConstant:
		return gostdlib.BindingConstant, nil
	case environmentcontract.ObjectType:
		return gostdlib.BindingType, nil
	case environmentcontract.ObjectVariable:
		return gostdlib.BindingVariable, nil
	case environmentcontract.ObjectFunction:
		return gostdlib.BindingFunction, nil
	case environmentcontract.ObjectBuiltin:
		return gostdlib.BindingBuiltin, nil
	default:
		return gostdlib.BindingInvalid, certifyError(
			"build binding",
			fmt.Sprint(source),
			"Go object kind is unsupported",
		)
	}
}

const moduleMapSchemaVersion = 1

type moduleMapDocument struct {
	SchemaVersion int          `json:"schemaVersion"`
	Modules       []moduleSeed `json:"modules"`
}

func readModuleSeeds(sourcePath string) ([]moduleSeed, error) {
	file, err := os.Open(sourcePath)
	if err != nil {
		return nil, certifyError("read module map", sourcePath, err.Error())
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var document moduleMapDocument
	if err := decoder.Decode(&document); err != nil {
		return nil, certifyError("read module map", sourcePath, err.Error())
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("multiple JSON values")
		}
		return nil, certifyError("read module map", sourcePath, err.Error())
	}
	if document.SchemaVersion != moduleMapSchemaVersion {
		return nil, certifyError("read module map", sourcePath, "schema is unsupported")
	}
	return validateSeeds(document.Modules)
}

func verifyExportSourceCallableShape(
	project *tsgo.ProjectInspection,
	evidence goObject,
	target tsgo.ProjectExport,
) error {
	signature, ok := evidence.object.Type().(*types.Signature)
	if !ok {
		return certifyError(
			"verify source callable shape",
			evidence.contract.Identity(),
			"selected Go function signature is absent",
		)
	}
	actual, err := project.CallableParameterCount(target)
	if err != nil {
		return err
	}
	actualTypes, err := project.CallableTypeParameterCount(target)
	if err != nil {
		return err
	}
	return verifySourceCallableShape(
		evidence.contract.Identity(),
		signature,
		gostdlib.AccessExport,
		actual,
		actualTypes,
	)
}

func verifyMethodSourceCallableShape(
	project *tsgo.ProjectInspection,
	evidence goObject,
	target tsgo.ProjectMember,
	access gostdlib.AccessKind,
) error {
	signature, ok := evidence.object.Type().(*types.Signature)
	if !ok {
		return certifyError(
			"verify source callable shape",
			evidence.contract.Identity(),
			"selected Go method signature is absent",
		)
	}
	actual, err := project.CallableParameterCount(target)
	if err != nil {
		return err
	}
	actualTypes, err := project.CallableTypeParameterCount(target)
	if err != nil {
		return err
	}
	return verifySourceCallableShape(
		evidence.contract.Identity(),
		signature,
		access,
		actual,
		actualTypes,
	)
}

func verifySourceCallableShape(
	identity string,
	signature *types.Signature,
	access gostdlib.AccessKind,
	actualValues int,
	actualTypes int,
) error {
	expectedValues, err := sourceCallableParameterCount(signature, access)
	if err != nil {
		return certifyError("verify source callable shape", identity, err.Error())
	}
	if actualValues != expectedValues {
		return certifyError(
			"verify source callable shape",
			identity,
			fmt.Sprintf(
				"target has %d value parameters, selected Go shape requires %d",
				actualValues,
				expectedValues,
			),
		)
	}
	expectedTypes := sourceCallableTypeParameterCount(signature)
	if actualTypes != expectedTypes {
		return certifyError(
			"verify source callable shape",
			identity,
			fmt.Sprintf(
				"target has %d type parameters, selected Go shape requires %d",
				actualTypes,
				expectedTypes,
			),
		)
	}
	return nil
}

func sourceCallableTypeParameterCount(signature *types.Signature) int {
	if signature == nil {
		return 0
	}
	count := 0
	if signature.RecvTypeParams() != nil {
		count += signature.RecvTypeParams().Len()
	}
	if signature.TypeParams() != nil {
		count += signature.TypeParams().Len()
	}
	return count
}

func sourceCallableParameterCount(
	signature *types.Signature,
	access gostdlib.AccessKind,
) (int, error) {
	if signature == nil {
		return 0, fmt.Errorf("selected Go signature is absent")
	}
	parameterCount := 0
	if signature.Params() != nil {
		parameterCount = signature.Params().Len()
	}
	receiver := signature.Recv()
	switch access {
	case gostdlib.AccessExport:
		if receiver != nil {
			return 0, fmt.Errorf("package export unexpectedly has a receiver")
		}
		return parameterCount, nil
	case gostdlib.AccessInstanceMethod:
		if receiver == nil {
			return 0, fmt.Errorf("instance method receiver is absent")
		}
		if _, pointer := receiver.Type().(*types.Pointer); pointer {
			return 0, fmt.Errorf("pointer receiver cannot use an instance operation")
		}
		return parameterCount, nil
	case gostdlib.AccessStaticMethod:
		if receiver == nil {
			return 0, fmt.Errorf("static method receiver is absent")
		}
		if _, pointer := receiver.Type().(*types.Pointer); !pointer {
			return 0, fmt.Errorf("value receiver cannot use a static operation")
		}
		return parameterCount + 1, nil
	default:
		return 0, fmt.Errorf("callable access %q is unsupported", access)
	}
}

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

func verifyGenericKernelCallableContract(
	identity string,
	binding gostdlib.BindingDocument,
	effect gostdlib.EffectKind,
	parameters []gostdlib.ProviderCallableParameterDocument,
) error {
	if binding.Kind != gostdlib.BindingFunction {
		return certifyError(
			"verify generic callable kernel",
			identity,
			"public binding is not a function",
		)
	}
	if effect != binding.Effect {
		return certifyError(
			"verify generic callable kernel",
			identity,
			fmt.Sprintf(
				"kernel effect %q does not match public effect %q",
				effect,
				binding.Effect,
			),
		)
	}
	if !slices.Equal(parameters, binding.CallableParameters) {
		return certifyError(
			"verify generic callable kernel",
			identity,
			fmt.Sprintf(
				"kernel callable parameters %#v do not match public parameters %#v",
				parameters,
				binding.CallableParameters,
			),
		)
	}
	return nil
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
