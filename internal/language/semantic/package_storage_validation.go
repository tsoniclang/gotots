package semantic

import "fmt"

type storageArenaUse struct {
	name string
	used []bool
}

func newStorageArenaUse(name string, size int) storageArenaUse {
	return storageArenaUse{name: name, used: make([]bool, size)}
}

func (arena *storageArenaUse) payload(reference uint64) error {
	if reference == 0 || reference > uint64(len(arena.used)) {
		return fmt.Errorf(
			"semantic %s payload reference %d exceeds %d",
			arena.name,
			reference,
			len(arena.used),
		)
	}
	index := reference - 1
	if arena.used[index] {
		return fmt.Errorf(
			"semantic %s payload %d has multiple owners",
			arena.name,
			reference,
		)
	}
	arena.used[index] = true
	return nil
}

func (arena *storageArenaUse) relation(
	start uint64,
	count uint64,
) error {
	if start > uint64(len(arena.used)) ||
		count > uint64(len(arena.used))-start {
		return fmt.Errorf(
			"semantic %s range %d+%d exceeds %d",
			arena.name,
			start,
			count,
			len(arena.used),
		)
	}
	for index := start; index < start+count; index++ {
		if arena.used[index] {
			return fmt.Errorf(
				"semantic %s relation index %d has multiple owners",
				arena.name,
				index,
			)
		}
		arena.used[index] = true
	}
	return nil
}

func (arena storageArenaUse) complete() error {
	for index, used := range arena.used {
		if !used {
			return fmt.Errorf(
				"semantic %s arena index %d has no owner",
				arena.name,
				index,
			)
		}
	}
	return nil
}

func completeStorageArenas(arenas ...storageArenaUse) error {
	for _, arena := range arenas {
		if err := arena.complete(); err != nil {
			return err
		}
	}
	return nil
}

func validateNormalizedPackageStorage(pkg Package) error {
	if err := validatePackageIdentityTable(pkg.identities); err != nil {
		return err
	}
	return validateAdmittedNormalizedPackageStorage(pkg)
}

func validateAdmittedNormalizedPackageStorage(pkg Package) error {
	if err := validateDefinitionStorage(pkg.definitions); err != nil {
		return err
	}
	if err := validateResolutionStorage(pkg.resolutions); err != nil {
		return err
	}
	if err := validateBindingStorage(pkg.bindings); err != nil {
		return err
	}
	if err := validateTypeStorage(pkg.types); err != nil {
		return err
	}
	if err := validateOperationStorage(pkg.operations); err != nil {
		return err
	}
	return validateNormalizedPackageSemantics(pkg)
}
