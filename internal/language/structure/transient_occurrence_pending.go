package structure

import (
	"fmt"
	"go/ast"
)

func (store *transientOccurrenceStore) bindPending(
	builder *OccurrenceStoreBuilder,
	index OccurrenceIndex,
	node ast.Node,
) error {
	if store == nil || store.sealed || builder == nil ||
		builder.sealed || builder.store == nil ||
		!index.valid() || node == nil {
		return fmt.Errorf(
			"transient pending occurrence binding is invalid",
		)
	}
	occurrence := builder.occurrence(index)
	if occurrence.ID().IsZero() {
		return fmt.Errorf(
			"transient pending occurrence is outside canonical construction storage",
		)
	}
	kind, err := Classify(node)
	if err != nil {
		return err
	}
	if uint16(kind) != occurrence.ID().KindID() {
		return fmt.Errorf(
			"transient pending occurrence %s has node kind %s",
			occurrence.ID(),
			kind,
		)
	}
	nodes := store.pending[builder]
	if len(nodes) < len(builder.store.records) {
		nodes = append(
			nodes,
			make([]ast.Node, len(builder.store.records)-len(nodes))...,
		)
	}
	existing := nodes[index-1]
	if existing != nil && existing != node {
		return fmt.Errorf(
			"transient pending occurrence %s has conflicting nodes",
			occurrence.ID(),
		)
	}
	nodes[index-1] = node
	store.pending[builder] = nodes
	return nil
}

func (store *transientOccurrenceStore) registerPending(
	domain transientOccurrenceDomain,
	builder *OccurrenceStoreBuilder,
	canonical *OccurrenceStore,
	compatible func(ast.Node, ast.Node) bool,
) error {
	if builder == nil || canonical == nil {
		return fmt.Errorf(
			"transient pending occurrence registration is absent",
		)
	}
	nodes, present := store.pending[builder]
	if !present {
		return fmt.Errorf(
			"canonical executable occurrence store has no pending nodes",
		)
	}
	if len(nodes) != canonical.Count() {
		return fmt.Errorf(
			"canonical executable occurrence store has %d nodes for %d records",
			len(nodes),
			canonical.Count(),
		)
	}
	if err := store.register(
		domain,
		canonical,
		nodes,
		compatible,
	); err != nil {
		return err
	}
	delete(store.pending, builder)
	return nil
}
