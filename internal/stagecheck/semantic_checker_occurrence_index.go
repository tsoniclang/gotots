package stagecheck

import (
	"fmt"
	"go/ast"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/structure"
)

func bindIndependentOccurrenceNodes(
	expected *semanticPackageExpectation,
	index *structure.TransientIndex,
) error {
	var files []identity.FileID
	for _, file := range expected.pkg.Files() {
		files = append(files, file.Owner().ID().File())
	}
	nodeCount, err := index.OccurrenceNodeCountForFiles(files)
	if err != nil {
		return err
	}
	expected.occurrences.byNode = make(
		map[ast.Node]semanticOccurrenceRef,
		nodeCount,
	)
	if err := index.VisitOccurrenceNodesForFiles(
		files,
		func(
			id identity.OccurrenceID,
			node ast.Node,
		) error {
			return expected.occurrences.bindNode(id, node)
		},
	); err != nil {
		return err
	}
	for _, reference := range expected.order {
		record := expected.occurrenceRecord(reference)
		id := record.ID()
		node, present := index.OccurrenceNode(id)
		if !present {
			return fmt.Errorf(
				"occurrence %s has no transient node", id,
			)
		}
		if existing, mapped := expected.occurrences.
			occurrenceID(node); !mapped || existing != id {
			return fmt.Errorf(
				"semantic occurrence %s is absent from independent file bindings",
				id,
			)
		}
	}
	return nil
}

func (verifier *checkerSemanticVerifier) occurrenceID(
	node ast.Node,
) (identity.OccurrenceID, bool) {
	return verifier.expected.occurrences.occurrenceID(node)
}
