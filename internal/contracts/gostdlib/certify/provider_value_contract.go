package certify

import (
	"crypto/sha256"
	"encoding/hex"
	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	runtimecontract "github.com/tsoniclang/gotots/internal/contracts/runtime"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
	"go/types"
	"os"
	"slices"
	"sort"
)

func selectMethodOwner(
	method goObject,
	static tsgo.ProjectMember,
	staticOK bool,
	instance tsgo.ProjectMember,
	instanceOK bool,
) (tsgo.ProjectMember, gostdlib.AccessKind, error) {
	signature, ok := method.object.Type().(*types.Signature)
	if !ok || signature.Recv() == nil {
		return tsgo.ProjectMember{}, gostdlib.AccessInvalid, certifyError(
			"build methods",
			method.contract.Identity(),
			"Go method receiver evidence is absent",
		)
	}
	_, pointerReceiver := signature.Recv().Type().(*types.Pointer)
	if pointerReceiver {
		if !staticOK {
			return tsgo.ProjectMember{}, gostdlib.AccessInvalid, certifyError(
				"build methods",
				method.contract.Identity(),
				"pointer receiver has no static operation",
			)
		}
		return static, gostdlib.AccessStaticMethod, nil
	}
	if !instanceOK {
		return tsgo.ProjectMember{}, gostdlib.AccessInvalid, certifyError(
			"build methods",
			method.contract.Identity(),
			"value receiver has no instance operation",
		)
	}
	return instance, gostdlib.AccessInstanceMethod, nil
}

func readRuntimeContract(
	sourcePath string,
) (runtimecontract.Requirements, string, error) {
	payload, err := os.ReadFile(sourcePath)
	if err != nil {
		return runtimecontract.Requirements{}, "", certifyError(
			"read runtime contract",
			sourcePath,
			err.Error(),
		)
	}
	requirements, err := runtimecontract.Decode(payload)
	if err != nil {
		return runtimecontract.Requirements{}, "", certifyError(
			"read runtime contract",
			sourcePath,
			err.Error(),
		)
	}
	digest := sha256.Sum256(payload)
	return requirements, hex.EncodeToString(digest[:]), nil
}

func applyDefinedValueRepresentations(
	source goSurface,
	modules []gostdlib.ModuleDocument,
	facets []facetSeed,
) ([]gostdlib.ModuleDocument, error) {
	operations := make(map[string]struct{})
	for _, facet := range facets {
		if facet.Kind != gostdlib.FacetDefinedValueOperations {
			continue
		}
		if _, duplicate := operations[facet.SourceIdentity]; duplicate {
			return nil, certifyError(
				"bind defined value",
				facet.SourceIdentity,
				"operation representation is duplicated",
			)
		}
		operations[facet.SourceIdentity] = struct{}{}
	}
	claimed := make(map[string]struct{}, len(operations))
	result := append([]gostdlib.ModuleDocument(nil), modules...)
	for moduleIndex := range result {
		result[moduleIndex].Bindings = append(
			[]gostdlib.BindingDocument(nil),
			result[moduleIndex].Bindings...,
		)
		for bindingIndex := range result[moduleIndex].Bindings {
			binding := &result[moduleIndex].Bindings[bindingIndex]
			if binding.Kind != gostdlib.BindingType ||
				binding.Access != gostdlib.AccessExport {
				continue
			}
			evidence, ok := source.objects[binding.Identity]
			if !ok {
				return nil, certifyError(
					"bind defined value",
					binding.Identity,
					"selected-Go declaration is absent",
				)
			}
			typeName, ok := evidence.object.(*types.TypeName)
			if !ok || typeName.IsAlias() || !definedValueType(typeName.Type()) {
				continue
			}
			_, operation := operations[binding.Identity]
			if operation {
				binding.DefinedValue = gostdlib.DefinedValueRepresentationOperations
				claimed[binding.Identity] = struct{}{}
				continue
			}
			named, namedOK := types.Unalias(typeName.Type()).(*types.Named)
			_, callable := typeName.Type().Underlying().(*types.Signature)
			if !namedOK || !callable || named.NumMethods() != 0 {
				return nil, certifyError(
					"bind defined value",
					binding.Identity,
					"defined value has no exact representation owner",
				)
			}
			binding.DefinedValue = gostdlib.DefinedValueRepresentationCanonical
		}
	}
	for identity := range operations {
		if _, ok := claimed[identity]; !ok {
			return nil, certifyError(
				"bind defined value",
				identity,
				"operation representation has no selected type",
			)
		}
	}
	return result, nil
}

