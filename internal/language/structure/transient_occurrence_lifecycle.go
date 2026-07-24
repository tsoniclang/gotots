package structure

import (
	"fmt"
	"go/ast"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
)

func (store *transientOccurrenceStore) seal() error {
	if store == nil || store.sealed {
		return fmt.Errorf(
			"transient occurrence store cannot seal",
		)
	}
	if err := store.compactSupplements(); err != nil {
		return err
	}
	for index := range store.files {
		file := &store.files[index]
		for domain := transientOccurrenceStructural; domain < transientOccurrenceDomainCount; domain++ {
			for _, node := range file.canonical[domain].nodes {
				if node == nil {
					return fmt.Errorf(
						"transient canonical occurrence in %s has no node",
						file.id,
					)
				}
			}
		}
		for _, supplement := range file.supplements {
			if supplement.node == nil {
				return fmt.Errorf(
					"transient supplemental occurrence in %s has no node",
					file.id,
				)
			}
		}
		file.supplementPositions = nil
	}
	if len(store.pending) != 0 {
		return fmt.Errorf(
			"transient occurrence store retains %d pending canonical stores",
			len(store.pending),
		)
	}
	store.byStore = nil
	store.pending = nil
	store.sealed = true
	return nil
}

func (store *transientOccurrenceStore) compactSupplements() error {
	for fileIndex := range store.files {
		file := &store.files[fileIndex]
		retained := make(
			[]transientOccurrenceBinding,
			0,
			len(file.supplements),
		)
		for _, binding := range file.supplements {
			if binding.node == nil {
				continue
			}
			retained = append(retained, binding)
		}
		sort.Slice(retained, func(left, right int) bool {
			return lessTransientOccurrenceKey(
				retained[left].key,
				retained[right].key,
			)
		})
		file.supplements = retained
		for index, entry := range retained {
			if index > 0 &&
				retained[index-1].key == entry.key {
				return fmt.Errorf(
					"transient supplemental occurrence key repeats in %s",
					file.id,
				)
			}
		}
	}
	return nil
}

func (store *transientOccurrenceStore) visitFiles(
	files []identity.FileID,
	visit func(identity.OccurrenceID, ast.Node) error,
) error {
	if store == nil || !store.sealed || visit == nil {
		return fmt.Errorf(
			"transient occurrence visit requires sealed storage and visitor",
		)
	}
	seen := map[identity.FileID]bool{}
	for _, fileID := range files {
		if fileID.IsZero() || seen[fileID] {
			return fmt.Errorf(
				"transient occurrence visit has zero or duplicate file %s",
				fileID,
			)
		}
		seen[fileID] = true
		_, file := store.file(fileID, false)
		if file == nil {
			continue
		}
		for domain := transientOccurrenceStructural; domain < transientOccurrenceDomainCount; domain++ {
			canonical := file.canonical[domain]
			if canonical.store == nil {
				continue
			}
			if err := canonical.store.Visit(func(reference OccurrenceRef) error {
				return visit(
					reference.ID(),
					canonical.nodes[reference.index-1],
				)
			}); err != nil {
				return err
			}
		}
		for _, binding := range file.supplements {
			id, err := transientOccurrenceIdentity(file.id, binding.key)
			if err != nil {
				return err
			}
			if err := visit(id, binding.node); err != nil {
				return err
			}
		}
	}
	return nil
}

func (store *transientOccurrenceStore) countFiles(
	files []identity.FileID,
) (int, error) {
	if store == nil || !store.sealed {
		return 0, fmt.Errorf(
			"transient occurrence count requires sealed storage",
		)
	}
	seen := map[identity.FileID]bool{}
	total := 0
	for _, fileID := range files {
		if fileID.IsZero() || seen[fileID] {
			return 0, fmt.Errorf(
				"transient occurrence count has zero or duplicate file %s",
				fileID,
			)
		}
		seen[fileID] = true
		_, file := store.file(fileID, false)
		if file == nil {
			continue
		}
		for domain := transientOccurrenceStructural; domain < transientOccurrenceDomainCount; domain++ {
			total += file.canonical[domain].store.Count()
		}
		total += len(file.supplements)
	}
	return total, nil
}
