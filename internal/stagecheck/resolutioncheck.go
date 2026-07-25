package stagecheck

import (
	"crypto/sha256"
	"fmt"
	"sort"

	"github.com/tsoniclang/gotots/internal/source"
)

func verifyResolutionFingerprint(
	workspace *source.Workspace,
	request source.Request,
) error {
	packages := workspace.Packages()
	if !sort.SliceIsSorted(packages, func(i, j int) bool {
		return packages[i].ID().Compare(packages[j].ID()) < 0
	}) {
		return fmt.Errorf("workspace packages are not canonical")
	}
	toolchain := workspace.Toolchain()
	hash := sha256.New()
	fmt.Fprintln(hash, "gotots-source-universe/v2")
	fmt.Fprintf(
		hash,
		"toolchain|%s|%s|%s|%s\n",
		toolchain.BinaryDigest(),
		toolchain.Version(),
		toolchain.BuildConfigurationDigest(),
		toolchain.Experiments(),
	)
	for _, flag := range request.BuildFlags {
		fmt.Fprintf(hash, "build-flag|%d:%s\n", len(flag), flag)
	}
	for _, pkg := range packages {
		fmt.Fprintf(
			hash,
			"package|%s|%d:%s|%d|%d|%d|%t|%s|%t\n",
			pkg.ID(),
			len(pkg.DeclaredName()),
			pkg.DeclaredName(),
			uint8(pkg.Provenance()),
			uint8(pkg.Acquisition()),
			uint8(pkg.Disposition()),
			pkg.RequestedRoot(),
			pkg.ModuleGoVersion(),
			pkg.HasCheckedView(),
		)
		initialization := pkg.Initialization()
		if ordinal, initializes := initialization.GoOrdinal(); initializes {
			fmt.Fprintf(
				hash, "initialization|%d|%d\n",
				initialization.Kind(), ordinal,
			)
		} else {
			fmt.Fprintf(
				hash, "initialization|%d\n", initialization.Kind(),
			)
		}
		for _, imported := range pkg.Imports() {
			fmt.Fprintf(
				hash, "import|%s|%s\n",
				imported.Importer(), imported.Imported(),
			)
		}
		for _, file := range pkg.Files() {
			fmt.Fprintf(
				hash,
				"go-file|%s|%s|%s|%t|%t\n",
				file.ID(),
				file.ByteDigest(),
				file.EffectiveGoVersion(),
				file.Overlaid(),
				file.CgoOriginal(),
			)
		}
		for _, input := range pkg.Inputs() {
			fmt.Fprintf(
				hash,
				"input|%s|%d|%s|%t\n",
				input.ID(),
				uint8(input.Kind()),
				input.ByteDigest(),
				input.Overlaid(),
			)
		}
		for _, pattern := range pkg.EmbedPatterns() {
			fmt.Fprintf(
				hash,
				"embed-pattern|%s|%d:%s\n",
				pkg.ID(), len(pattern), pattern,
			)
		}
	}
	derived := fmt.Sprintf("%x", hash.Sum(nil))
	if workspace.ResolutionFingerprint() != derived {
		return fmt.Errorf(
			"resolution fingerprint mismatch: recorded %s, independent %s",
			workspace.ResolutionFingerprint(), derived,
		)
	}
	return nil
}