func definedValueType(source types.Type) bool {
	named, ok := types.Unalias(source).(*types.Named)
	if !ok || named.Obj() == nil {
		return false
	}
	switch underlying := named.Underlying().(type) {
	case *types.Basic, *types.Array, *types.Slice, *types.Pointer,
		*types.Map, *types.Chan:
		return true
	case *types.Signature:
		return underlying.Recv() == nil
	default:
		return false
	}
}

type providerStatefulProfileSeed struct {
	SourceIdentity string                                 `json:"sourceIdentity"`
	Specifier      string                                 `json:"specifier"`
	SourcePath     string                                 `json:"sourcePath"`
	Export         string                                 `json:"export"`
	Interfaces     []providerCallableProfileInterfaceSeed `json:"interfaces"`
	TypeArguments  []string                               `json:"typeArguments"`
	Operations     []gostdlib.FacetCapability             `json:"operations,omitempty"`
}

func buildStatefulProfileOperations(
	seed providerStatefulProfileSeed,
	facets []facetSeed,
	target tsgo.ProjectExport,
) ([]gostdlib.FacetCapability, error) {
	var expected []gostdlib.FacetCapability
	found := false
	for _, facet := range facets {
		if facet.SourceIdentity != seed.SourceIdentity ||
			facet.Kind != gostdlib.FacetNamedStructOperations {
			continue
		}
		if found {
			return nil, certifyError(
				"build provider stateful profile",
				seed.SourceIdentity,
				"named-struct operation owner is duplicated",
			)
		}
		found = true
		for _, capability := range facet.Capabilities {
			if capability.NamedStructOperation() &&
				capability != gostdlib.FacetCapabilityRepresentation {
				expected = append(expected, capability)
			}
		}
	}
	sort.Slice(expected, func(left, right int) bool {
		return expected[left] < expected[right]
	})
	if !slices.Equal(expected, seed.Operations) {
		return nil, certifyError(
			"build provider stateful profile",
			seed.SourceIdentity,
			"profile operations do not exact-join the named-struct facet",
		)
	}
	for _, capability := range expected {
		members, err := facetCapabilityMembers(capability)
		if err != nil {
			return nil, err
		}
		for _, name := range members {
			member, ok := target.ValueMember(name)
			if !ok || !member.Visible() {
				return nil, certifyError(
					"build provider stateful profile",
					seed.Export+"."+name,
					"profile operation member is absent",
				)
			}
		}
	}
	return slices.Clone(expected), nil
}

