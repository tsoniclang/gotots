package tsgo

import (
	"fmt"
	"path/filepath"
	"slices"
)

type ProjectCallableScalarAliases struct {
	Parameters [][]string
	Results    []string
}

func (p *ProjectInspection) MemberScalarAliases(
	target ProjectMember,
	aliases map[string]string,
) ([]string, error) {
	if p == nil || target.Name() == "" {
		return nil, &ProjectInspectionError{
			Operation: "member scalar aliases",
			Reason:    "target is absent",
		}
	}
	expectedPaths := make(map[string]string, len(aliases))
	for alias, sourcePath := range aliases {
		if alias == "" || sourcePath == "" {
			return nil, &ProjectInspectionError{
				Operation: "member scalar aliases",
				Reason:    "scalar alias identity is incomplete",
			}
		}
		absolute, err := filepath.Abs(sourcePath)
		if err != nil {
			return nil, err
		}
		expectedPaths[alias] = filepath.ToSlash(absolute)
	}
	handles := target.scalarDeclarationHandles()
	if len(handles) == 0 {
		return nil, &ProjectInspectionError{
			Operation: "member scalar aliases",
			Path:      target.Name(),
			Reason:    "declaration is absent",
		}
	}
	var result []string
	for index, handle := range handles {
		node, _, sourcePath, err := parseProjectNodeHandle(handle)
		if err != nil {
			return nil, err
		}
		source, err := p.projectSourceEvidence(sourcePath)
		if err != nil {
			return nil, err
		}
		typeNode, err := source.singleDirectTypeNode(node)
		if err != nil {
			return nil, projectNodeEvidenceError(
				"member "+target.Name(),
				err.Error(),
			)
		}
		var selected []string
		if typeNode != 0 {
			selected, err = p.scalarAliasesInTypeNode(
				sourcePath,
				source,
				typeNode,
				expectedPaths,
				make(map[uint64]error),
				make(map[uint64]bool),
				true,
			)
			if err != nil {
				return nil, err
			}
		}
		if index == 0 {
			result = selected
			continue
		}
		if !slices.Equal(result, selected) {
			return nil, &ProjectInspectionError{
				Operation: "member scalar aliases",
				Path:      target.Name(),
				Reason:    "declarations disagree on scalar aliases",
			}
		}
	}
	return slices.Clone(result), nil
}

type projectDeclaredCallable interface {
	projectCallable
	scalarDeclarationHandles() []string
}

func (e ProjectExport) scalarDeclarationHandles() []string {
	return cloneStrings(e.handles)
}

func (m ProjectMember) scalarDeclarationHandles() []string {
	return cloneStrings(m.handles)
}

func (p *ProjectInspection) CallableScalarAliases(
	target projectDeclaredCallable,
	aliases map[string]string,
) (ProjectCallableScalarAliases, error) {
	if p == nil || target == nil {
		return ProjectCallableScalarAliases{}, &ProjectInspectionError{
			Operation: "callable scalar aliases",
			Reason:    "target is absent",
		}
	}
	expectedPaths := make(map[string]string, len(aliases))
	for alias, sourcePath := range aliases {
		if alias == "" || sourcePath == "" {
			return ProjectCallableScalarAliases{}, &ProjectInspectionError{
				Operation: "callable scalar aliases",
				Reason:    "scalar alias identity is incomplete",
			}
		}
		absolute, err := filepath.Abs(sourcePath)
		if err != nil {
			return ProjectCallableScalarAliases{}, err
		}
		expectedPaths[alias] = filepath.ToSlash(absolute)
	}
	handles := target.scalarDeclarationHandles()
	if len(handles) == 0 {
		return ProjectCallableScalarAliases{}, &ProjectInspectionError{
			Operation: "callable scalar aliases",
			Path:      target.callableSubject(),
			Reason:    "declaration is absent",
		}
	}
	var result ProjectCallableScalarAliases
	for index, handle := range handles {
		selected, err := p.callableDeclarationScalarAliases(
			handle,
			expectedPaths,
			make(map[uint64]error),
		)
		if err != nil {
			return ProjectCallableScalarAliases{}, err
		}
		if index == 0 {
			result = selected
			continue
		}
		if !sameCallableScalarAliases(result, selected) {
			return ProjectCallableScalarAliases{}, &ProjectInspectionError{
				Operation: "callable scalar aliases",
				Path:      target.callableSubject(),
				Reason:    "declarations disagree on scalar aliases",
			}
		}
	}
	return result, nil
}

