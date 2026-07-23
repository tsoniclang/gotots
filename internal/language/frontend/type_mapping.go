package frontend

import (
	"fmt"
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/semantic"
	"github.com/tsoniclang/gotots/internal/language/typesemantics"
)

func (builder *typeBuilder) memberPackage(
	object types.Object,
) (identity.PackageID, error) {
	if object.Exported() {
		return identity.PackageID{}, nil
	}
	return builder.objects.packageID(object.Pkg())
}

func (builder *typeBuilder) memberOwnerID(
	object types.Object,
) (identity.SemanticTypeID, error) {
	owners := builder.objects.memberOwnerRelations[object]
	if len(owners) != 1 {
		return identity.SemanticTypeID{}, fmt.Errorf(
			"member %s (%T, type=%s, package=%v, position=%d) has %d declaring owner types; contextual owner required",
			object.Name(),
			object,
			types.TypeString(object.Type(), nil),
			object.Pkg(),
			object.Pos(),
			len(owners),
		)
	}
	return builder.memberOwnerTypeID(owners[0])
}

func (builder *typeBuilder) memberOwnerTypeID(
	owner types.Type,
) (identity.SemanticTypeID, error) {
	if owner == nil {
		return identity.SemanticTypeID{}, fmt.Errorf(
			"semantic member owner type is absent",
		)
	}
	for {
		pointer, ok := owner.(*types.Pointer)
		if !ok {
			break
		}
		owner = pointer.Elem()
	}
	if named, ok := owner.(*types.Named); ok {
		declaration, err := builder.objects.declarationID(
			named.Obj(),
		)
		if err != nil {
			return identity.SemanticTypeID{}, err
		}
		return semantic.NominalTypeID(
			semantic.TypeNamed,
			declaration,
			nil,
		)
	}
	if owner == nil {
		return identity.SemanticTypeID{}, fmt.Errorf(
			"semantic member has no declaring owner type",
		)
	}
	return builder.build(owner)
}

func (builder *typeBuilder) admit(
	goType types.Type,
	record semantic.Type,
) error {
	if existing, present := builder.records[record.ID()]; present &&
		existing.Canonical() != record.Canonical() {
		return fmt.Errorf(
			"semantic type identity collision %s", record.ID(),
		)
	}
	builder.records[record.ID()] = record
	builder.byGoType[goType] = record.ID()
	return nil
}

func (builder *typeBuilder) recordsSorted() []semantic.Type {
	out := make([]semantic.Type, 0, len(builder.records))
	for _, record := range builder.records {
		out = append(out, record)
	}
	sort.Slice(out, func(left, right int) bool {
		return out[left].ID().String() < out[right].ID().String()
	})
	return out
}

func (builder *typeBuilder) finish() error {
	for len(builder.pending) != 0 {
		pending := make([]types.Type, 0, len(builder.pending))
		for typ := range builder.pending {
			pending = append(pending, typ)
		}
		sort.Slice(pending, func(left, right int) bool {
			return builder.byGoType[pending[left]].String() <
				builder.byGoType[pending[right]].String()
		})
		progress := false
		for _, typ := range pending {
			if _, err := builder.build(typ); err != nil {
				return err
			}
			if !builder.pending[typ] {
				progress = true
			}
		}
		if !progress {
			return fmt.Errorf(
				"semantic nominal descriptor queue made no progress",
			)
		}
	}
	return nil
}

func semanticBasic(
	kind types.BasicKind,
) (semantic.BasicKind, error) {
	switch kind {
	case types.Bool:
		return semantic.BasicBool, nil
	case types.Int:
		return semantic.BasicInt, nil
	case types.Int8:
		return semantic.BasicInt8, nil
	case types.Int16:
		return semantic.BasicInt16, nil
	case types.Int32:
		return semantic.BasicInt32, nil
	case types.Int64:
		return semantic.BasicInt64, nil
	case types.Uint:
		return semantic.BasicUint, nil
	case types.Uint8:
		return semantic.BasicUint8, nil
	case types.Uint16:
		return semantic.BasicUint16, nil
	case types.Uint32:
		return semantic.BasicUint32, nil
	case types.Uint64:
		return semantic.BasicUint64, nil
	case types.Uintptr:
		return semantic.BasicUintptr, nil
	case types.Float32:
		return semantic.BasicFloat32, nil
	case types.Float64:
		return semantic.BasicFloat64, nil
	case types.Complex64:
		return semantic.BasicComplex64, nil
	case types.Complex128:
		return semantic.BasicComplex128, nil
	case types.String:
		return semantic.BasicString, nil
	case types.UnsafePointer:
		return semantic.BasicUnsafePointer, nil
	case types.UntypedBool:
		return semantic.BasicUntypedBool, nil
	case types.UntypedInt:
		return semantic.BasicUntypedInt, nil
	case types.UntypedRune:
		return semantic.BasicUntypedRune, nil
	case types.UntypedFloat:
		return semantic.BasicUntypedFloat, nil
	case types.UntypedComplex:
		return semantic.BasicUntypedComplex, nil
	case types.UntypedString:
		return semantic.BasicUntypedString, nil
	case types.UntypedNil:
		return semantic.BasicUntypedNil, nil
	default:
		return semantic.BasicInvalid, fmt.Errorf(
			"unsupported basic type kind %d", kind,
		)
	}
}

func semanticChannelDirection(
	direction types.ChanDir,
) semantic.ChannelDirection {
	switch direction {
	case types.SendRecv:
		return semantic.ChannelSendReceive
	case types.SendOnly:
		return semantic.ChannelSendOnly
	case types.RecvOnly:
		return semantic.ChannelReceiveOnly
	default:
		return semantic.ChannelInvalid
	}
}

func semanticTypeSetKind(
	kind typesemantics.SetKind,
) semantic.TypeSetKind {
	switch kind {
	case typesemantics.SetUniverse:
		return semantic.TypeSetUniverse
	case typesemantics.SetFinite:
		return semantic.TypeSetFinite
	case typesemantics.SetEmpty:
		return semantic.TypeSetEmpty
	default:
		return semantic.TypeSetInvalid
	}
}
