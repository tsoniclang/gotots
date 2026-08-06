package tsgo

import (
	"fmt"
	"path/filepath"
)

type ProjectStatementEvidence struct {
	kind     SyntaxKind
	typeOnly bool
}

func (e ProjectStatementEvidence) Kind() SyntaxKind { return e.kind }
func (e ProjectStatementEvidence) TypeOnly() bool   { return e.typeOnly }

func (p *ProjectInspection) SourceStatements(
	sourceFile string,
) ([]ProjectStatementEvidence, error) {
	absolute, err := filepath.Abs(sourceFile)
	if err != nil {
		return nil, projectInspectionError("statements", sourceFile, err)
	}
	path := filepath.ToSlash(absolute)
	source, err := p.projectSourceEvidence(path)
	if err != nil {
		return nil, err
	}
	if len(source.nodes) < 3 ||
		source.nodes[1].kind != uint32(SyntaxKindSourceFile) ||
		source.nodes[1].parent != 0 {
		return nil, &ProjectInspectionError{
			Operation: "statements",
			Path:      path,
			Reason:    "official source-file root is invalid",
		}
	}
	statementLists := make([]uint32, 0, 1)
	for _, child := range source.directChildren(1) {
		if source.nodes[child].kind == kindNodeList {
			statementLists = append(statementLists, child)
		}
	}
	if len(statementLists) != 1 {
		return nil, &ProjectInspectionError{
			Operation: "statements",
			Path:      path,
			Reason:    fmt.Sprintf("official source has %d statement lists", len(statementLists)),
		}
	}
	statementIndexes := source.directChildren(statementLists[0])
	if uint32(len(statementIndexes)) != source.nodes[statementLists[0]].data {
		return nil, &ProjectInspectionError{
			Operation: "statements",
			Path:      path,
			Reason:    "official statement-list cardinality differs",
		}
	}
	result := make([]ProjectStatementEvidence, len(statementIndexes))
	for index, statementIndex := range statementIndexes {
		statement := source.nodes[statementIndex]
		result[index] = ProjectStatementEvidence{kind: SyntaxKind(statement.kind)}
		switch statement.kind {
		case uint32(SyntaxKindExportDeclaration):
			result[index].typeOnly = statement.data>>24 == 1
		case uint32(SyntaxKindImportDeclaration):
			for _, child := range source.directChildren(statementIndex) {
				clause := source.nodes[child]
				if clause.kind != uint32(SyntaxKindImportClause) {
					continue
				}
				result[index].typeOnly = clause.data>>24 == 1
				break
			}
		default:
			continue
		}
	}
	return result, nil
}
