package frontend

import (
	"fmt"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/executable"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/scope"
	"github.com/tsoniclang/gotots/internal/source"
)

func (stage *stageInput) visitPackageInputs(
	visit func(*packageInput) error,
) (int, error) {
	if stage == nil || visit == nil {
		return 0, fmt.Errorf(
			"semantic package input projection requires stage and visitor",
		)
	}
	count := 0
	var previous identity.PackageID
	consume := func(input *packageInput) error {
		if input == nil {
			return nil
		}
		if !previous.IsZero() && input.id.Compare(previous) <= 0 {
			return fmt.Errorf(
				"semantic package inputs are not canonical at %s",
				input.id,
			)
		}
		previous = input.id
		count++
		return visit(input)
	}
	if !stage.allLocal {
		var builtins []*source.LoadedPackage
		for _, loaded := range stage.loaded {
			if loaded.Disposition() == source.DispositionBuiltinUniverse {
				builtins = append(builtins, loaded)
			}
		}
		sort.Slice(builtins, func(left, right int) bool {
			return builtins[left].ID().Compare(
				builtins[right].ID(),
			) < 0
		})
		builtinIndex := 0
		consumeBuiltin := func() error {
			loaded := builtins[builtinIndex]
			builtinIndex++
			input, err := stage.buildBuiltinInput(loaded)
			if err != nil {
				return err
			}
			return consume(input)
		}
		err := stage.graph.VisitResidentPackages(func(
			pkg structure.PackageGraph,
		) error {
			for builtinIndex < len(builtins) &&
				builtins[builtinIndex].ID().Compare(pkg.ID()) < 0 {
				if err := consumeBuiltin(); err != nil {
					return err
				}
			}
			input, err := stage.buildPackageInput(pkg)
			if err != nil {
				return err
			}
			return consume(input)
		})
		if err != nil {
			return count, err
		}
		for builtinIndex < len(builtins) {
			if err := consumeBuiltin(); err != nil {
				return count, err
			}
		}
		return count, nil
	}
	err := stage.graph.VisitResidentPackages(func(
		pkg structure.PackageGraph,
	) error {
		input, err := stage.buildPackageInput(pkg)
		if err != nil {
			return err
		}
		return consume(input)
	})
	return count, err
}

func (stage *stageInput) buildPackageInput(
	pkg structure.PackageGraph,
) (*packageInput, error) {
	loaded := stage.loaded[pkg.ID()]
	if loaded == nil {
		return nil, fmt.Errorf(
			"semantic package %s is absent from source universe",
			pkg.ID(),
		)
	}
	input := newPackageInput(loaded, pkg, stage.index)
	for _, definition := range pkg.Definitions() {
		if !stage.allLocal && !definitionUsesLocalSemantics(
			stage.plan, loaded, definition.ID(),
		) {
			continue
		}
		input.definitions[definition.ID()] = definition
		selection, present := stage.selections.For(definition.ID())
		if !present {
			return nil, fmt.Errorf(
				"local definition %s has no selection",
				definition.ID(),
			)
		}
		input.selections[definition.ID()] = selection
		if region, present := stage.executable.For(
			definition.ID(),
		); present {
			input.regions[definition.ID()] = region
		}
	}
	for _, site := range pkg.Sites() {
		if _, local := input.definitions[site.Definition()]; local {
			input.parents[site.Definition()] = site.ParentDefinition()
		}
	}
	if err := finishPackageInput(input, stage.executable); err != nil {
		return nil, err
	}
	if len(input.definitions) == 0 &&
		input.occurrences.count() == 0 {
		return nil, nil
	}
	authority, err := checkerAuthority(
		stage.universe, pkg, loaded, stage.facts,
	)
	if err != nil {
		return nil, err
	}
	input.authority = authority
	return input, nil
}

func (stage *stageInput) buildBuiltinInput(
	loaded *source.LoadedPackage,
) (*packageInput, error) {
	input := newPackageInput(
		loaded, structure.PackageGraph{}, stage.index,
	)
	if err := finishPackageInput(input, stage.executable); err != nil {
		return nil, err
	}
	authority, err := checkerAuthority(
		stage.universe,
		structure.PackageGraph{},
		loaded,
		stage.facts,
	)
	if err != nil {
		return nil, err
	}
	input.authority = authority
	return input, nil
}

func newPackageInput(
	loaded *source.LoadedPackage,
	graph structure.PackageGraph,
	index *structure.TransientIndex,
) *packageInput {
	return &packageInput{
		id: loaded.ID(),
		provenance: semanticProvenance(
			loaded.Provenance(),
		),
		loaded:      loaded,
		graph:       graph,
		index:       index,
		occurrences: newOccurrenceStore(),
		definitions: map[identity.DefinitionID]structure.ImplementationDefinition{},
		parents:     map[identity.DefinitionID]identity.DefinitionID{},
		regions:     map[identity.DefinitionID]executable.Region{},
		selections:  map[identity.DefinitionID]scope.DefinitionSelection{},
	}
}

func finishPackageInput(
	input *packageInput,
	inventory *executable.Inventory,
) error {
	definitions := make(
		map[identity.DefinitionID]struct{},
		len(input.definitions),
	)
	for definition := range input.definitions {
		definitions[definition] = struct{}{}
	}
	containment, err := buildDefinitionContainment(
		definitions, input.parents, &input.work,
	)
	if err != nil {
		return err
	}
	input.containment = containment
	return input.buildOccurrences(input.index, inventory)
}
