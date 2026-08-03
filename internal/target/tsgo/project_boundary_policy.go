package tsgo

import (
	"fmt"
	"path/filepath"
	"slices"
)

type BoundaryCapabilityKind uint8

const (
	BoundaryCapabilityInvalid BoundaryCapabilityKind = iota
	BoundaryCapabilityFromProvider
	BoundaryCapabilityInterfaceGuard
)

func (k BoundaryCapabilityKind) Valid() bool {
	return k == BoundaryCapabilityFromProvider ||
		k == BoundaryCapabilityInterfaceGuard
}

type ProjectTypeIdentity struct {
	symbolID     uint64
	name         string
	declarations []string
	ownerKeys    []string
}

func (i ProjectTypeIdentity) Name() string {
	return i.name
}

func (i ProjectTypeIdentity) Declarations() []string {
	return slices.Clone(i.declarations)
}

func (i ProjectTypeIdentity) ImplementationOwners() []string {
	return slices.Clone(i.ownerKeys)
}

func (i ProjectTypeIdentity) Matches(target ProjectExport) bool {
	return i.symbolID != 0 && i.symbolID == target.typeSymbolID
}

type BoundaryCapability struct {
	member string
	kind   BoundaryCapabilityKind
	source ProjectTypeIdentity
}

func (c BoundaryCapability) Member() string {
	return c.member
}

func (c BoundaryCapability) Kind() BoundaryCapabilityKind {
	return c.kind
}

func (c BoundaryCapability) Source() ProjectTypeIdentity {
	return c.source
}

type ProjectBoundaryPolicy struct {
	parameter    int
	source       ProjectTypeIdentity
	capabilities []BoundaryCapability
}

func (p ProjectBoundaryPolicy) Parameter() int {
	return p.parameter
}

func (p ProjectBoundaryPolicy) Source() ProjectTypeIdentity {
	return p.source
}

func (p ProjectBoundaryPolicy) Capabilities() []BoundaryCapability {
	return slices.Clone(p.capabilities)
}

func (p *ProjectInspection) BoundaryPolicy(
	target projectCallable,
	policyMarker ProjectExport,
	fromProviderMarker ProjectExport,
	interfaceGuardMarker ProjectExport,
) (ProjectBoundaryPolicy, error) {
	if p == nil || target == nil {
		return ProjectBoundaryPolicy{}, boundaryPolicyError("target is absent")
	}
	signature, err := p.singleCallSignature(target, target.callableSubject())
	if err != nil {
		return ProjectBoundaryPolicy{}, err
	}
	if len(signature.Parameters) == 0 {
		return ProjectBoundaryPolicy{}, boundaryPolicyError(
			"target has no policy parameter",
		)
	}
	policyType, err := p.projectSymbolType(
		signature.Parameters[len(signature.Parameters)-1],
		"boundary policy",
	)
	if err != nil {
		return ProjectBoundaryPolicy{}, err
	}
	policyTypeID, err := p.nonNullableType(policyType.ID)
	if err != nil {
		return ProjectBoundaryPolicy{}, err
	}
	policyMembers, err := p.projectMembers(
		p.config,
		"boundary policy",
		policyTypeID,
	)
	if err != nil {
		return ProjectBoundaryPolicy{}, err
	}
	policyIdentity, err := exactTypeMarker(
		policyMarker,
		"$go$canonicalBoundarySource",
	)
	if err != nil {
		return ProjectBoundaryPolicy{}, err
	}
	fromProviderIdentity, err := exactTypeMarker(
		fromProviderMarker,
		"$go$fromProviderSource",
	)
	if err != nil {
		return ProjectBoundaryPolicy{}, err
	}
	interfaceGuardIdentity, err := exactTypeMarker(
		interfaceGuardMarker,
		"$go$guardSource",
	)
	if err != nil {
		return ProjectBoundaryPolicy{}, err
	}
	foundPolicyMarker := false
	var source ProjectTypeIdentity
	for _, member := range policyMembers {
		if sameMemberIdentity(member, policyIdentity) {
			foundPolicyMarker = true
			source, err = p.projectTypeIdentity(member.typeID)
			if err != nil {
				return ProjectBoundaryPolicy{}, err
			}
			break
		}
	}
	if !foundPolicyMarker {
		return ProjectBoundaryPolicy{}, boundaryPolicyError(
			"policy parameter does not inherit the canonical policy marker",
		)
	}
	capabilities := make([]BoundaryCapability, 0, len(policyMembers))
	for _, member := range policyMembers {
		if sameMemberIdentity(member, policyIdentity) {
			continue
		}
		capability, err := p.boundaryCapability(
			member,
			fromProviderMarker,
			fromProviderIdentity,
			interfaceGuardMarker,
			interfaceGuardIdentity,
		)
		if err != nil {
			return ProjectBoundaryPolicy{}, err
		}
		capabilities = append(capabilities, capability)
	}
	return ProjectBoundaryPolicy{
		parameter:    len(signature.Parameters) - 1,
		source:       source,
		capabilities: capabilities,
	}, nil
}

