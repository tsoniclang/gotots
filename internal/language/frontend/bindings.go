package frontend

import (
	"fmt"
	"go/ast"
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/semantic"
)

func (index *objectIndex) visitBindingRecords(
	visit func(semantic.Binding) error,
) (int, error) {
	if visit == nil {
		return 0, fmt.Errorf(
			"semantic binding record visitor is absent",
		)
	}
	captures, err := index.bindingCaptures()
	if err != nil {
		return 0, err
	}
	type identified struct {
		id        identity.SemanticBindingID
		candidate *bindingCandidate
	}
	records := make([]identified, 0, len(index.bindingIDs))
	for candidate, id := range index.bindingIDs {
		records = append(records, identified{
			id: id, candidate: candidate,
		})
	}
	sort.Slice(records, func(left, right int) bool {
		return records[left].id.Compare(records[right].id) < 0
	})
	for _, item := range records {
		typeID := identity.SemanticTypeID{}
		if item.candidate.typ != nil {
			var err error
			typeID, err = index.typeBuilder.build(
				item.candidate.typ,
			)
			if err != nil {
				return 0, fmt.Errorf(
					"binding %s role=%s name=%q: %w",
					item.id, item.candidate.role,
					item.candidate.name, err,
				)
			}
		}
		record, err := semantic.NewBinding(
			item.id,
			index.input.id,
			item.candidate.definition,
			item.candidate.role,
			item.candidate.name,
			typeID,
			item.candidate.source,
			captures[item.candidate],
			index.input.authority,
		)
		if err != nil {
			return 0, err
		}
		if err := visit(record); err != nil {
			return 0, err
		}
	}
	return len(records), nil
}

func (index *objectIndex) bindingCaptures() (map[*bindingCandidate][]identity.DefinitionID, error) {
	sets := map[*bindingCandidate]map[identity.DefinitionID]bool{}
	view := index.input.loaded.CheckerView()
	for _, occurrenceID := range index.input.order {
		index.work.CaptureOccurrenceVisits++
		record := index.input.occurrence(occurrenceID)
		identifier, ok := record.node.(*ast.Ident)
		if !ok {
			continue
		}
		object, present := view.UseOf(identifier)
		if !present || object == nil {
			continue
		}
		candidate := index.bindingByObject[object]
		if candidate == nil ||
			!semantic.BindingRoleCanBeCaptured(candidate.role) ||
			candidate.definition.IsZero() ||
			record.owner.IsZero() ||
			record.owner == candidate.definition {
			continue
		}
		index.work.ContainmentProbes++
		if !index.input.containment.contains(
			candidate.definition, record.owner,
		) {
			return nil, fmt.Errorf(
				"binding %s is used by unrelated definition %s",
				object.Name(), record.owner,
			)
		}
		if sets[candidate] == nil {
			sets[candidate] = map[identity.DefinitionID]bool{}
		}
		sets[candidate][record.owner] = true
	}
	out := map[*bindingCandidate][]identity.DefinitionID{}
	for candidate, set := range sets {
		for definition := range set {
			out[candidate] = append(
				out[candidate], definition,
			)
		}
		sort.Slice(out[candidate], func(left, right int) bool {
			return out[candidate][left].Compare(
				out[candidate][right],
			) < 0
		})
	}
	return out, nil
}

func objectUse(
	view interface {
		UseOf(*ast.Ident) (types.Object, bool)
	},
	node ast.Node,
) types.Object {
	identifier, ok := node.(*ast.Ident)
	if !ok {
		return nil
	}
	object, _ := view.UseOf(identifier)
	return object
}
