package structure

import (
	"crypto/sha256"
	"fmt"
	"hash"
	"strconv"
)

// PackageDigest is the canonical content digest of one logical package graph.
// It is independent of acquisition paths and provider-container layout.
func PackageDigest(pkg PackageGraph) string {
	digest := sha256.New()
	writeDigestPart(digest, "gotots-structure-package/v1")
	writeDigestPart(digest, pkg.ID().String())
	for _, file := range pkg.Files() {
		writeFileDigest(digest, file)
	}
	for _, owner := range pkg.SyntheticOwners() {
		writeOwnerDigest(digest, owner)
	}
	writeDefinitionDigest(
		digest,
		pkg.ownedDefinitions,
		pkg.ownedSites,
		pkg.ownedHeaders,
		pkg.ownedBoundaries,
	)
	return fmt.Sprintf("%x", digest.Sum(nil))
}

func writeFileDigest(digest hash.Hash, file FileGraph) {
	writeOwnerDigest(digest, file.Owner())
	for _, occurrence := range file.Occurrences() {
		writeDigestPart(digest, "occurrence")
		writeDigestPart(digest, occurrence.ID().String())
		writeDigestInt(digest, int64(occurrence.Kind()))
		writeDigestPart(digest, occurrence.Parent().String())
		writeDigestInt(digest, int64(occurrence.Edge()))
		writeDigestInt(digest, int64(occurrence.Ordinal()))
		writeSpanDigest(digest, occurrence.Span())
		writeDisplayDigest(digest, occurrence.Display())
		writeDigestInt(digest, int64(occurrence.Token()))
	}
	containment := file.Containment()
	writeDigestPart(digest, "containment")
	writeDigestPart(digest, containment.Owner().String())
	for _, anchor := range containment.Anchors() {
		writeDigestPart(digest, anchor.String())
	}
	writeDefinitionDigest(
		digest,
		file.Definitions(),
		file.Sites(),
		file.Headers(),
		file.Boundaries(),
	)
	for _, mapping := range file.CheckedMappings() {
		writeDigestPart(digest, "checked-mapping")
		writeDigestPart(digest, mapping.Definition().String())
		writeDigestInt(digest, int64(mapping.OriginLine()))
		writeDigestInt(digest, int64(mapping.OriginColumn()))
		writeDigestInt(digest, int64(mapping.OriginMatch()))
		writeDigestPart(digest, mapping.CheckedDigest())
	}
}

func writeOwnerDigest(digest hash.Hash, owner OwnerRegion) {
	writeDigestPart(digest, "owner")
	writeDigestPart(digest, owner.ID().String())
	for _, member := range owner.Members() {
		writeDigestPart(digest, member.String())
	}
	for _, directive := range owner.Directives() {
		writeDigestPart(digest, "directive")
		writeDigestInt(digest, int64(directive.Kind()))
		writeDigestPart(digest, directive.Tool())
		writeDigestPart(digest, directive.Name())
		writeDigestPart(digest, directive.Args())
		writeSpanDigest(digest, directive.Span())
		writeDisplayDigest(digest, directive.Display())
	}
}

func writeDefinitionDigest(
	digest hash.Hash,
	definitions []ImplementationDefinition,
	sites []DefinitionSite,
	headers []HeaderRegion,
	boundaries []ExecutionBoundary,
) {
	for _, definition := range definitions {
		writeDigestPart(digest, "definition")
		writeDigestPart(digest, definition.ID().String())
		writeDigestPart(digest, definition.Owner().String())
		writeDigestPart(digest, definition.Header().String())
		writeDigestPart(digest, definition.Boundary().String())
		writeDigestPart(digest, definition.Name())
	}
	for _, site := range sites {
		writeDigestPart(digest, "site")
		writeDigestInt(digest, int64(site.Kind()))
		writeDigestPart(digest, site.Definition().String())
		writeDigestPart(digest, site.Owner().String())
		writeDigestPart(digest, site.ParentDefinition().String())
		writeDigestPart(digest, site.Terminal().String())
	}
	for _, header := range headers {
		writeDigestPart(digest, "header")
		writeDigestPart(digest, header.ID().String())
		writeDigestPart(digest, header.Digest())
		for _, member := range header.Members() {
			writeDigestPart(digest, member.String())
		}
	}
	for _, boundary := range boundaries {
		writeDigestPart(digest, "boundary")
		writeDigestPart(digest, boundary.ID().String())
		writeDigestInt(digest, int64(boundary.Kind()))
		writeDigestPart(digest, boundary.CombinedDigest())
		writeDigestInt(digest, int64(boundary.ImplicitOp()))
		writeDigestInt(digest, int64(boundary.SyntheticRole()))
		for _, entry := range boundary.Entries() {
			writeDigestPart(digest, entry.ID().String())
			writeDigestPart(digest, entry.Hash())
		}
	}
}

func writeSpanDigest(digest hash.Hash, span Span) {
	writeDigestInt(digest, int64(span.Start.Line))
	writeDigestInt(digest, int64(span.Start.Column))
	writeDigestInt(digest, int64(span.Start.Offset))
	writeDigestInt(digest, int64(span.End.Line))
	writeDigestInt(digest, int64(span.End.Column))
	writeDigestInt(digest, int64(span.End.Offset))
}

func writeDisplayDigest(digest hash.Hash, span DisplaySpan) {
	writeDigestPart(digest, span.Start.Filename)
	writeDigestInt(digest, int64(span.Start.Line))
	writeDigestInt(digest, int64(span.Start.Column))
	writeDigestPart(digest, span.End.Filename)
	writeDigestInt(digest, int64(span.End.Line))
	writeDigestInt(digest, int64(span.End.Column))
}

func writeDigestInt(digest hash.Hash, value int64) {
	writeDigestPart(digest, strconv.FormatInt(value, 10))
}

func writeDigestPart(digest hash.Hash, value string) {
	fmt.Fprintf(digest, "%d:", len(value))
	digest.Write([]byte(value))
	digest.Write([]byte{'|'})
}
