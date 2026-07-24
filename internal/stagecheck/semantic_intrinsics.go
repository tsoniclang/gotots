package stagecheck

import (
	"fmt"
	"go/constant"
	"go/types"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/selectionfacts"
	"github.com/tsoniclang/gotots/internal/language/semantic"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/source"
)

func verifyIntrinsicSemanticPackage(
	actual semantic.Package,
	expected semanticPackageExpectation,
	universe *source.Universe,
	facts *selectionfacts.Artifact,
) error {
	switch expected.loaded.Disposition() {
	case source.DispositionBuiltinUniverse:
		authority, err := expectedCheckerAuthority(
			universe,
			structure.PackageGraph{},
			expected.loaded,
			facts,
		)
		if err != nil {
			return err
		}
		return verifyBuiltinSemanticPackage(
			actual,
			authority,
		)
	case source.DispositionUnsafeIntrinsic:
		return verifyUnsafeSemanticPackage(actual)
	default:
		return nil
	}
}

func verifyBuiltinSemanticPackage(
	actual semantic.Package,
	authority semantic.Authority,
) error {
	if actual.DefinitionCount() != 0 ||
		actual.ResolutionCount() != 0 ||
		actual.BindingCount() != 0 ||
		actual.OperationCount() != 0 ||
		actual.UnsupportedCount() != 0 {
		return semanticVerificationError(
			"builtin",
			"builtin pseudo-package contains source-shaped semantics",
		)
	}
	catalogMembers := catalog.AllPredeclared()
	if actual.DeclarationCount() != len(catalogMembers) ||
		len(types.Universe.Names()) != len(catalogMembers) {
		return semanticVerificationError(
			"builtin",
			fmt.Sprintf(
				"declarations=%d universe=%d catalog=%d",
				actual.DeclarationCount(),
				len(types.Universe.Names()),
				len(catalogMembers),
			),
		)
	}
	for _, member := range catalogMembers {
		object := types.Universe.Lookup(member.Name())
		class := predeclaredObjectClass(member.Class())
		id, err := identity.NewPredeclaredDeclarationID(
			uint16(member), class,
		)
		if err != nil {
			return semanticVerificationError("builtin", err.Error())
		}
		record, present := actual.Declaration(id)
		if object == nil ||
			!present ||
			record.Package() != actual.ID() ||
			record.Name() != member.Name() ||
			record.Class() != class ||
			record.Exported() != object.Exported() ||
			record.Authority() != authority {
			return semanticVerificationError(
				"builtin",
				"catalog, checker, and semantic declaration differ for "+
					member.Name(),
			)
		}
		if err := verifyIntrinsicDeclarationValue(
			record, object,
		); err != nil {
			return err
		}
	}
	for _, name := range types.Universe.Names() {
		if !catalogedPredeclaredName(name) {
			return semanticVerificationError(
				"builtin",
				"toolchain universe has uncataloged declaration "+name,
			)
		}
	}
	return nil
}

func catalogedPredeclaredName(name string) bool {
	for _, member := range catalog.AllPredeclared() {
		if member.Name() == name {
			return true
		}
	}
	return false
}

func verifyUnsafeSemanticPackage(actual semantic.Package) error {
	members := catalog.AllUnsafeMembers()
	scope := types.Unsafe.Scope()
	if actual.DeclarationCount() != len(members) ||
		len(scope.Names()) != len(members) {
		return semanticVerificationError(
			"unsafe",
			fmt.Sprintf(
				"declarations=%d toolchain=%d catalog=%d",
				actual.DeclarationCount(),
				len(scope.Names()),
				len(members),
			),
		)
	}
	for _, member := range members {
		object := scope.Lookup(member.Name())
		class := identity.SemanticObjectBuiltin
		if member.Class() == catalog.UnsafeMemberClassType {
			class = identity.SemanticObjectType
		}
		id, err := identity.NewPackageDeclarationID(
			actual.ID(), class, member.Name(),
		)
		if err != nil {
			return semanticVerificationError("unsafe", err.Error())
		}
		record, present := actual.Declaration(id)
		if object == nil || !present ||
			record.Package() != actual.ID() ||
			record.Name() != object.Name() ||
			record.Exported() != object.Exported() {
			return semanticVerificationError(
				"unsafe",
				"catalog, checker, and semantic declaration differ for "+
					member.Name(),
			)
		}
		switch member.Class() {
		case catalog.UnsafeMemberClassType:
			if _, ok := object.(*types.TypeName); !ok ||
				record.Class() != identity.SemanticObjectType ||
				record.Type().IsZero() {
				return semanticVerificationError(
					"unsafe",
					"unsafe type semantics differ for "+member.Name(),
				)
			}
		case catalog.UnsafeMemberClassBuiltin:
			if _, ok := object.(*types.Builtin); !ok ||
				record.Class() != identity.SemanticObjectBuiltin ||
				!record.Type().IsZero() {
				return semanticVerificationError(
					"unsafe",
					"unsafe builtin semantics differ for "+member.Name(),
				)
			}
		default:
			return semanticVerificationError(
				"unsafe", "invalid catalog member "+member.Name(),
			)
		}
	}
	for _, name := range scope.Names() {
		if !catalog.UnsafeMemberByName(name).Valid() {
			return semanticVerificationError(
				"unsafe",
				"toolchain unsafe scope has uncataloged declaration "+name,
			)
		}
	}
	return nil
}

func predeclaredObjectClass(
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

func verifyIntrinsicDeclarationValue(
	record semantic.Declaration,
	object types.Object,
) error {
	_, builtin := object.(*types.Builtin)
	if builtin != record.Type().IsZero() {
		return semanticVerificationError(
			"builtin",
			"type presence differs for "+record.Name(),
		)
	}
	value, isConstant := object.(*types.Const)
	if !isConstant {
		if !record.Constant().IsZero() {
			return semanticVerificationError(
				"builtin",
				"non-constant carries a value "+record.Name(),
			)
		}
		return nil
	}
	if record.Constant().Exact() != value.Val().ExactString() ||
		record.Constant().Kind() != semanticConstantKind(
			value.Val().Kind(),
		) {
		return semanticVerificationError(
			"builtin",
			"constant differs for "+record.Name(),
		)
	}
	return nil
}

func semanticConstantKind(kind constant.Kind) semantic.ConstantKind {
	switch kind {
	case constant.Bool:
		return semantic.ConstantBool
	case constant.String:
		return semantic.ConstantString
	case constant.Int:
		return semantic.ConstantInteger
	case constant.Float:
		return semantic.ConstantFloat
	case constant.Complex:
		return semantic.ConstantComplex
	default:
		return 0
	}
}
