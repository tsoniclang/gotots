package stagecheck

import (
	"fmt"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
)

type checkerDefinitionInterval struct {
	enter int
	leave int
}

func deriveCheckerDefinitionIntervals(
	expected semanticPackageExpectation,
) (map[identity.DefinitionID]checkerDefinitionInterval, error) {
	nodes := map[identity.DefinitionID]bool{}
	for definition := range expected.definitions {
		nodes[definition] = true
	}
	for child, parent := range expected.parents {
		if child.IsZero() || child == parent {
			return nil, fmt.Errorf(
				"checker definition containment has invalid edge %s -> %s",
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
		parent := expected.parents[definition]
		if parent.IsZero() {
			roots = append(roots, definition)
			continue
		}
		children[parent] = append(children[parent], definition)
	}
	sortIDs := func(ids []identity.DefinitionID) {
		sort.Slice(ids, func(left, right int) bool {
			return ids[left].String() < ids[right].String()
		})
	}
	sortIDs(roots)
	for parent := range children {
		sortIDs(children[parent])
	}
	type frame struct {
		id    identity.DefinitionID
		close bool
	}
	intervals := map[identity.DefinitionID]checkerDefinitionInterval{}
	state := map[identity.DefinitionID]uint8{}
	clock := 0
	for _, root := range roots {
		stack := []frame{{id: root}}
		for len(stack) != 0 {
			index := len(stack) - 1
			current := stack[index]
			stack = stack[:index]
			if current.close {
				interval := intervals[current.id]
				interval.leave = clock
				clock++
				intervals[current.id] = interval
				state[current.id] = 2
				continue
			}
			switch state[current.id] {
			case 1:
				return nil, fmt.Errorf(
					"checker definition containment cycle at %s",
					current.id,
				)
			case 2:
				continue
			}
			state[current.id] = 1
			intervals[current.id] = checkerDefinitionInterval{
				enter: clock,
			}
			clock++
			stack = append(stack, frame{
				id: current.id, close: true,
			})
			descendants := children[current.id]
			for child := len(descendants) - 1; child >= 0; child-- {
				stack = append(stack, frame{
					id: descendants[child],
				})
			}
		}
	}
	if len(intervals) != len(nodes) {
		return nil, fmt.Errorf(
			"checker definition containment reached %d of %d definitions",
			len(intervals), len(nodes),
		)
	}
	return intervals, nil
}
