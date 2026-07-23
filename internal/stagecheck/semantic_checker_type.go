package stagecheck

import (
	"fmt"
	"go/types"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/semantic"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/language/typesemantics"
	"github.com/tsoniclang/gotots/internal/source"
)

type checkerTypePair struct {
	id  identity.SemanticTypeID
	typ types.Type
}

type checkerTypeVerifier struct {
	expected           semanticPackageExpectation
	types              map[identity.SemanticTypeID]semantic.Type
	declarations       map[identity.SemanticDeclarationID]semantic.Declaration
	packageByPath      map[string]identity.PackageID
	visiting           map[checkerTypePair]bool
	verified           map[checkerTypePair]bool
	parameterOwners    map[*types.TypeParam]checkerTypeParameterOwner
	parameterLocations map[checkerTypeParameterLocation]checkerTypeParameterOwner
	parameterConflict  string
}

func newCheckerTypeVerifier(
	expected semanticPackageExpectation,
	actual semantic.Package,
	universe *source.Universe,
	index *structure.TransientIndex,
) *checkerTypeVerifier {
	out := &checkerTypeVerifier{
		expected:           expected,
		types:              map[identity.SemanticTypeID]semantic.Type{},
		declarations:       map[identity.SemanticDeclarationID]semantic.Declaration{},
		packageByPath:      map[string]identity.PackageID{},
		visiting:           map[checkerTypePair]bool{},
		verified:           map[checkerTypePair]bool{},
		parameterOwners:    map[*types.TypeParam]checkerTypeParameterOwner{},
		parameterLocations: map[checkerTypeParameterLocation]checkerTypeParameterOwner{},
	}
	for _, record := range actual.Types() {
		out.types[record.ID()] = record
	}
	for _, record := range actual.Declarations() {
		out.declarations[record.ID()] = record
	}
	for _, pkg := range universe.Packages() {
		out.packageByPath[pkg.ID().ImportPath()] = pkg.ID()
	}
	out.indexTypeParameterOwners(universe, index)
	return out
}

func (verifier *checkerTypeVerifier) verify(
	id identity.SemanticTypeID,
	typ types.Type,
) error {
	if id.IsZero() || typ == nil {
		if id.IsZero() && typ == nil {
			return nil
		}
		return fmt.Errorf(
			"type presence differs: semantic=%s checker=%T",
			id, typ,
		)
	}
	pair := checkerTypePair{id: id, typ: typ}
	if verifier.verified[pair] || verifier.visiting[pair] {
		return nil
	}
	record, present := verifier.types[id]
	if !present {
		return fmt.Errorf("semantic type %s is absent", id)
	}
	verifier.visiting[pair] = true
	err := verifier.verifyRecord(record, typ)
	delete(verifier.visiting, pair)
	if err != nil {
		return fmt.Errorf(
			"type %s differs from checker %T: %w",
			id, typ, err,
		)
	}
	verifier.verified[pair] = true
	return nil
}

