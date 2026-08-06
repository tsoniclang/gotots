package tsgo

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

const typeFlagUniqueESSymbol uint32 = 1 << 14

type projectSymbolFingerprint struct {
	Name       string                     `json:"name"`
	Flags      uint32                     `json:"flags"`
	Type       string                     `json:"type"`
	Owners     []string                   `json:"owners"`
	Value      []projectMemberFingerprint `json:"value,omitempty"`
	TypeMember []projectMemberFingerprint `json:"typeMembers,omitempty"`
}

type projectMemberFingerprint struct {
	Name   string   `json:"name"`
	Flags  uint32   `json:"flags"`
	Type   string   `json:"type"`
	Owners []string `json:"owners"`
}

type projectExportContract struct {
	Name           string                  `json:"name"`
	Type           string                  `json:"type"`
	DeclaredType   string                  `json:"declaredType"`
	TypeParameters int                     `json:"typeParameters"`
	Value          []projectMemberContract `json:"value,omitempty"`
	TypeMember     []projectMemberContract `json:"typeMembers,omitempty"`
}

type projectMemberContract struct {
	Name  string `json:"name"`
	Flags uint32 `json:"flags"`
	Type  string `json:"type"`
}

func (e ProjectExport) ContractEncoding() []byte {
	if e.name == "" || e.typeString == "" {
		return nil
	}
	payload, err := json.Marshal(projectExportContract{
		Name:           e.name,
		Type:           e.typeString,
		DeclaredType:   e.declaredTypeString,
		TypeParameters: e.typeParameters,
		Value:          visibleMemberContracts(e.valueMembers),
		TypeMember:     visibleMemberContracts(e.typeMembers),
	})
	if err != nil {
		return nil
	}
	return payload
}

func visibleMemberContracts(
	members []ProjectMember,
) []projectMemberContract {
	var result []projectMemberContract
	for _, member := range members {
		if !member.Visible() {
			continue
		}
		result = append(result, projectMemberContract{
			Name:  member.fingerprintName,
			Flags: member.flags,
			Type:  member.typeString,
		})
	}
	sort.Slice(result, func(left int, right int) bool {
		return result[left].Name < result[right].Name
	})
	return result
}

func (e ProjectExport) Fingerprint() string {
	if e.name == "" || e.typeString == "" || len(e.ownerKeys) == 0 {
		return ""
	}
	record := projectSymbolFingerprint{
		Name:       e.name,
		Flags:      e.flags,
		Type:       e.typeString,
		Owners:     slices.Clone(e.ownerKeys),
		Value:      visibleMemberFingerprints(e.valueMembers),
		TypeMember: visibleMemberFingerprints(e.typeMembers),
	}
	return fingerprintProjectSymbol(record)
}

func (e ProjectExport) ValueMember(name string) (ProjectMember, bool) {
	return findProjectMember(e.valueMembers, name)
}

func (e ProjectExport) TypeMember(name string) (ProjectMember, bool) {
	return findProjectMember(e.typeMembers, name)
}

func (m ProjectMember) Fingerprint() string {
	if m.fingerprintName == "" || m.typeString == "" || len(m.ownerKeys) == 0 {
		return ""
	}
	return fingerprintProjectSymbol(projectMemberFingerprint{
		Name:   m.fingerprintName,
		Flags:  m.flags,
		Type:   m.typeString,
		Owners: slices.Clone(m.ownerKeys),
	})
}

func visibleMemberFingerprints(
	members []ProjectMember,
) []projectMemberFingerprint {
	var result []projectMemberFingerprint
	for _, member := range members {
		if !member.Visible() {
			continue
		}
		result = append(result, projectMemberFingerprint{
			Name:   member.fingerprintName,
			Flags:  member.flags,
			Type:   member.typeString,
			Owners: slices.Clone(member.ownerKeys),
		})
	}
	sort.Slice(result, func(left int, right int) bool {
		return result[left].Name < result[right].Name
	})
	return result
}

