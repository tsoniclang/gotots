package frontend

import (
	"fmt"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
)

type definitionInterval struct {
	enter int
	leave int
}

type definitionContainment struct {
	definitions *definitionStore
	intervals   []definitionInterval
}

func buildDefinitionContainment(
	definitions *definitionStore,
	work *Work,
) (*definitionContainment, error) {
	if definitions == nil {
		return nil, fmt.Errorf(
			"semantic definition containment requires definitions",
		)
	}
	children := make(
		[][]packageDefinitionRef, definitions.count()+1,
	)
	var roots []packageDefinitionRef
	if err := definitions.visit(func(
		child packageDefinitionRef,
		record *definitionInput,
	) error {
		parent := record.parent
		if child == parent {
			return fmt.Errorf(
				"semantic definition containment has invalid edge %s -> %s",
				definitions.id(child), definitions.id(parent),
			)
		}
		if !parent.valid() {
			roots = append(roots, child)
			return nil
		}
		work.DefinitionContainmentEdges++
		children[parent] = append(children[parent], child)
		return nil
	}); err != nil {
		return nil, err
	}
	sortDefinitionRefs(definitions, roots)
	for _, descendants := range children {
		sortDefinitionRefs(definitions, descendants)
	}
	type frame struct {
		definition packageDefinitionRef
		leave      bool
	}
	intervals := make(
		[]definitionInterval, definitions.count()+1,
	)
	state := make([]uint8, definitions.count()+1)
	clock := 0
	for _, root := range roots {
		stack := []frame{{definition: root}}
		for len(stack) != 0 {
			work.DefinitionContainmentVisits++
			last := len(stack) - 1
			current := stack[last]
			stack = stack[:last]
			if current.leave {
				interval := intervals[current.definition]
				interval.leave = clock
				clock++
				intervals[current.definition] = interval
				state[current.definition] = 2
				continue
			}
			switch state[current.definition] {
			case 1:
				return nil, fmt.Errorf(
					"semantic definition containment cycle at %s",
					definitions.id(current.definition),
				)
			case 2:
				continue
			}
			state[current.definition] = 1
			intervals[current.definition] = definitionInterval{
				enter: clock,
			}
			clock++
			stack = append(stack, frame{
				definition: current.definition,
				leave:      true,
			})
			descendants := children[current.definition]
			for index := len(descendants) - 1; index >= 0; index-- {
				stack = append(stack, frame{
					definition: descendants[index],
				})
			}
		}
	}
	reached := 0
	for reference := packageDefinitionRef(1); int(reference) <= definitions.count(); reference++ {
		if state[reference] == 2 {
			reached++
		}
	}
	if reached != definitions.count() {
		return nil, fmt.Errorf(
			"semantic definition containment reached %d of %d definitions",
			reached, definitions.count(),
		)
	}
	work.DefinitionContainmentEntries += reached
	work.CanonicalSortInputs += len(roots)
	for _, descendants := range children {
		work.CanonicalSortInputs += len(descendants)
	}
	return &definitionContainment{
		definitions: definitions,
		intervals:   intervals,
	}, nil
}

func sortDefinitionRefs(
	store *definitionStore,
	definitions []packageDefinitionRef,
) {
	sort.Slice(definitions, func(left, right int) bool {
		return store.id(definitions[left]).Compare(
			store.id(definitions[right]),
		) < 0
	})
}

func (containment *definitionContainment) contains(
	outerID identity.DefinitionID,
	innerID identity.DefinitionID,
) bool {
	if containment == nil || outerID.IsZero() || innerID.IsZero() {
		return false
	}
	outer := containment.definitions.reference(outerID)
	inner := containment.definitions.reference(innerID)
	if !outer.valid() || !inner.valid() {
		return false
	}
	outerInterval := containment.intervals[outer]
	innerInterval := containment.intervals[inner]
	return outerInterval.enter <= innerInterval.enter &&
		innerInterval.leave <= outerInterval.leave
}