func (p *ProjectInspection) callableDeclarationScalarAliases(
	handle string,
	expectedPaths map[string]string,
	verified map[uint64]error,
) (ProjectCallableScalarAliases, error) {
	sourcePath, source, parameterNodes, err :=
		p.callableDeclarationParameters(handle)
	if err != nil {
		return ProjectCallableScalarAliases{}, err
	}
	index, _, _, err := parseProjectNodeHandle(handle)
	if err != nil {
		return ProjectCallableScalarAliases{}, err
	}
	result := ProjectCallableScalarAliases{
		Parameters: make([][]string, len(parameterNodes)),
	}
	for parameterIndex, parameter := range parameterNodes {
		typeNode, err := source.singleDirectTypeNode(parameter)
		if err != nil {
			return ProjectCallableScalarAliases{}, projectNodeEvidenceError(
				fmt.Sprintf("callable parameter %d", parameterIndex),
				err.Error(),
			)
		}
		if typeNode == 0 {
			continue
		}
		result.Parameters[parameterIndex], err = p.scalarAliasesInTypeNode(
			sourcePath,
			source,
			typeNode,
			expectedPaths,
			verified,
			make(map[uint64]bool),
			false,
		)
		if err != nil {
			return ProjectCallableScalarAliases{}, err
		}
	}
	returnNode, err := source.singleDirectTypeNode(index)
	if err != nil {
		return ProjectCallableScalarAliases{}, projectNodeEvidenceError(
			"callable result",
			err.Error(),
		)
	}
	if returnNode == 0 {
		return result, nil
	}
	result.Results, err = p.scalarAliasesInTypeNode(
		sourcePath,
		source,
		returnNode,
		expectedPaths,
		verified,
		make(map[uint64]bool),
		false,
	)
	if err != nil {
		return ProjectCallableScalarAliases{}, err
	}
	return result, nil
}

func (source projectSourceEvidence) singleDirectTypeNode(
	parent uint32,
) (uint32, error) {
	var selected uint32
	for _, child := range source.directChildren(parent) {
		if !IsTypeNodeSyntaxKind(SyntaxKind(source.nodes[child].kind)) {
			continue
		}
		if selected != 0 {
			return 0, fmt.Errorf("type annotation is duplicated")
		}
		selected = child
	}
	return selected, nil
}

func (p *ProjectInspection) scalarAliasesInTypeNode(
	sourcePath string,
	source projectSourceEvidence,
	root uint32,
	expectedPaths map[string]string,
	verified map[uint64]error,
	expanding map[uint64]bool,
	expandAliases bool,
) ([]string, error) {
	var result []string
	var visit func(uint32) error
	visit = func(index uint32) error {
		node := source.nodes[index]
		if node.kind == uint32(SyntaxKindTypeReference) {
			identifier, ok := source.directIdentifier(index)
			if ok {
				symbol, err := p.symbolAtProjectNode(
					sourcePath,
					identifier,
					"scalar type reference",
				)
				if err != nil {
					return err
				}
				expectedPath, scalar := expectedPaths[symbol.Name]
				if scalar {
					if err := p.verifyScalarAliasSymbol(
						symbol,
						expectedPath,
						verified,
					); err != nil {
						return err
					}
					result = append(result, symbol.Name)
					return nil
				}
				if expandAliases &&
					!expanding[symbol.ID] &&
					declarationsWithinProject(
						symbol.Declarations,
						filepath.Dir(p.config),
					) {
					expanding[symbol.ID] = true
					defer delete(expanding, symbol.ID)
					for _, declaration := range symbol.Declarations {
						aliasNode, kind, aliasPath, err :=
							parseProjectNodeHandle(declaration)
						if err != nil {
							return err
						}
						if kind != uint32(SyntaxKindTypeAliasDeclaration) {
							continue
						}
						aliasSource, err := p.projectSourceEvidence(aliasPath)
						if err != nil {
							return err
						}
						aliasType, err := aliasSource.singleDirectTypeNode(aliasNode)
						if err != nil {
							return err
						}
						if aliasType != 0 {
							aliases, err := p.scalarAliasesInTypeNode(
								aliasPath,
								aliasSource,
								aliasType,
								expectedPaths,
								verified,
								expanding,
								expandAliases,
							)
							if err != nil {
								return err
							}
							result = append(result, aliases...)
						}
						break
					}
				}
			}
		}
		for _, child := range source.directChildren(index) {
			if err := visit(child); err != nil {
				return err
			}
		}
		return nil
	}
	if err := visit(root); err != nil {
		return nil, err
	}
	return result, nil
}

