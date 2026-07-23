package source

import (
	"crypto/sha256"
	"fmt"
)

// ProviderInputFingerprint binds exactly the package-local inputs that can
// change a certified structural graph or selection fact. It deliberately
// excludes application roots and acquisition locations so one certified
// provider package remains reusable across unrelated workspaces and relocated
// caches. Toolchain/build configuration is bound separately by the artifact
// context.
func (p *LoadedPackage) ProviderInputFingerprint() string {
	if p == nil {
		return ""
	}
	hash := sha256.New()
	fmt.Fprintln(hash, "gotots-provider-package/v1")
	fmt.Fprintf(
		hash,
		"package|%s|%d|%d|%s|%t\n",
		p.id,
		p.provenance,
		p.disposition,
		p.moduleGoVersion,
		p.hasCheckedView,
	)
	for _, imported := range p.imports {
		fmt.Fprintf(hash, "import|%d:%s\n", len(imported), imported)
	}
	for _, file := range p.files {
		fmt.Fprintf(
			hash,
			"go-file|%s|%s|%s|%t\n",
			file.id,
			file.byteDigest,
			file.effectiveVersion,
			file.cgoOriginal,
		)
	}
	for _, input := range p.inputs {
		fmt.Fprintf(
			hash,
			"input|%s|%d|%s|%t\n",
			input.id,
			input.kind,
			input.byteDigest,
			input.overlaid,
		)
	}
	for _, pattern := range p.embedPatterns {
		fmt.Fprintf(
			hash, "embed-pattern|%d:%s\n",
			len(pattern), pattern,
		)
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}

// ResolutionFingerprint binds the complete canonical source universe without
// acquisition paths: package closure, selected Go bytes, supplemental
// toolchain/embed bytes, build selection, versions, and cgo-view facts.
func (u *Universe) ResolutionFingerprint() string {
	if u == nil {
		return ""
	}
	hash := sha256.New()
	fmt.Fprintln(hash, "gotots-source-universe/v1")
	fmt.Fprintf(
		hash,
		"toolchain|%s|%s|%s|%s\n",
		u.toolchain.binaryDigest,
		u.toolchain.version,
		u.toolchain.buildConfigDigest,
		u.toolchain.experiments,
	)
	for _, flag := range u.request.BuildFlags {
		fmt.Fprintf(hash, "build-flag|%d:%s\n", len(flag), flag)
	}
	for _, pkg := range u.packages {
		fmt.Fprintf(
			hash,
			"package|%s|%d|%d|%d|%t|%s|%t\n",
			pkg.id,
			pkg.provenance,
			pkg.acquisition,
			pkg.disposition,
			pkg.requestedRoot,
			pkg.moduleGoVersion,
			pkg.hasCheckedView,
		)
		for _, imported := range pkg.imports {
			fmt.Fprintf(hash, "import|%s|%s\n", pkg.id, imported)
		}
		for _, file := range pkg.files {
			fmt.Fprintf(
				hash,
				"go-file|%s|%s|%s|%t|%t\n",
				file.id,
				file.byteDigest,
				file.effectiveVersion,
				file.overlaid,
				file.cgoOriginal,
			)
		}
		for _, input := range pkg.inputs {
			fmt.Fprintf(
				hash,
				"input|%s|%d|%s|%t\n",
				input.id,
				input.kind,
				input.byteDigest,
				input.overlaid,
			)
		}
		for _, pattern := range pkg.embedPatterns {
			fmt.Fprintf(
				hash, "embed-pattern|%s|%d:%s\n",
				pkg.id, len(pattern), pattern,
			)
		}
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}
