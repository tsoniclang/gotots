package stagecheck

import (
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/semantic"
)

func semanticBindingPresent(
	records []semantic.Binding,
	id identity.SemanticBindingID,
) bool {
	index := sort.Search(len(records), func(index int) bool {
		return records[index].ID().String() >= id.String()
	})
	return index < len(records) && records[index].ID() == id
}

func semanticTypePresent(
	records []semantic.Type,
	id identity.SemanticTypeID,
) bool {
	index := sort.Search(len(records), func(index int) bool {
		return records[index].ID().String() >= id.String()
	})
	return index < len(records) && records[index].ID() == id
}

func semanticOperationPresent(
	records []semantic.Operation,
	id identity.OperationID,
) bool {
	index := sort.Search(len(records), func(index int) bool {
		return records[index].ID().String() >= id.String()
	})
	return index < len(records) && records[index].ID() == id
}

func semanticUnsupportedPresent(
	records []semantic.Unsupported,
	id identity.UnsupportedID,
) bool {
	index := sort.Search(len(records), func(index int) bool {
		return records[index].ID().String() >= id.String()
	})
	return index < len(records) && records[index].ID() == id
}

func semanticResolutionPresent(
	records []semantic.OccurrenceResolution,
	id identity.OccurrenceID,
) bool {
	index := sort.Search(len(records), func(index int) bool {
		return records[index].Occurrence().String() >= id.String()
	})
	return index < len(records) &&
		records[index].Occurrence() == id
}