func (verifier *checkerTypeVerifier) verifyRecord(
	record semantic.Type,
	typ types.Type,
) error {
	spec := record.Spec()
	switch typed := typ.(type) {
	case *types.Named:
		if spec.Kind != semantic.TypeNamed {
			return checkerTypeKindError(spec.Kind, semantic.TypeNamed)
		}
		verifier.indexEncounteredNamedTypeParameters(typed)
		if err := verifier.verifyNominalDeclaration(
			spec.Declaration, typed.Obj(),
			identity.SemanticObjectType,
		); err != nil {
			return err
		}
		if err := verifier.verifyTypeList(
			spec.Arguments, typed.TypeArgs(),
		); err != nil {
			return err
		}
		if err := verifier.verify(
			spec.Underlying, typed.Underlying(),
		); err != nil {
			return err
		}
		return verifier.verifyNamedMethods(spec.Methods, typed)
	case *types.Alias:
		if spec.Kind != semantic.TypeAlias {
			return checkerTypeKindError(spec.Kind, semantic.TypeAlias)
		}
		verifier.indexEncounteredAliasTypeParameters(typed)
		if err := verifier.verifyNominalDeclaration(
			spec.Declaration, typed.Obj(),
			identity.SemanticObjectAlias,
		); err != nil {
			return err
		}
		if err := verifier.verifyTypeList(
			spec.Arguments, typed.TypeArgs(),
		); err != nil {
			return err
		}
		return verifier.verify(spec.Target, types.Unalias(typed))
	case *types.TypeParam:
		if spec.Kind != semantic.TypeParameter {
			return checkerTypeKindError(
				spec.Kind, semantic.TypeParameter,
			)
		}
		if err := verifier.verifyTypeParameterOwner(
			spec.Parameter, typed,
		); err != nil {
			return err
		}
		return verifier.verify(
			spec.Constraint, typed.Constraint(),
		)
	}
	switch typed := typ.(type) {
	case *types.Basic:
		if spec.Kind != semantic.TypeBasic ||
			spec.Basic != independentBasicKind(typed.Kind()) {
			return fmt.Errorf("basic kind differs")
		}
	case *types.Pointer:
		if spec.Kind != semantic.TypePointer {
			return checkerTypeKindError(
				spec.Kind, semantic.TypePointer,
			)
		}
		return verifier.verify(spec.Element, typed.Elem())
	case *types.Slice:
		if spec.Kind != semantic.TypeSlice {
			return checkerTypeKindError(spec.Kind, semantic.TypeSlice)
		}
		return verifier.verify(spec.Element, typed.Elem())
	case *types.Array:
		if spec.Kind != semantic.TypeArray ||
			spec.Length != typed.Len() {
			return fmt.Errorf("array shape differs")
		}
		return verifier.verify(spec.Element, typed.Elem())
	case *types.Map:
		if spec.Kind != semantic.TypeMap {
			return checkerTypeKindError(spec.Kind, semantic.TypeMap)
		}
		if err := verifier.verify(spec.Key, typed.Key()); err != nil {
			return err
		}
		return verifier.verify(spec.Element, typed.Elem())
	case *types.Chan:
		if spec.Kind != semantic.TypeChannel ||
			spec.Direction != independentChannelDirection(typed.Dir()) {
			return fmt.Errorf("channel direction differs")
		}
		return verifier.verify(spec.Element, typed.Elem())
	case *types.Signature:
		if spec.Kind != semantic.TypeSignature {
			return checkerTypeKindError(
				spec.Kind, semantic.TypeSignature,
			)
		}
		return verifier.verifySignature(spec.Signature, typed)
	case *types.Struct:
		if spec.Kind != semantic.TypeStruct {
			return checkerTypeKindError(spec.Kind, semantic.TypeStruct)
		}
		return verifier.verifyStruct(spec.Fields, typed)
	case *types.Interface:
		if spec.Kind != semantic.TypeInterface {
			return checkerTypeKindError(
				spec.Kind, semantic.TypeInterface,
			)
		}
		return verifier.verifyInterface(spec, typed)
	case *types.Tuple:
		if spec.Kind != semantic.TypeTuple {
			return checkerTypeKindError(spec.Kind, semantic.TypeTuple)
		}
		return verifier.verifyTuple(spec.Elements, typed)
	case *types.Union:
		if spec.Kind != semantic.TypeUnion {
			return checkerTypeKindError(spec.Kind, semantic.TypeUnion)
		}
		return verifier.verifyUnion(spec.Terms, typed)
	default:
		return fmt.Errorf("unsupported checker type %T", typ)
	}
	return nil
}

