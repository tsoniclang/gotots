package structure

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
)

func validateOccurrence(
	reference OccurrenceRef,
	lookup func(identity.OccurrenceID) (OccurrenceRef, bool),
) error {
	occurrence := reference.Occurrence()
	canonical, err := NewOccurrence(
		occurrence.id,
		occurrence.kind,
		occurrence.parent,
		occurrence.edge,
		occurrence.ordinal,
		occurrence.span,
		occurrence.display,
		occurrence.token,
	)
	if err != nil || canonical != occurrence {
		return fmt.Errorf(
			"occurrence %s has noncanonical identity facts",
			occurrence.id,
		)
	}
	if occurrence.parent.IsZero() {
		if occurrence.edge != catalog.EdgeInvalid ||
			occurrence.ordinal != 0 {
			return fmt.Errorf(
				"root occurrence %s has child-edge facts",
				occurrence.id,
			)
		}
		return nil
	}
	parent, present := lookup(occurrence.parent)
	if !present {
		return fmt.Errorf(
			"occurrence %s has absent parent %s",
			occurrence.id,
			occurrence.parent,
		)
	}
	if !occurrence.edge.Valid() ||
		occurrence.edge.Parent() != parent.Kind() ||
		(!occurrence.edge.IsList() &&
			occurrence.ordinal != 0) {
		return fmt.Errorf(
			"occurrence %s has invalid parent edge %s",
			occurrence.id,
			occurrence.edge,
		)
	}
	return nil
}
