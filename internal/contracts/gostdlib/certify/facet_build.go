package certify

import (
	"fmt"
	"go/types"
	"path/filepath"
	"slices"
	"sort"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func buildFacet(
	project *tsgo.ProjectInspection,
	source goSurface,
	seed facetSeed,
	targets map[string]tsgo.ProjectExport,
	representations map[string]gostdlib.ProviderRepresentationDocument,
	genericOperations map[string][]gostdlib.GenericOperationDocument,
	bindings map[string]gostdlib.BindingDocument,
	effectMarker tsgo.ProjectExport,
) (gostdlib.FacetDocument, error) {
	evidence, ok := source.objects[seed.SourceIdentity]
	if !ok {
		return gostdlib.FacetDocument{}, certifyError(
			"build facet",
			seed.SourceIdentity,
			"selected-GOROOT declaration is absent",
		)
	}
	switch seed.Kind {
	case gostdlib.FacetNamedStructOperations,
		gostdlib.FacetDefinedValueOperations:
		if _, ok := evidence.object.(*types.TypeName); !ok {
			return gostdlib.FacetDocument{}, certifyError(
				"build facet",
				seed.SourceIdentity,
				"type facet does not own a type",
			)
		}
	case gostdlib.FacetRecoveryCallable,
		gostdlib.FacetGenericCallableKernel:
		if _, ok := evidence.object.(*types.Func); !ok {
			return gostdlib.FacetDocument{}, certifyError(
				"build facet",
				seed.SourceIdentity,
				"callable facet does not own a function or method",
			)
		}
	}
	target, ok := targets[seed.Export]
	if !ok {
		return gostdlib.FacetDocument{}, certifyError(
			"build facet",
			seed.Export,
			"target export is absent",
		)
	}
	if err := validateFacetTarget(seed, target); err != nil {
		return gostdlib.FacetDocument{}, err
	}
	effect := seed.Effect
	var callableParameters []gostdlib.ProviderCallableParameterDocument
	var err error
	if seed.Kind == gostdlib.FacetGenericCallableKernel {
		operations := genericOperations[seed.SourceIdentity]
		if err := verifyGenericKernelProjection(
			project,
			evidence,
			target,
			seed.GenericTypeArguments,
			operations,
		); err != nil {
			return gostdlib.FacetDocument{}, err
		}
		effect, err = exportCallableEffect(project, target, effectMarker)
		if err != nil {
			return gostdlib.FacetDocument{}, err
		}
		callableParameters, err = genericKernelCallableParameters(
			evidence,
			target,
			len(operations),
			project,
			effectMarker,
		)
		if err != nil {
			return gostdlib.FacetDocument{}, err
		}
		binding, ok := bindings[seed.SourceIdentity]
		if !ok {
			return gostdlib.FacetDocument{}, certifyError(
				"build generic callable kernel",
				seed.SourceIdentity,
				"public binding is absent",
			)
		}
		if err := verifyGenericKernelCallableContract(
			seed.SourceIdentity,
			binding,
			effect,
			callableParameters,
		); err != nil {
			return gostdlib.FacetDocument{}, err
		}
	}
	owner, err := singleImplementationOwner(seed.Export, target.ImplementationOwners())
	if err != nil {
		return gostdlib.FacetDocument{}, err
	}
	document := gostdlib.FacetDocument{
		Kind:                 seed.Kind,
		SourceIdentity:       seed.SourceIdentity,
		Capabilities:         append([]gostdlib.FacetCapability(nil), seed.Capabilities...),
		Export:               seed.Export,
		StorageExport:        seed.StorageExport,
		RepresentationExport: seed.RepresentationExport,
		Effect:               effect,
		CallableParameters:   callableParameters,
		GenericTypeArguments: slices.Clone(seed.GenericTypeArguments),
		ImplementationOwner:  owner,
		TargetFingerprint:    target.Fingerprint(),
	}
	if seed.RepresentationExport != "" {
		representation, ok := representations[seed.RepresentationExport]
		if !ok || !slices.Contains(
			representation.SourceTypes,
			seed.SourceIdentity,
		) {
			return gostdlib.FacetDocument{}, certifyError(
				"build facet",
				seed.SourceIdentity,
				"representation does not own the selected source type",
			)
		}
	}
	if seed.StorageExport == "" {
		return document, nil
	}
	storage, ok := targets[seed.StorageExport]
	if !ok {
		return gostdlib.FacetDocument{}, certifyError(
			"build facet",
			seed.StorageExport,
			"storage target export is absent",
		)
	}
	document.StorageImplementationOwner, err = singleImplementationOwner(
		seed.StorageExport,
		storage.ImplementationOwners(),
	)
	if err != nil {
		return gostdlib.FacetDocument{}, err
	}
	document.StorageTargetFingerprint = storage.Fingerprint()
	return document, nil
}

func providerInterfaceTargets(
	config resolvedConfig,
	project *tsgo.ProjectInspection,
	representationSeeds []providerRepresentationSeed,
	modules []gostdlib.ModuleDocument,
) (map[string]tsgo.ProjectExport, error) {
	needed := make(map[string]struct{})
	for _, seed := range representationSeeds {
		for _, identity := range seed.SourceInterfaces {
			needed[identity] = struct{}{}
		}
	}
	result := make(map[string]tsgo.ProjectExport, len(needed))
	for _, module := range modules {
		var selected []gostdlib.BindingDocument
		for _, binding := range module.Bindings {
			if _, required := needed[binding.Identity]; required {
				selected = append(selected, binding)
			}
		}
		if len(selected) == 0 {
			continue
		}
		exports, err := project.Exports(filepath.Join(
			config.providerRoot,
			filepath.FromSlash(module.SourcePath),
		))
		if err != nil {
			return nil, err
		}
		byName := make(map[string]tsgo.ProjectExport, len(exports))
		for _, target := range exports {
			byName[target.Name()] = target
		}
		for _, binding := range selected {
			target, ok := byName[binding.Export]
			if !ok || binding.Kind != gostdlib.BindingType ||
				binding.Access != gostdlib.AccessExport ||
				binding.Representation != gostdlib.RepresentationDirect {
				return nil, certifyError(
					"build representation",
					binding.Identity,
					"source interface has no direct provider type",
				)
			}
			result[binding.Identity] = target
		}
	}
	for identity := range needed {
		if _, ok := result[identity]; !ok {
			return nil, certifyError(
				"build representation",
				identity,
				"source interface has no direct provider type",
			)
		}
	}
	return result, nil
}

func buildProviderRepresentation(
	source goSurface,
	seed providerRepresentationSeed,
	target tsgo.ProjectExport,
	interfaceTargets map[string]tsgo.ProjectExport,
	project *tsgo.ProjectInspection,
	effectMarker tsgo.ProjectExport,
) (gostdlib.ProviderRepresentationDocument, error) {
	implementationOwner, err := singleImplementationOwner(
		seed.Export,
		target.ImplementationOwners(),
	)
	if err != nil {
		return gostdlib.ProviderRepresentationDocument{}, err
	}
	interfaces := make([]*types.Interface, 0, len(seed.SourceInterfaces))
	expectedMembers := make(map[string]tsgo.ProjectMember)
	for _, identity := range seed.SourceInterfaces {
		evidence, ok := source.objects[identity]
		if !ok {
			return gostdlib.ProviderRepresentationDocument{}, certifyError(
				"build representation",
				identity,
				"selected-Go interface is absent",
			)
		}
		typeName, ok := evidence.object.(*types.TypeName)
		if !ok {
			return gostdlib.ProviderRepresentationDocument{}, certifyError(
				"build representation",
				identity,
				"source interface owner is not a type",
			)
		}
		selected, ok := typeName.Type().Underlying().(*types.Interface)
		if !ok {
			return gostdlib.ProviderRepresentationDocument{}, certifyError(
				"build representation",
				identity,
				"source interface owner is not an interface",
			)
		}
		interfaces = append(interfaces, selected.Complete())
		interfaceTarget := interfaceTargets[identity]
		for _, member := range interfaceTarget.TypeMembers() {
			if !member.Visible() {
				continue
			}
			if err := addRepresentationMember(
				expectedMembers,
				member,
				identity,
			); err != nil {
				return gostdlib.ProviderRepresentationDocument{}, err
			}
		}
	}
	methodsByIdentity := make(
		map[string]gostdlib.ProviderRepresentationMethodDocument,
	)
	methodByMember := make(map[string]*types.Func)
	for _, identity := range seed.SourceTypes {
		evidence, ok := source.objects[identity]
		if !ok {
			return gostdlib.ProviderRepresentationDocument{}, certifyError(
				"build representation",
				identity,
				"selected-Go source type is absent",
			)
		}
		typeName, ok := evidence.object.(*types.TypeName)
		if !ok || typeName.IsAlias() || typeName.Exported() {
			return gostdlib.ProviderRepresentationDocument{}, certifyError(
				"build representation",
				identity,
				"represented source owner is not a private named type",
			)
		}
		for _, contract := range interfaces {
			if !types.Implements(typeName.Type(), contract) {
				return gostdlib.ProviderRepresentationDocument{}, certifyError(
					"build representation",
					identity,
					"source type does not implement a certified interface",
				)
			}
		}
		methodSet := types.NewMethodSet(typeName.Type())
		for index := range methodSet.Len() {
			method, ok := methodSet.At(index).Obj().(*types.Func)
			if !ok || !method.Exported() {
				continue
			}
			method = method.Origin()
			contract, err := environmentcontract.Describe(method)
			if err != nil {
				return gostdlib.ProviderRepresentationDocument{}, err
			}
			methodEvidence, ok := source.objects[contract.Identity()]
			if !ok {
				return gostdlib.ProviderRepresentationDocument{}, certifyError(
					"build representation",
					contract.Identity(),
					"selected-Go method evidence is absent",
				)
			}
			targetMember, ok := target.TypeMember(method.Name())
			if !ok || !targetMember.Visible() {
				return gostdlib.ProviderRepresentationDocument{}, certifyError(
					"build representation",
					contract.Identity(),
					"target method is absent",
				)
			}
			if existing := methodByMember[method.Name()]; existing != nil &&
				!environmentcontract.EquivalentMethods(existing, method) {
				return gostdlib.ProviderRepresentationDocument{}, certifyError(
					"build representation",
					method.Name(),
					"represented source methods have incompatible contracts",
				)
			}
			methodByMember[method.Name()] = method
			if err := addRepresentationMember(
				expectedMembers,
				targetMember,
				contract.Identity(),
			); err != nil {
				return gostdlib.ProviderRepresentationDocument{}, err
			}
			owner, err := singleImplementationOwner(
				method.Name(),
				targetMember.ImplementationOwners(),
			)
			if err != nil {
				return gostdlib.ProviderRepresentationDocument{}, err
			}
			effect, err := memberCallableEffect(
				project,
				targetMember,
				effectMarker,
			)
			if err != nil {
				return gostdlib.ProviderRepresentationDocument{}, err
			}
			document := gostdlib.ProviderRepresentationMethodDocument{
				SourceIdentity:      contract.Identity(),
				Member:              method.Name(),
				Effect:              effect,
				SourceSignature:     contract.Signature(),
				SourceLocation:      methodEvidence.location,
				ImplementationOwner: owner,
				TargetFingerprint:   targetMember.Fingerprint(),
			}
			if existing, duplicate := methodsByIdentity[contract.Identity()]; duplicate && existing != document {
				return gostdlib.ProviderRepresentationDocument{}, certifyError(
					"build representation",
					contract.Identity(),
					"method binding is inconsistent",
				)
			}
			methodsByIdentity[contract.Identity()] = document
		}
	}
	if err := verifyRepresentationTargetMembers(target, expectedMembers); err != nil {
		return gostdlib.ProviderRepresentationDocument{}, err
	}
	methods := make(
		[]gostdlib.ProviderRepresentationMethodDocument,
		0,
		len(methodsByIdentity),
	)
	for _, method := range methodsByIdentity {
		methods = append(methods, method)
	}
	sort.Slice(methods, func(left, right int) bool {
		return methods[left].SourceIdentity < methods[right].SourceIdentity
	})
	return gostdlib.ProviderRepresentationDocument{
		Export:              seed.Export,
		SourceTypes:         append([]string(nil), seed.SourceTypes...),
		SourceInterfaces:    append([]string(nil), seed.SourceInterfaces...),
		Methods:             methods,
		ImplementationOwner: implementationOwner,
		TargetFingerprint:   target.Fingerprint(),
	}, nil
}

func addRepresentationMember(
	members map[string]tsgo.ProjectMember,
	member tsgo.ProjectMember,
	subject string,
) error {
	existing, duplicate := members[member.Name()]
	if duplicate && (existing.Flags() != member.Flags() ||
		existing.TypeString() != member.TypeString()) {
		return certifyError(
			"build representation",
			subject,
			"target member contracts conflict",
		)
	}
	members[member.Name()] = member
	return nil
}

func verifyRepresentationTargetMembers(
	target tsgo.ProjectExport,
	expected map[string]tsgo.ProjectMember,
) error {
	actual := make(map[string]tsgo.ProjectMember)
	for _, member := range target.TypeMembers() {
		if member.Visible() {
			actual[member.Name()] = member
		}
	}
	if len(actual) != len(expected) {
		return certifyError(
			"build representation",
			target.Name(),
			fmt.Sprintf(
				"target has %d visible members, want %d",
				len(actual),
				len(expected),
			),
		)
	}
	for name, wanted := range expected {
		got, ok := actual[name]
		if !ok || got.Flags() != wanted.Flags() ||
			got.TypeString() != wanted.TypeString() {
			return certifyError(
				"build representation",
				name,
				"target member contract does not exact-join its source",
			)
		}
	}
	for _, member := range target.ValueMembers() {
		if member.Visible() {
			return certifyError(
				"build representation",
				target.Name(),
				"representation target has a runtime value member",
			)
		}
	}
	return nil
}
