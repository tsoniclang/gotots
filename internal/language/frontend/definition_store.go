package frontend

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/executable"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/scope"
)

type packageDefinitionRef uint32

func (reference packageDefinitionRef) valid() bool {
	return reference != 0
}

type definitionInput struct {
	definition   structure.ImplementationDefinition
	parent       packageDefinitionRef
	selection    scope.DefinitionSelection
	region       executable.Region
	hasSelection bool
	hasRegion    bool
}

type definitionStore struct {
	byIdentity map[identity.DefinitionID]packageDefinitionRef
	records    []definitionInput
}

func newDefinitionStore(capacity int) *definitionStore {
	if capacity < 0 {
		panic("package definition store has negative capacity")
	}
	return &definitionStore{
		byIdentity: make(
			map[identity.DefinitionID]packageDefinitionRef,
			capacity,
		),
		records: make([]definitionInput, 0, capacity),
	}
}

func (store *definitionStore) admit(
	definition structure.ImplementationDefinition,
) (packageDefinitionRef, error) {
	id := definition.ID()
	if id.IsZero() {
		return 0, fmt.Errorf(
			"package definition store rejects zero identity",
		)
	}
	if reference := store.byIdentity[id]; reference.valid() {
		existing := store.record(reference)
		if existing.definition == definition {
			return reference, nil
		}
		return 0, fmt.Errorf(
			"package definition %s has conflicting payloads", id,
		)
	}
	if uint64(len(store.records)) >= uint64(^uint32(0)) {
		return 0, fmt.Errorf("package definition table overflows uint32")
	}
	store.records = append(store.records, definitionInput{
		definition: definition,
	})
	reference := packageDefinitionRef(len(store.records))
	store.byIdentity[id] = reference
	return reference, nil
}

func (store *definitionStore) reference(
	id identity.DefinitionID,
) packageDefinitionRef {
	if store == nil || id.IsZero() {
		return 0
	}
	return store.byIdentity[id]
}

func (store *definitionStore) record(
	reference packageDefinitionRef,
) *definitionInput {
	if store == nil ||
		!reference.valid() ||
		int(reference) > len(store.records) {
		return nil
	}
	return &store.records[reference-1]
}

func (store *definitionStore) get(
	id identity.DefinitionID,
) *definitionInput {
	return store.record(store.reference(id))
}

func (store *definitionStore) id(
	reference packageDefinitionRef,
) identity.DefinitionID {
	record := store.record(reference)
	if record == nil {
		return identity.DefinitionID{}
	}
	return record.definition.ID()
}

func (store *definitionStore) count() int {
	if store == nil {
		return 0
	}
	return len(store.records)
}

func (store *definitionStore) visit(
	visit func(packageDefinitionRef, *definitionInput) error,
) error {
	if store == nil || visit == nil {
		return fmt.Errorf(
			"package definition store visit requires store and visitor",
		)
	}
	for index := range store.records {
		if err := visit(
			packageDefinitionRef(index+1), &store.records[index],
		); err != nil {
			return err
		}
	}
	return nil
}
