package ordering

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func CompareObjects(left types.Object, right types.Object) int {
	leftPackage := ""
	if left != nil && left.Pkg() != nil {
		leftPackage = left.Pkg().Path()
	}
	rightPackage := ""
	if right != nil && right.Pkg() != nil {
		rightPackage = right.Pkg().Path()
	}
	switch {
	case leftPackage < rightPackage:
		return -1
	case leftPackage > rightPackage:
		return 1
	case left.Pos() < right.Pos():
		return -1
	case left.Pos() > right.Pos():
		return 1
	case left.Name() < right.Name():
		return -1
	case left.Name() > right.Name():
		return 1
	default:
		return 0
	}
}

func CompareArtifactOwners(
	left api.ArtifactOwner,
	right api.ArtifactOwner,
) int {
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
