package tsgo

import "fmt"

func (source projectSourceEvidence) directChildren(parent uint32) []uint32 {
	children := make([]uint32, 0)
	for index := 1; index < len(source.nodes); index++ {
		if source.nodes[index].parent == parent {
			children = append(children, uint32(index))
		}
	}
	return children
}

func (source projectSourceEvidence) firstChild(parent uint32) uint32 {
	for index := uint32(1); index < uint32(len(source.nodes)); index++ {
		if source.nodes[index].parent == parent {
			return index
		}
	}
	return 0
}

func (source projectSourceEvidence) node(
	index uint32,
	kind uint32,
) (projectNodeEvidence, bool) {
	if index == 0 ||
		index >= uint32(len(source.nodes)) ||
		source.nodes[index].kind != kind {
		return projectNodeEvidence{}, false
	}
	return source.nodes[index], true
}

func (inspection *ProjectInspection) typeAtProjectNode(
	sourcePath string,
	index uint32,
	subject string,
) (typeResponse, error) {
	return inspection.projectNodeType(
		"getTypeAtLocation",
		sourcePath,
		index,
		subject,
	)
}

func (inspection *ProjectInspection) typeFromProjectTypeNode(
	sourcePath string,
	index uint32,
	subject string,
) (typeResponse, error) {
	return inspection.projectNodeType(
		"getTypeFromTypeNode",
		sourcePath,
		index,
		subject,
	)
}

func (inspection *ProjectInspection) projectNodeType(
	operation string,
	sourcePath string,
	index uint32,
	subject string,
) (typeResponse, error) {
	if index == 0 {
		return typeResponse{}, projectNodeEvidenceError(subject, "node is absent")
	}
	source, err := inspection.projectSourceEvidence(sourcePath)
	if err != nil {
		return typeResponse{}, err
	}
	if index >= uint32(len(source.nodes)) {
		return typeResponse{}, projectNodeEvidenceError(
			subject,
			"node is outside the source",
		)
	}
	var selected *typeResponse
	if err := requestProjectJSON(
		inspection.client,
		operation,
		getTypeAtLocationParams{
			Snapshot: inspection.snapshot,
			Project:  inspection.project,
			Location: projectNodeHandle(sourcePath, index, source.nodes[index].kind),
		},
		&selected,
	); err != nil {
		return typeResponse{}, err
	}
	if selected == nil || selected.ID == 0 {
		return typeResponse{}, projectNodeEvidenceError(
			subject,
			"checker type is absent",
		)
	}
	return *selected, nil
}

func (inspection *ProjectInspection) symbolAtProjectNode(
	sourcePath string,
	index uint32,
	subject string,
) (symbolResponse, error) {
	source, err := inspection.projectSourceEvidence(sourcePath)
	if err != nil {
		return symbolResponse{}, err
	}
	if index == 0 || index >= uint32(len(source.nodes)) {
		return symbolResponse{}, projectNodeEvidenceError(
			subject,
			"node is outside the source",
		)
	}
	var selected *symbolResponse
	if err := requestProjectJSON(
		inspection.client,
		"getSymbolAtLocation",
		getSymbolAtLocationParams{
			Snapshot: inspection.snapshot,
			Project:  inspection.project,
			Location: projectNodeHandle(sourcePath, index, source.nodes[index].kind),
		},
		&selected,
	); err != nil {
		return symbolResponse{}, err
	}
	if selected == nil || selected.ID == 0 || selected.Name == "" {
		return symbolResponse{}, projectNodeEvidenceError(
			subject,
			"checker symbol is absent",
		)
	}
	return *selected, nil
}

func projectNodeHandle(sourcePath string, index uint32, kind uint32) string {
	return fmt.Sprintf("%d.%d.%s", index, kind, sourcePath)
}

func projectNodeEvidenceError(subject string, reason string) error {
	return &ProjectInspectionError{
		Operation: "node evidence",
		Path:      subject,
		Reason:    reason,
	}
}

type getTypeAtLocationParams struct {
	Snapshot uint64 `json:"snapshot"`
	Project  string `json:"project"`
	Location string `json:"location"`
}
