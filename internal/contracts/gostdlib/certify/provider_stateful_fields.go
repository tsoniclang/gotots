package certify

import (
	"go/types"
	"slices"
	"sort"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

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

func buildStatefulProfileFields(
	selectedToolchain toolchain,
	source goSurface,
	typeName *types.TypeName,
	named *types.Named,
	target tsgo.ProjectExport,
) ([]gostdlib.ProviderStatefulProfileFieldDocument, error) {
	structure, ok := named.Underlying().(*types.Struct)
	if !ok {
		return nil, nil
	}
	selectedPackage := source.packages[typeName.Pkg().Path()]
	if selectedPackage == nil || selectedPackage.selected == nil ||
		selectedPackage.selected.Fset == nil {
		return nil, certifyError(
			"build provider stateful profile",
			typeName.Name(),
			"source package field evidence is absent",
		)
	}
	result := make([]gostdlib.ProviderStatefulProfileFieldDocument, 0)
	for ordinal := range structure.NumFields() {
		field := structure.Field(ordinal)
		if field == nil || !field.Exported() {
			continue
		}
		member, found := target.TypeMember(field.Name())
		if !found || !member.Visible() {
			return nil, certifyError(
				"build provider stateful profile",
				typeName.Name()+"."+field.Name(),
				"exported source field has no public target member",
			)
		}
		location, selected, err := selectedGoSourceLocation(
			selectedToolchain.root,
			selectedPackage.selected.Fset,
			field.Pos(),
		)
		if err != nil {
			return nil, certifyError(
				"build provider stateful profile",
				typeName.Name()+"."+field.Name(),
				err.Error(),
			)
		}
		if !selected {
			return nil, certifyError(
				"build provider stateful profile",
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
		result = append(result, gostdlib.ProviderStatefulProfileFieldDocument{
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
	fields []gostdlib.ProviderStatefulProfileFieldDocument,
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
