package load

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"golang.org/x/tools/go/packages"
)

type PackageOwner uint8

const (
	PackageOwnerInvalid PackageOwner = iota
	PackageOwnerModule
	PackageOwnerWorkspace
	PackageOwnerStandardLibrary
	PackageOwnerToolchain
	PackageOwnerLanguage
	PackageOwnerExternal
)

func (o PackageOwner) Valid() bool {
	return o >= PackageOwnerModule && o <= PackageOwnerExternal
}

func classifyPackageOwner(
	selected *packages.Package,
	toolchainKey string,
	toolchainPackages toolchainPackageMembership,
	environment bool,
) (PackageOwner, string, error) {
	if selected == nil || selected.PkgPath == "" {
		return PackageOwnerInvalid, "", fmt.Errorf("package owner evidence is incomplete")
	}
	if selected.Module != nil {
		key := moduleContractKey(selected.Module.Path, selected.Module.Version)
		if environment {
			return PackageOwnerExternal, key, nil
		}
		if selected.Module.Main {
			return PackageOwnerWorkspace, key, nil
		}
		return PackageOwnerModule, key, nil
	}
	if selected.PkgPath == "builtin" || selected.PkgPath == "unsafe" {
		if !toolchainPackages.standardContains(selected.PkgPath) {
			return PackageOwnerInvalid, "", fmt.Errorf(
				"language package %q is absent from the selected toolchain standard set",
				selected.PkgPath,
			)
		}
		return PackageOwnerLanguage, toolchainKey, nil
	}
	if toolchainPackages.commandContains(selected.PkgPath) {
		return PackageOwnerToolchain, toolchainKey, nil
	}
	if toolchainPackages.standardContains(selected.PkgPath) {
		return PackageOwnerStandardLibrary, toolchainKey, nil
	}
	if environment {
		return PackageOwnerInvalid, "", fmt.Errorf(
			"environment package %q is neither module-owned nor selected-toolchain-owned",
			selected.PkgPath,
		)
	}
	return PackageOwnerWorkspace, "workspace-package:" + selected.PkgPath, nil
}

func moduleContractKey(modulePath string, moduleVersion string) string {
	digest := sha256.Sum256([]byte(modulePath + "\x00" + moduleVersion))
	return hex.EncodeToString(digest[:])
}
