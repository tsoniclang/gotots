package semantic

import (
	"fmt"
	"sort"
)

type storedUnsupported struct {
	id        unsupportedRef
	reason    UnsupportedReason
	evidence  string
	authority authorityRef
}

type packageUnsupportedBuilder struct {
	records []storedUnsupported
}

func (builder *packageUnsupportedBuilder) add(
	identities *packageIdentityBuilder,
	authorities *packageAuthorityBuilder,
	record Unsupported,
) {
	builder.records = append(builder.records, storedUnsupported{
		id:        identities.unsupportedID(record.id),
		reason:    record.reason,
		evidence:  record.evidence,
		authority: authorities.authority(record.authority),
	})
}

type packageUnsupportedStore struct {
	records []storedUnsupported
}

func (builder *packageUnsupportedBuilder) seal(
	identities packageIdentityRemap,
	authorities []uint64,
) (packageUnsupportedStore, error) {
	store := packageUnsupportedStore{records: builder.records}
	var err error
	for index := range store.records {
		record := &store.records[index]
		if record.id, err = remapReference(
			record.id, identities.unsupported,
		); err != nil {
			return packageUnsupportedStore{}, err
		}
		if record.authority, err = remapReference(
			record.authority, authorities,
		); err != nil {
			return packageUnsupportedStore{}, err
		}
	}
	sort.Slice(store.records, func(left, right int) bool {
		return store.records[left].id < store.records[right].id
	})
	return store, nil
}

func (store packageUnsupportedStore) record(
	identities *packageIdentityProjection,
	authorities packageAuthorityTable,
	index int,
) (Unsupported, error) {
	if index < 0 || index >= len(store.records) {
		return Unsupported{}, fmt.Errorf(
			"semantic unsupported index %d is invalid", index,
		)
	}
	stored := store.records[index]
	authority, present := authorities.authority(stored.authority)
	if !present {
		return Unsupported{}, fmt.Errorf(
			"semantic unsupported authority reference is invalid",
		)
	}
	return Unsupported{
		id:        identities.unsupportedID(stored.id),
		reason:    stored.reason,
		evidence:  stored.evidence,
		authority: authority,
	}, nil
}
