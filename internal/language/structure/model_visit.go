package structure

import "fmt"

// VisitDefinitions visits file-owned definitions in canonical order without
// constructing a snapshot.
func (g FileGraph) VisitDefinitions(
	visit func(ImplementationDefinition) error,
) error {
	if visit == nil {
		return fmt.Errorf("definition visit requires a visitor")
	}
	for _, definition := range g.definitions {
		if err := visit(definition); err != nil {
			return err
		}
	}
	return nil
}

// DefinitionCount reports the exact number of package-owned and file-owned
// definitions without constructing a package-wide snapshot.
func (g PackageGraph) DefinitionCount() int {
	count := len(g.ownedDefinitions)
	for _, file := range g.files {
		count += len(file.definitions)
	}
	return count
}

// VisitDefinitions visits definitions in the same canonical order exposed by
// Definitions without constructing a package-wide snapshot.
func (g PackageGraph) VisitDefinitions(
	visit func(ImplementationDefinition) error,
) error {
	if visit == nil {
		return fmt.Errorf("definition visit requires a visitor")
	}
	for _, definition := range g.ownedDefinitions {
		if err := visit(definition); err != nil {
			return err
		}
	}
	for _, file := range g.files {
		for _, definition := range file.definitions {
			if err := visit(definition); err != nil {
				return err
			}
		}
	}
	return nil
}
