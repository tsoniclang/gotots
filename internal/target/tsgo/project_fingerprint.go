package tsgo

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
)

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
	if m.name == "" || m.typeString == "" || len(m.ownerKeys) == 0 {
		return ""
	}
	return fingerprintProjectSymbol(projectMemberFingerprint{
		Name:   m.name,
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
			Name:   member.name,
			Flags:  member.flags,
			Type:   member.typeString,
			Owners: slices.Clone(member.ownerKeys),
		})
	}
	return result
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
