package ordering

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func CompareBasicKinds(left types.BasicKind, right types.BasicKind) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

func StableTypeString(source types.Type) string {
	return types.TypeString(source, func(sourcePackage *types.Package) string {
		if sourcePackage == nil {
			return ""
		}
		return sourcePackage.Path()
	})
}

func CompareObjects(left types.Object, right types.Object) int {
	return compareObjects(left, right, StableTypeString)
}

func compareObjects(
	left types.Object,
	right types.Object,
	typeString func(types.Type) string,
) int {
	switch {
	case left == right:
		return 0
	case left == nil && right != nil:
		return -1
	case left != nil && right == nil:
		return 1
	}
	leftPackage := ""
	if left.Pkg() != nil {
		leftPackage = left.Pkg().Path()
	}
	rightPackage := ""
	if right.Pkg() != nil {
		rightPackage = right.Pkg().Path()
	}
	if order := compareStrings(leftPackage, rightPackage); order != 0 {
		return order
	}
	if order := compareInts(
		objectKindOrder(left),
		objectKindOrder(right),
	); order != 0 {
		return order
	}
	leftReceiver := receiverType(left)
	rightReceiver := receiverType(right)
	if leftReceiver != nil || rightReceiver != nil {
		if order := compareStrings(
			typeStringOrEmpty(leftReceiver, typeString),
			typeStringOrEmpty(rightReceiver, typeString),
		); order != 0 {
			return order
		}
	}
	if order := compareStrings(left.Name(), right.Name()); order != 0 {
		return order
	}
	if left.Type() != nil || right.Type() != nil {
		if order := compareStrings(
			typeStringOrEmpty(left.Type(), typeString),
			typeStringOrEmpty(right.Type(), typeString),
		); order != 0 {
			return order
		}
	}
	switch {
	case left.Pos() < right.Pos():
		return -1
	case left.Pos() > right.Pos():
		return 1
	default:
		return 0
	}
}

func typeStringOrEmpty(
	source types.Type,
	typeString func(types.Type) string,
) string {
	if source == nil {
		return ""
	}
	return typeString(source)
}

func receiverType(object types.Object) types.Type {
	function, ok := object.(*types.Func)
	if !ok {
		return nil
	}
	signature, _ := function.Type().(*types.Signature)
	if signature == nil || signature.Recv() == nil {
		return nil
	}
	return signature.Recv().Type()
}

func compareStrings(left string, right string) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

func compareInts(left int, right int) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

func objectKindOrder(object types.Object) int {
	switch object.(type) {
	case *types.PkgName:
		return 1
	case *types.Const:
		return 2
	case *types.TypeName:
		return 3
	case *types.Var:
		return 4
	case *types.Func:
		return 5
	case *types.Label:
		return 6
	case *types.Builtin:
		return 7
	case *types.Nil:
		return 8
	default:
		return 9
	}
}

func CompareArtifactOwners(
	left api.ArtifactOwner,
	right api.ArtifactOwner,
) int {
	if left == right {
		return 0
	}
	leftSource, leftIsSource := left.Source()
	rightSource, rightIsSource := right.Source()
	switch {
	case leftIsSource && rightIsSource:
		return CompareObjects(leftSource, rightSource)
	case leftIsSource:
		return -1
	case rightIsSource:
		return 1
	}
	leftPackage, leftInitializer, leftIsInitializer :=
		left.PackageInitializer()
	rightPackage, rightInitializer, rightIsInitializer :=
		right.PackageInitializer()
	switch {
	case leftIsInitializer && rightIsInitializer:
		switch {
		case leftPackage.Path() < rightPackage.Path():
			return -1
		case leftPackage.Path() > rightPackage.Path():
			return 1
		case leftInitializer.Rhs.Pos() < rightInitializer.Rhs.Pos():
			return -1
		case leftInitializer.Rhs.Pos() > rightInitializer.Rhs.Pos():
			return 1
		default:
			return 0
		}
	case leftIsInitializer:
		return -1
	case rightIsInitializer:
		return 1
	}
	leftAssembly, leftIsAssembly := left.PackageAssembly()
	rightAssembly, rightIsAssembly := right.PackageAssembly()
	switch {
	case leftIsAssembly && rightIsAssembly:
		switch {
		case leftAssembly.Path() < rightAssembly.Path():
			return -1
		case leftAssembly.Path() > rightAssembly.Path():
			return 1
		default:
			return 0
		}
	case leftIsAssembly:
		return -1
	case rightIsAssembly:
		return 1
	}
	leftGenerated, leftOK := left.Generated()
	rightGenerated, rightOK := right.Generated()
	if !leftOK || !rightOK {
		switch {
		case leftOK:
			return 1
		case rightOK:
			return -1
		default:
			return 0
		}
	}
	switch {
	case leftGenerated.Kind() < rightGenerated.Kind():
		return -1
	case leftGenerated.Kind() > rightGenerated.Kind():
		return 1
	case leftGenerated.ArtifactKey() < rightGenerated.ArtifactKey():
		return -1
	case leftGenerated.ArtifactKey() > rightGenerated.ArtifactKey():
		return 1
	default:
		return 0
	}
}
