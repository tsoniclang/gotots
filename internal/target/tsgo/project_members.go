package tsgo

import (
	"path/filepath"
	"slices"
	"sort"
)

type ProjectMemberAccess uint8

const (
	ProjectMemberAccessInvalid ProjectMemberAccess = iota
	ProjectMemberAccessPublic
	ProjectMemberAccessNonPublic
)

type ProjectMember struct {
	name         string
	symbolID     uint64
	flags        uint32
	typeString   string
	typeID       uint32
	handles      []string
	declarations []string
	ownerKeys    []string
	access       ProjectMemberAccess
}

func (m ProjectMember) Name() string {
	return m.name
}

func (m ProjectMember) Flags() uint32 {
	return m.flags
}

func (m ProjectMember) TypeString() string {
	return m.typeString
}

func (m ProjectMember) Declarations() []string {
	return cloneStrings(m.declarations)
}

func (m ProjectMember) ImplementationOwners() []string {
	return cloneStrings(m.ownerKeys)
}

func (m ProjectMember) Visible() bool {
	return m.name != "" && m.access == ProjectMemberAccessPublic
}

func (p *ProjectInspection) projectTypeDeclarations(
	selectedType *typeResponse,
) ([]string, bool, error) {
	if selectedType == nil ||
		selectedType.ID == 0 ||
		selectedType.Symbol == 0 {
		return nil, false, nil
	}
	var symbol *symbolResponse
	if err := requestProjectJSON(
		p.client,
		"getSymbolOfType",
		getSymbolOfTypeParams{
			Snapshot: p.snapshot,
			Type:     selectedType.ID,
		},
		&symbol,
	); err != nil {
		return nil, false, err
	}
	if symbol == nil ||
		len(symbol.Declarations) == 0 ||
		!declarationsWithinProject(
			symbol.Declarations,
			filepath.Dir(p.config),
		) {
		return nil, false, nil
	}
	return cloneStrings(symbol.Declarations), true, nil
}

func (p *ProjectInspection) projectMembers(
	sourcePath string,
	operation string,
	typeID uint32,
) ([]ProjectMember, error) {
	var symbols []symbolResponse
	if err := requestProjectJSON(
		p.client,
		"getPropertiesOfType",
		getPropertiesOfTypeParams{
			Snapshot: p.snapshot,
			Project:  p.project,
			Type:     typeID,
		},
		&symbols,
	); err != nil {
		return nil, err
	}
	members := make([]ProjectMember, 0, len(symbols))
	for _, symbol := range symbols {
		if len(symbol.Declarations) == 0 ||
			!declarationsWithinProject(
				symbol.Declarations,
				filepath.Dir(p.config),
			) {
			continue
		}
		member, err := p.projectMember(sourcePath, operation, symbol)
		if err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	sort.Slice(members, func(left, right int) bool {
		return members[left].name < members[right].name
	})
	for index := 1; index < len(members); index++ {
		if members[index-1].name == members[index].name {
			return nil, &ProjectInspectionError{
				Operation: operation,
				Path:      sourcePath,
				Reason:    "duplicate member " + members[index].name,
			}
		}
	}
	return members, nil
}

func (p *ProjectInspection) projectMember(
	sourcePath string,
	operation string,
	symbol symbolResponse,
) (ProjectMember, error) {
	if symbol.ID == 0 || symbol.Name == "" {
		return ProjectMember{}, &ProjectInspectionError{
			Operation: operation,
			Path:      sourcePath,
			Reason:    "member symbol is invalid",
		}
	}
	var selectedType *typeResponse
	if err := requestProjectJSON(
		p.client,
		"getTypeOfSymbol",
		getTypeOfSymbolParams{
			Snapshot: p.snapshot,
			Project:  p.project,
			Symbol:   symbol.ID,
		},
		&selectedType,
	); err != nil {
		return ProjectMember{}, err
	}
	if selectedType == nil || selectedType.ID == 0 {
		return ProjectMember{}, &ProjectInspectionError{
			Operation: operation,
			Path:      sourcePath,
			Reason:    "member " + symbol.Name + " has no type",
		}
	}
	var typeString string
	if err := requestProjectJSON(
		p.client,
		"typeToString",
		typeToStringParams{
			Snapshot: p.snapshot,
			Project:  p.project,
			Type:     selectedType.ID,
		},
		&typeString,
	); err != nil {
		return ProjectMember{}, err
	}
	if typeString == "" {
		return ProjectMember{}, &ProjectInspectionError{
			Operation: operation,
			Path:      sourcePath,
			Reason:    "member " + symbol.Name + " has an empty type",
		}
	}
	declarations, err := projectDeclarationPaths(
		sourcePath,
		operation,
		symbol.Name,
		symbol.Declarations,
	)
	if err != nil {
		return ProjectMember{}, err
	}
	ownerKeys, err := projectOwnerKeys(
		declarations,
		filepath.Dir(p.config),
	)
	if err != nil {
		return ProjectMember{}, &ProjectInspectionError{
			Operation: operation,
			Path:      sourcePath,
			Reason:    "member " + symbol.Name + " " + err.Error(),
		}
	}
	access, err := p.projectMemberAccess(symbol.Declarations)
	if err != nil {
		return ProjectMember{}, &ProjectInspectionError{
			Operation: operation,
			Path:      sourcePath,
			Reason:    "member " + symbol.Name + " " + err.Error(),
		}
	}
	return ProjectMember{
		name:         symbol.Name,
		symbolID:     symbol.ID,
		flags:        symbol.Flags,
		typeString:   typeString,
		typeID:       selectedType.ID,
		handles:      sortedStrings(symbol.Declarations),
		declarations: declarations,
		ownerKeys:    ownerKeys,
		access:       access,
	}, nil
}

func sortedStrings(source []string) []string {
	result := slices.Clone(source)
	slices.Sort(result)
	return result
}

func projectDeclarationPaths(
	sourcePath string,
	operation string,
	name string,
	handles []string,
) ([]string, error) {
	declarations := make([]string, 0, len(handles))
	seen := make(map[string]struct{}, len(handles))
	for _, handle := range handles {
		path, ok := declarationPath(handle)
		if !ok {
			return nil, &ProjectInspectionError{
				Operation: operation,
				Path:      sourcePath,
				Reason:    "member " + name + " has an invalid declaration handle",
			}
		}
		if _, duplicate := seen[path]; duplicate {
			continue
		}
		seen[path] = struct{}{}
		declarations = append(declarations, path)
	}
	sort.Strings(declarations)
	if len(declarations) == 0 {
		return nil, &ProjectInspectionError{
			Operation: operation,
			Path:      sourcePath,
			Reason:    "member " + name + " has no declaration owner",
		}
	}
	return declarations, nil
}

func cloneStrings(values []string) []string {
	if values == nil {
		return nil
	}
	return append([]string(nil), values...)
}
