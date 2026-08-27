package tsgo

import "path/filepath"

func (p *ProjectInspection) SourceForbiddenDynamicTypes(
	sourceFile string,
) ([]SyntaxKind, error) {
	absolute, err := filepath.Abs(sourceFile)
	if err != nil {
		return nil, projectInspectionError("forbidden dynamic types", sourceFile, err)
	}
	path := filepath.ToSlash(absolute)
	source, err := p.projectSourceEvidence(path)
	if err != nil {
		return nil, err
	}
	found := make(map[SyntaxKind]struct{}, 2)
	for _, node := range source.nodes {
		kind := SyntaxKind(node.kind)
		if kind == SyntaxKindAnyKeyword || kind == SyntaxKindUnknownKeyword {
			found[kind] = struct{}{}
		}
	}
	result := make([]SyntaxKind, 0, 2)
	for _, kind := range []SyntaxKind{SyntaxKindAnyKeyword, SyntaxKindUnknownKeyword} {
		if _, exists := found[kind]; exists {
			result = append(result, kind)
		}
	}
	return result, nil
}
