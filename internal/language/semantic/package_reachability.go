package semantic

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
)

type packageReachability struct {
	declarations       map[identity.SemanticDeclarationID]bool
	bindings           map[identity.SemanticBindingID]bool
	types              map[identity.SemanticTypeID]bool
	operations         map[identity.OperationID]bool
	unsupported        map[identity.UnsupportedID]bool
	declarationRecords map[identity.SemanticDeclarationID]Declaration
	bindingRecords     map[identity.SemanticBindingID]Binding
	typeRecords        map[identity.SemanticTypeID]Type
	operationRecords   map[identity.OperationID]Operation
}

func validatePackageReachability(pkg Package) error {
	reachable := packageReachability{
		declarations:       map[identity.SemanticDeclarationID]bool{},
		bindings:           map[identity.SemanticBindingID]bool{},
		types:              map[identity.SemanticTypeID]bool{},
		operations:         map[identity.OperationID]bool{},
		unsupported:        map[identity.UnsupportedID]bool{},
		declarationRecords: map[identity.SemanticDeclarationID]Declaration{},
		bindingRecords:     map[identity.SemanticBindingID]Binding{},
		typeRecords:        map[identity.SemanticTypeID]Type{},
		operationRecords:   map[identity.OperationID]Operation{},
	}
	for _, record := range pkg.declarations {
		reachable.declarationRecords[record.ID()] = record
	}
	for _, record := range pkg.types {
		reachable.typeRecords[record.ID()] = record
	}
	for _, record := range pkg.bindings {
		reachable.bindingRecords[record.ID()] = record
	}
	for _, record := range pkg.operations {
		reachable.operationRecords[record.ID()] = record
	}
	for _, record := range pkg.definitions {
		spec := record.Spec()
		for _, declaration := range spec.Declarations {
			reachable.markDeclaration(declaration)
		}
		reachable.markBinding(spec.Receiver)
		for _, binding := range spec.Bindings {
			reachable.markBinding(binding)
		}
		reachable.markType(spec.Signature)
	}
	for _, record := range pkg.declarations {
		reachable.markDeclaration(record.ID())
	}
	for _, record := range pkg.bindings {
		reachable.markBinding(record.ID())
	}
	for _, record := range pkg.resolutions {
		reachable.markResolution(record)
	}
	for _, record := range pkg.operations {
		if !record.ID().Source() {
			reachable.markOperation(record.ID())
		}
	}
	if err := requireAllReachable(
		"declaration",
		pkg.declarations,
		func(record Declaration) bool {
			return reachable.declarations[record.ID()]
		},
		func(record Declaration) string { return record.ID().String() },
	); err != nil {
		return err
	}
	if err := requireAllReachable(
		"binding",
		pkg.bindings,
		func(record Binding) bool {
			return reachable.bindings[record.ID()]
		},
		func(record Binding) string { return record.ID().String() },
	); err != nil {
		return err
	}
	if err := requireAllReachable(
		"type",
		pkg.types,
		func(record Type) bool {
			return reachable.types[record.ID()]
		},
		func(record Type) string {
			return fmt.Sprintf(
				"%s (%s)", record.ID(), record.Spec().Kind,
			)
		},
	); err != nil {
		return err
	}
	if err := requireAllReachable(
		"operation",
		pkg.operations,
		func(record Operation) bool {
			return reachable.operations[record.ID()]
		},
		func(record Operation) string { return record.ID().String() },
	); err != nil {
		return err
	}
	return requireAllReachable(
		"unsupported",
		pkg.unsupported,
		func(record Unsupported) bool {
			return reachable.unsupported[record.ID()]
		},
		func(record Unsupported) string { return record.ID().String() },
	)
}

func (reachable *packageReachability) markResolution(
	record OccurrenceResolution,
) {
	switch record.Kind() {
	case ResolutionStructuralOnly:
		reachable.markDeclaration(record.Structural().Declaration())
		reachable.markType(record.Structural().Type())
	case ResolutionDeclaration:
		reachable.markDeclaration(record.Declaration())
	case ResolutionBinding:
		reachable.markBinding(record.Binding())
	case ResolutionType:
		reachable.markType(record.Type())
	case ResolutionOperation:
		reachable.markOperation(record.Operation())
	case ResolutionUnsupported:
		reachable.unsupported[record.Unsupported()] = true
	}
}