func buildProviderStructFields(
	selectedToolchain toolchain,
	source goSurface,
	typeName *types.TypeName,
	named *types.Named,
	target tsgo.ProjectExport,
	project *tsgo.ProjectInspection,
	scalarAliases map[string]string,
) ([]gostdlib.ProviderStructFieldDocument, error) {
	structure, ok := named.Underlying().(*types.Struct)
	if !ok {
		return nil, nil
	}
	selectedPackage := source.packages[typeName.Pkg().Path()]
	if selectedPackage == nil || selectedPackage.selected == nil ||
		selectedPackage.selected.Fset == nil {
		return nil, certifyError(
			"build provider struct fields",
			typeName.Name(),
			"source package field evidence is absent",
		)
	}
	result := make([]gostdlib.ProviderStructFieldDocument, 0)
	for ordinal := range structure.NumFields() {
		field := structure.Field(ordinal)
		if field == nil || !field.Exported() {
			continue
		}
		member, found := target.TypeMember(field.Name())
		if !found || !member.Visible() {
			return nil, certifyError(
				"build provider struct fields",
				typeName.Name()+"."+field.Name(),
				"exported source field has no public target member",
			)
		}
		if err := verifyProviderStructFieldScalars(
			project,
			typeName,
			field,
			member,
			scalarAliases,
		); err != nil {
			return nil, err
		}
		location, selected, err := selectedGoSourceLocation(
			selectedToolchain.root,
			selectedPackage.selected.Fset,
			field.Pos(),
		)
		if err != nil {
			return nil, certifyError(
				"build provider struct fields",
				typeName.Name()+"."+field.Name(),
				err.Error(),
			)
		}
		if !selected {
			return nil, certifyError(
				"build provider struct fields",
				typeName.Name()+"."+field.Name(),
				"exported source field is outside the selected GOROOT",
			)
		}
		owner, err := singleImplementationOwner(
			field.Name(),
			member.ImplementationOwners(),
		)
		if err != nil {
			return nil, err
		}
		result = append(result, gostdlib.ProviderStructFieldDocument{
			Member:              field.Name(),
			Ordinal:             ordinal,
			Embedded:            field.Embedded(),
			SourceSignature:     environmentcontract.StableTypeString(field.Type()),
			SourceLocation:      location,
			ImplementationOwner: owner,
			TargetFingerprint:   member.Fingerprint(),
		})
	}
	sort.Slice(result, func(left, right int) bool {
		return result[left].Member < result[right].Member
	})
	return result, nil
}

func verifyStatefulProfileTargetMembers(
	target tsgo.ProjectExport,
	fields []gostdlib.ProviderStructFieldDocument,
	methods []gostdlib.ProviderStatefulProfileMethodDocument,
	operations []gostdlib.FacetCapability,
) error {
	instance := make(map[string]struct{}, len(fields)+len(methods))
	statics := make(map[string]struct{}, len(methods)+len(operations))
	for _, field := range fields {
		instance[field.Member] = struct{}{}
	}
	for _, method := range methods {
		if _, duplicate := instance[method.Member]; duplicate {
			return certifyError(
				"build provider stateful profile",
				method.Member,
				"source field and method target members collide",
			)
		}
		instance[method.Member] = struct{}{}
		statics[method.Member] = struct{}{}
	}
	for _, capability := range operations {
		members, err := facetCapabilityMembers(capability)
		if err != nil {
			return err
		}
		for _, member := range members {
			if _, duplicate := statics[member]; duplicate {
				return certifyError(
					"build provider stateful profile",
					member,
					"profile operation and method target members collide",
				)
			}
			statics[member] = struct{}{}
		}
	}
	if err := exactJoinStatefulMembers(target.TypeMembers(), instance, "instance"); err != nil {
		return err
	}
	return exactJoinStatefulMembers(target.ValueMembers(), statics, "static")
}

func exactJoinStatefulMembers(
	actual []tsgo.ProjectMember,
	expected map[string]struct{},
	kind string,
) error {
	seen := make(map[string]struct{}, len(expected))
	for _, member := range actual {
		if !member.Visible() {
			continue
		}
		if _, ok := expected[member.Name()]; !ok {
			return certifyError(
				"build provider stateful profile",
				member.Name(),
				"profile exposes an unowned public "+kind+" member",
			)
		}
		seen[member.Name()] = struct{}{}
	}
	for name := range expected {
		if _, ok := seen[name]; !ok {
			return certifyError(
				"build provider stateful profile",
				name,
				"profile omits an owned public "+kind+" member",
			)
		}
	}
	return nil
}