func (verifier *checkerTypeVerifier) verifyNominalDeclaration(
	id identity.SemanticDeclarationID,
	object *types.TypeName,
	class identity.SemanticObjectClass,
) error {
	if object == nil {
		return fmt.Errorf("nominal type has no package object")
	}
	if object.Pkg() == nil {
		predeclared := independentPredeclaredKind(object)
		predeclaredClass := independentPredeclaredClass(
			predeclared.Class(),
		)
		expected, err := identity.NewPredeclaredDeclarationID(
			uint16(predeclared), predeclaredClass,
		)
		if !predeclared.Valid() ||
			!predeclaredClass.Valid() ||
			err != nil ||
			expected != id {
			return fmt.Errorf(
				"predeclared nominal declaration differs: %s vs %s",
				id, expected,
			)
		}
		return nil
	}
	pkg := verifier.packageByPath[object.Pkg().Path()]
	expected, err := identity.NewPackageDeclarationID(
		pkg, class, object.Name(),
	)
	if err != nil || expected != id {
		return fmt.Errorf(
			"nominal declaration differs: %s vs %s",
			id, expected,
		)
	}
	return nil
}

func independentPredeclaredClass(
	class catalog.PredeclaredClass,
) identity.SemanticObjectClass {
	switch class {
	case catalog.PredeclaredClassType:
		return identity.SemanticObjectType
	case catalog.PredeclaredClassConstant:
		return identity.SemanticObjectConstant
	case catalog.PredeclaredClassNil:
		return identity.SemanticObjectNil
	case catalog.PredeclaredClassFunction:
		return identity.SemanticObjectBuiltin
	default:
		return identity.SemanticObjectInvalid
	}
}

func (verifier *checkerTypeVerifier) verifyTypeList(
	ids []identity.SemanticTypeID,
	typesList *types.TypeList,
) error {
	length := 0
	if typesList != nil {
		length = typesList.Len()
	}
	if len(ids) != length {
		return fmt.Errorf(
			"type argument count %d differs from %d",
			len(ids), length,
		)
	}
	for index, id := range ids {
		if err := verifier.verify(
			id, typesList.At(index),
		); err != nil {
			return err
		}
	}
	return nil
}

func (verifier *checkerTypeVerifier) verifyTuple(
	ids []identity.SemanticTypeID,
	tuple *types.Tuple,
) error {
	length := 0
	if tuple != nil {
		length = tuple.Len()
	}
	if len(ids) != length {
		return fmt.Errorf(
			"tuple length %d differs from %d", len(ids), length,
		)
	}
	for index, id := range ids {
		if err := verifier.verify(
			id, tuple.At(index).Type(),
		); err != nil {
			return err
		}
	}
	return nil
}

func independentBasicKind(kind types.BasicKind) semantic.BasicKind {
	switch kind {
	case types.Bool:
		return semantic.BasicBool
	case types.Int:
		return semantic.BasicInt
	case types.Int8:
		return semantic.BasicInt8
	case types.Int16:
		return semantic.BasicInt16
	case types.Int32:
		return semantic.BasicInt32
	case types.Int64:
		return semantic.BasicInt64
	case types.Uint:
		return semantic.BasicUint
	case types.Uint8:
		return semantic.BasicUint8
	case types.Uint16:
		return semantic.BasicUint16
	case types.Uint32:
		return semantic.BasicUint32
	case types.Uint64:
		return semantic.BasicUint64
	case types.Uintptr:
		return semantic.BasicUintptr
	case types.Float32:
		return semantic.BasicFloat32
	case types.Float64:
		return semantic.BasicFloat64
	case types.Complex64:
		return semantic.BasicComplex64
	case types.Complex128:
		return semantic.BasicComplex128
	case types.String:
		return semantic.BasicString
	case types.UnsafePointer:
		return semantic.BasicUnsafePointer
	case types.UntypedBool:
		return semantic.BasicUntypedBool
	case types.UntypedInt:
		return semantic.BasicUntypedInt
	case types.UntypedRune:
		return semantic.BasicUntypedRune
	case types.UntypedFloat:
		return semantic.BasicUntypedFloat
	case types.UntypedComplex:
		return semantic.BasicUntypedComplex
	case types.UntypedString:
		return semantic.BasicUntypedString
	case types.UntypedNil:
		return semantic.BasicUntypedNil
	default:
		return semantic.BasicInvalid
	}
}

func independentChannelDirection(
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

func independentTypeSetKind(
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

func checkerTypeKindError(
	got semantic.TypeKind,
	want semantic.TypeKind,
) error {
	return fmt.Errorf("type kind %s differs from %s", got, want)
}
