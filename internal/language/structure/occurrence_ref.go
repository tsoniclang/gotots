package structure

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
)

// OccurrenceRef is an immutable reference to one owner-held canonical
// occurrence payload. It prevents downstream stages from copying that payload
// into a second record while exposing only read methods.
type OccurrenceRef struct {
	occurrence *Occurrence
}

func NewOccurrenceRef(occurrence *Occurrence) (OccurrenceRef, error) {
	if occurrence == nil || occurrence.ID().IsZero() {
		return OccurrenceRef{}, fmt.Errorf(
			"occurrence reference requires canonical payload",
		)
	}
	return OccurrenceRef{occurrence: occurrence}, nil
}

func (reference OccurrenceRef) ID() identity.OccurrenceID {
	if reference.occurrence == nil {
		return identity.OccurrenceID{}
	}
	return reference.occurrence.ID()
}

func (reference OccurrenceRef) Kind() catalog.Kind {
	if reference.occurrence == nil {
		return catalog.KindInvalid
	}
	return reference.occurrence.Kind()
}

func (reference OccurrenceRef) Parent() identity.OccurrenceID {
	if reference.occurrence == nil {
		return identity.OccurrenceID{}
	}
	return reference.occurrence.Parent()
}

func (reference OccurrenceRef) Edge() catalog.Edge {
	if reference.occurrence == nil {
		return catalog.EdgeInvalid
	}
	return reference.occurrence.Edge()
}

func (reference OccurrenceRef) Role() catalog.Role {
	if reference.occurrence == nil {
		return catalog.RoleInvalid
	}
	return reference.occurrence.Role()
}

func (reference OccurrenceRef) Ordinal() int {
	if reference.occurrence == nil {
		return 0
	}
	return reference.occurrence.Ordinal()
}

func (reference OccurrenceRef) Span() Span {
	if reference.occurrence == nil {
		return Span{}
	}
	return reference.occurrence.Span()
}

func (reference OccurrenceRef) Display() DisplaySpan {
	if reference.occurrence == nil {
		return DisplaySpan{}
	}
	return reference.occurrence.Display()
}

func (reference OccurrenceRef) Token() catalog.TokenKind {
	if reference.occurrence == nil {
		return catalog.TokenInvalid
	}
	return reference.occurrence.Token()
}
