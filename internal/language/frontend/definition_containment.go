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
	intervals map[identity.DefinitionID]definitionInterval
}

func buildDefinitionContainment(
	definitions map[identity.DefinitionID]struct{},
	parents map[identity.DefinitionID]identity.DefinitionID,
	work *Work,
) (*definitionContainment, error) {
	nodes := map[identity.DefinitionID]bool{}
	for definition := range definitions {
		nodes[definition] = true
	}
	for child, parent := range parents {
		work.DefinitionContainmentEdges++
		if child.IsZero() || child == parent {
			return nil, fmt.Errorf(
				"semantic definition containment has invalid edge %s -> %s",
				child, parent,
			)
		}
		nodes[child] = true
		if !parent.IsZero() {
			nodes[parent] = true
		}
	}
	children := map[identity.DefinitionID][]identity.DefinitionID{}
	var roots []identity.DefinitionID
	for definition := range nodes {
		parent := parents[definition]
		if parent.IsZero() {
			roots = append(roots, definition)
			continue
		}
		children[parent] = append(children[parent], definition)
	}
	sortDefinitionIDs(roots)
	for parent := range children {
		sortDefinitionIDs(children[parent])
	}
	type frame struct {
		definition identity.DefinitionID
		leave      bool
	}
	intervals := map[identity.DefinitionID]definitionInterval{}
	state := map[identity.DefinitionID]uint8{}
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
					current.definition,
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
	if len(intervals) != len(nodes) {
		return nil, fmt.Errorf(
			"semantic definition containment reached %d of %d definitions",
			len(intervals), len(nodes),
		)
	}
	work.DefinitionContainmentEntries += len(intervals)
	work.CanonicalSortInputs += len(roots)
	for _, descendants := range children {
		work.CanonicalSortInputs += len(descendants)
	}
	return &definitionContainment{intervals: intervals}, nil
}

func sortDefinitionIDs(definitions []identity.DefinitionID) {
	sort.Slice(definitions, func(left, right int) bool {
		return definitions[left].Compare(definitions[right]) < 0
	})
}

func (containment *definitionContainment) contains(
	outer identity.DefinitionID,
	inner identity.DefinitionID,
) bool {
	if containment == nil || outer.IsZero() || inner.IsZero() {
		return false
	}
	outerInterval, outerPresent := containment.intervals[outer]
	innerInterval, innerPresent := containment.intervals[inner]
	return outerPresent &&
		innerPresent &&
		outerInterval.enter <= innerInterval.enter &&
		innerInterval.leave <= outerInterval.leave
}