type projectComputedMemberKey struct {
	Name   string   `json:"name"`
	Owners []string `json:"owners"`
}

func (p *ProjectInspection) projectMemberFingerprintName(
	displayName string,
	handles []string,
) (string, error) {
	canonical := ""
	computedDeclarations := 0
	for _, handle := range handles {
		key, found, err := p.projectComputedMemberKey(handle)
		if err != nil {
			return "", err
		}
		if !found {
			continue
		}
		computedDeclarations++
		selected := "computed:" + fingerprintProjectSymbol(key)
		if canonical != "" && canonical != selected {
			return "", fmt.Errorf("computed member declarations disagree")
		}
		canonical = selected
	}
	if canonical == "" {
		return displayName, nil
	}
	if computedDeclarations != len(handles) {
		return "", fmt.Errorf("computed member declarations are mixed")
	}
	return canonical, nil
}

func (p *ProjectInspection) projectComputedMemberKey(
	handle string,
) (projectComputedMemberKey, bool, error) {
	declaration, _, sourcePath, err := parseProjectNodeHandle(handle)
	if err != nil {
		return projectComputedMemberKey{}, false, err
	}
	source, err := p.projectSourceEvidence(sourcePath)
	if err != nil {
		return projectComputedMemberKey{}, false, err
	}
	var computedName uint32
	for _, child := range source.directChildren(declaration) {
		if source.nodes[child].kind != uint32(SyntaxKindComputedPropertyName) {
			continue
		}
		if computedName != 0 {
			return projectComputedMemberKey{}, false, fmt.Errorf(
				"declaration has multiple computed names",
			)
		}
		computedName = child
	}
	if computedName == 0 {
		return projectComputedMemberKey{}, false, nil
	}
	expressions := source.directChildren(computedName)
	if len(expressions) != 1 {
		return projectComputedMemberKey{}, false, fmt.Errorf(
			"computed member name has %d expressions, want one",
			len(expressions),
		)
	}
	keyType, err := p.typeAtProjectNode(
		sourcePath,
		expressions[0],
		"computed member key",
	)
	if err != nil {
		return projectComputedMemberKey{}, false, err
	}
	if keyType.Flags&typeFlagUniqueESSymbol == 0 {
		return projectComputedMemberKey{}, false, nil
	}
	keySymbol, err := p.symbolAtProjectNode(
		sourcePath,
		expressions[0],
		"computed unique-symbol member key",
	)
	if err != nil {
		return projectComputedMemberKey{}, false, err
	}
	declarations, err := projectDeclarationPaths(
		sourcePath,
		"computed unique-symbol member key",
		keySymbol.Name,
		keySymbol.Declarations,
	)
	if err != nil {
		return projectComputedMemberKey{}, false, err
	}
	owners, err := projectOwnerKeys(declarations, filepath.Dir(p.config))
	if err != nil {
		return projectComputedMemberKey{}, false, err
	}
	return projectComputedMemberKey{
		Name:   keySymbol.Name,
		Owners: owners,
	}, true, nil
}

func findProjectMember(
	members []ProjectMember,
	name string,
) (ProjectMember, bool) {
	index, found := slices.BinarySearchFunc(
		members,
		name,
		func(member ProjectMember, selected string) int {
			return strings.Compare(member.name, selected)
		},
	)
	if !found {
		return ProjectMember{}, false
	}
	return members[index], true
}

func fingerprintProjectSymbol(source any) string {
	payload, err := json.Marshal(source)
	if err != nil {
		return ""
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:])
}

func projectOwnerKeys(paths []string, root string) ([]string, error) {
	result := make([]string, len(paths))
	for index, sourcePath := range paths {
		relative, err := filepath.Rel(root, filepath.FromSlash(sourcePath))
		if err != nil || relative == ".." ||
			strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("declaration owner is outside the project")
		}
		result[index] = filepath.ToSlash(relative)
	}
	slices.Sort(result)
	result = slices.Compact(result)
	if len(result) == 0 {
		return nil, fmt.Errorf("declaration owner is absent")
	}
	return result, nil
}