func (p *ProjectInspection) boundaryCapability(
	member ProjectMember,
	fromProviderTarget ProjectExport,
	fromProviderMarker ProjectMember,
	interfaceGuardTarget ProjectExport,
	interfaceGuardMarker ProjectMember,
) (BoundaryCapability, error) {
	typeID, err := p.nonNullableType(member.typeID)
	if err != nil {
		return BoundaryCapability{}, err
	}
	members, err := p.projectMembers(
		p.config,
		"boundary capability "+member.name,
		typeID,
	)
	if err != nil {
		return BoundaryCapability{}, err
	}
	kind := BoundaryCapabilityInvalid
	for _, selected := range members {
		switch {
		case sameMemberIdentity(selected, fromProviderMarker):
			if kind.Valid() {
				return BoundaryCapability{}, boundaryPolicyError(
					"capability " + member.name + " has multiple policy kinds",
				)
			}
			kind = BoundaryCapabilityFromProvider
		case sameMemberIdentity(selected, interfaceGuardMarker):
			if kind.Valid() {
				return BoundaryCapability{}, boundaryPolicyError(
					"capability " + member.name + " has multiple policy kinds",
				)
			}
			kind = BoundaryCapabilityInterfaceGuard
		}
	}
	if !kind.Valid() {
		return BoundaryCapability{}, boundaryPolicyError(
			"policy member " + member.name + " has no recognized capability marker",
		)
	}
	owner, err := p.projectTypeIdentity(typeID)
	if err != nil {
		return BoundaryCapability{}, err
	}
	expectedOwner := fromProviderTarget
	if kind == BoundaryCapabilityInterfaceGuard {
		expectedOwner = interfaceGuardTarget
	}
	if !owner.Matches(expectedOwner) {
		return BoundaryCapability{}, boundaryPolicyError(
			"capability " + member.name + " is not a direct marker instantiation",
		)
	}
	typeArguments, err := p.typeArguments(typeID)
	if err != nil {
		return BoundaryCapability{}, err
	}
	if len(typeArguments) != 2 {
		return BoundaryCapability{}, boundaryPolicyError(
			fmt.Sprintf(
				"capability %s has %d type arguments, want two",
				member.name,
				len(typeArguments),
			),
		)
	}
	source, err := p.projectTypeIdentity(typeArguments[0].ID)
	if err != nil {
		return BoundaryCapability{}, err
	}
	return BoundaryCapability{member: member.name, kind: kind, source: source}, nil
}

func exactTypeMarker(target ProjectExport, name string) (ProjectMember, error) {
	marker, ok := target.TypeMember(name)
	if !ok || len(marker.handles) == 0 {
		return ProjectMember{}, boundaryPolicyError(
			fmt.Sprintf("marker %s.%s is absent", target.Name(), name),
		)
	}
	return marker, nil
}

func sameMemberIdentity(left ProjectMember, right ProjectMember) bool {
	return len(left.handles) != 0 && slices.Equal(left.handles, right.handles)
}

func (p *ProjectInspection) typeArguments(typeID uint32) ([]typeResponse, error) {
	var selected []typeResponse
	if err := requestProjectJSON(
		p.client,
		"getTypeArguments",
		getTypePropertyParams{
			Snapshot: p.snapshot,
			Project:  p.project,
			Type:     typeID,
		},
		&selected,
	); err != nil {
		return nil, err
	}
	return selected, nil
}

func (p *ProjectInspection) projectTypeIdentity(
	typeID uint32,
) (ProjectTypeIdentity, error) {
	nonNullable, err := p.nonNullableType(typeID)
	if err != nil {
		return ProjectTypeIdentity{}, err
	}
	var symbol *symbolResponse
	if err := requestProjectJSON(
		p.client,
		"getSymbolOfType",
		getSymbolOfTypeParams{Snapshot: p.snapshot, Type: nonNullable},
		&symbol,
	); err != nil {
		return ProjectTypeIdentity{}, err
	}
	if symbol == nil || symbol.ID == 0 || symbol.Name == "" {
		return ProjectTypeIdentity{}, boundaryPolicyError(
			"capability source type has no symbol",
		)
	}
	declarations, err := projectDeclarationPaths(
		p.config,
		"boundary capability source",
		symbol.Name,
		symbol.Declarations,
	)
	if err != nil {
		return ProjectTypeIdentity{}, err
	}
	owners, err := projectOwnerKeys(declarations, filepath.Dir(p.config))
	if err != nil {
		return ProjectTypeIdentity{}, boundaryPolicyError(err.Error())
	}
	return ProjectTypeIdentity{
		symbolID:     symbol.ID,
		name:         symbol.Name,
		declarations: declarations,
		ownerKeys:    owners,
	}, nil
}

func boundaryPolicyError(reason string) error {
	return &ProjectInspectionError{Operation: "boundary policy", Reason: reason}
}