func (source projectSourceEvidence) directIdentifier(
	parent uint32,
) (uint32, bool) {
	var selected uint32
	for _, child := range source.directChildren(parent) {
		if source.nodes[child].kind != uint32(SyntaxKindIdentifier) {
			continue
		}
		if selected != 0 {
			return 0, false
		}
		selected = child
	}
	return selected, selected != 0
}

func (p *ProjectInspection) verifyScalarAliasSymbol(
	symbol symbolResponse,
	expectedPath string,
	verified map[uint64]error,
) error {
	if result, exists := verified[symbol.ID]; exists {
		return result
	}
	result := p.verifyScalarAliasSymbolUncached(symbol, expectedPath)
	verified[symbol.ID] = result
	return result
}

func (p *ProjectInspection) verifyScalarAliasSymbolUncached(
	symbol symbolResponse,
	expectedPath string,
) error {
	for _, declaration := range symbol.Declarations {
		index, kind, sourcePath, err := parseProjectNodeHandle(declaration)
		if err != nil {
			return err
		}
		if filepath.ToSlash(sourcePath) == expectedPath &&
			kind == uint32(SyntaxKindTypeAliasDeclaration) {
			return nil
		}
		if kind != uint32(SyntaxKindImportSpecifier) {
			continue
		}
		source, err := p.projectSourceEvidence(sourcePath)
		if err != nil {
			return err
		}
		importDeclaration := source.ancestorOfKind(
			index,
			uint32(SyntaxKindImportDeclaration),
		)
		if importDeclaration == 0 {
			continue
		}
		for _, child := range source.directChildren(importDeclaration) {
			if source.nodes[child].kind != uint32(SyntaxKindStringLiteral) {
				continue
			}
			module, err := p.symbolAtProjectNode(
				sourcePath,
				child,
				"scalar import module",
			)
			if err != nil {
				return err
			}
			exports, err := p.symbolExports(module.ID)
			if err != nil {
				return err
			}
			for _, exported := range exports {
				if exported.Name != symbol.Name {
					continue
				}
				for _, exportedDeclaration := range exported.Declarations {
					_, exportedKind, exportedPath, err :=
						parseProjectNodeHandle(exportedDeclaration)
					if err != nil {
						return err
					}
					if filepath.ToSlash(exportedPath) == expectedPath &&
						exportedKind == uint32(SyntaxKindTypeAliasDeclaration) {
						return nil
					}
				}
			}
		}
	}
	return &ProjectInspectionError{
		Operation: "scalar alias identity",
		Path:      symbol.Name,
		Reason:    "type reference is not owned by " + expectedPath,
	}
}

func (source projectSourceEvidence) ancestorOfKind(
	index uint32,
	kind uint32,
) uint32 {
	for index != 0 {
		if source.nodes[index].kind == kind {
			return index
		}
		index = source.nodes[index].parent
	}
	return 0
}

func (p *ProjectInspection) symbolExports(
	symbol uint64,
) ([]symbolResponse, error) {
	var exports []symbolResponse
	if err := requestProjectJSON(
		p.client,
		"getExportsOfSymbol",
		getExportsOfSymbolParams{
			Snapshot: p.snapshot,
			Symbol:   symbol,
		},
		&exports,
	); err != nil {
		return nil, err
	}
	return exports, nil
}

func sameCallableScalarAliases(
	left ProjectCallableScalarAliases,
	right ProjectCallableScalarAliases,
) bool {
	if len(left.Parameters) != len(right.Parameters) ||
		!slices.Equal(left.Results, right.Results) {
		return false
	}
	for index := range left.Parameters {
		if !slices.Equal(left.Parameters[index], right.Parameters[index]) {
			return false
		}
	}
	return true
}