func (reachable *packageReachability) markOperation(
	id identity.OperationID,
) {
	if id.IsZero() || reachable.operations[id] {
		return
	}
	reachable.operations[id] = true
	record, present := reachable.operationRecords[id]
	if !present {
		return
	}
	spec := record.Spec()
	reachable.markType(spec.ResultType)
	reachable.markType(spec.ExpectedType)
	reachable.markObject(spec.Object)
	if !spec.Selection.IsZero() {
		reachable.markType(spec.Selection.Receiver())
		reachable.markDeclaration(spec.Selection.Object())
	}
	if !spec.Instance.IsZero() {
		reachable.markObject(spec.Instance.Target())
		for _, typeID := range spec.Instance.Types() {
			reachable.markType(typeID)
		}
		reachable.markType(spec.Instance.Signature())
	}
	for _, implicit := range spec.Implicit {
		reachable.markType(implicit.Source())
		reachable.markType(implicit.Target())
	}
	reachable.markBinding(spec.Label)
	reachable.markOperation(spec.ControlTarget)
}

func (reachable *packageReachability) markObject(
	record ObjectReference,
) {
	switch record.Kind() {
	case ObjectReferenceDeclaration:
		reachable.markDeclaration(record.Declaration())
	case ObjectReferenceBinding:
		reachable.markBinding(record.Binding())
	}
}

func (reachable *packageReachability) markDeclaration(
	id identity.SemanticDeclarationID,
) {
	if id.IsZero() || reachable.declarations[id] {
		return
	}
	if id.Form() == identity.SemanticDeclarationMember {
		reachable.markType(id.OwnerType())
	}
	record, present := reachable.declarationRecords[id]
	if !present {
		return
	}
	reachable.declarations[id] = true
	reachable.markType(record.Type())
}

func (reachable *packageReachability) markBinding(
	id identity.SemanticBindingID,
) {
	if id.IsZero() || reachable.bindings[id] {
		return
	}
	record, present := reachable.bindingRecords[id]
	if !present {
		return
	}
	reachable.bindings[id] = true
	reachable.markType(record.Type())
}

func (reachable *packageReachability) markType(
	id identity.SemanticTypeID,
) {
	if id.IsZero() || reachable.types[id] {
		return
	}
	reachable.types[id] = true
	record, present := reachable.typeRecords[id]
	if !present {
		return
	}
	spec := record.Spec()
	reachable.markDeclaration(spec.Declaration)
	reachable.markDeclaration(spec.Parameter.Declaration())
	typeIDs := []identity.SemanticTypeID{
		spec.Underlying,
		spec.Target,
		spec.Constraint,
		spec.Element,
		spec.Key,
		spec.Signature.Receiver,
	}
	typeIDs = append(typeIDs, spec.Arguments...)
	typeIDs = append(
		typeIDs, spec.Signature.ReceiverTypeParameters...,
	)
	typeIDs = append(typeIDs, spec.Signature.TypeParameters...)
	typeIDs = append(typeIDs, spec.Signature.Parameters...)
	typeIDs = append(typeIDs, spec.Signature.Results...)
	typeIDs = append(typeIDs, spec.Embeddeds...)
	typeIDs = append(typeIDs, spec.Elements...)
	for _, field := range spec.Fields {
		typeIDs = append(typeIDs, field.Type)
		declaration, err := identity.NewMemberDeclarationID(
			record.ID(),
			field.Package,
			identity.SemanticObjectField,
			field.Name,
			field.Ordinal,
		)
		if err == nil {
			reachable.markDeclaration(declaration)
		}
	}
	for _, method := range spec.Methods {
		typeIDs = append(typeIDs, method.Signature)
		declaration, err := identity.NewMemberDeclarationID(
			record.ID(),
			method.Package,
			identity.SemanticObjectMethod,
			method.Name,
			0,
		)
		if err == nil {
			reachable.markDeclaration(declaration)
		}
	}
	for _, term := range spec.Terms {
		typeIDs = append(typeIDs, term.Type)
	}
	for _, typeID := range typeIDs {
		reachable.markType(typeID)
	}
}

func requireAllReachable[Record any](
	class string,
	records []Record,
	reached func(Record) bool,
	identityOf func(Record) string,
) error {
	for _, record := range records {
		if !reached(record) {
			return fmt.Errorf(
				"semantic package contains unreachable %s %s",
				class, identityOf(record),
			)
		}
	}
	return nil
}
