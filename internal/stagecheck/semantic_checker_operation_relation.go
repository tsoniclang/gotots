package stagecheck

import (
	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/semantic"
)

func operationOperandsEqual(
	operation semantic.Operation,
	expected []identity.OccurrenceID,
) bool {
	if operation.OperandCount() != len(expected) {
		return false
	}
	for index, expectedID := range expected {
		actual, present := operation.Operand(index)
		if !present || actual != expectedID {
			return false
		}
	}
	return true
}

func operationDefinitionsEqual(
	operation semantic.Operation,
	expected []identity.DefinitionID,
) bool {
	if operation.NestedDefinitionCount() != len(expected) {
		return false
	}
	for index, expectedID := range expected {
		actual, present := operation.NestedDefinition(index)
		if !present || actual != expectedID {
			return false
		}
	}
	return true
}

func operationOperandIdentities(
	operation semantic.Operation,
) []identity.OccurrenceID {
	out := make(
		[]identity.OccurrenceID,
		0,
		operation.OperandCount(),
	)
	for index := 0; index < operation.OperandCount(); index++ {
		id, _ := operation.Operand(index)
		out = append(out, id)
	}
	return out
}

func operationDefinitionIdentities(
	operation semantic.Operation,
) []identity.DefinitionID {
	out := make(
		[]identity.DefinitionID,
		0,
		operation.NestedDefinitionCount(),
	)
	for index := 0; index < operation.NestedDefinitionCount(); index++ {
		id, _ := operation.NestedDefinition(index)
		out = append(out, id)
	}
	return out
}
