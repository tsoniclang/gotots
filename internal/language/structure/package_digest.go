package structure

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"hash"

	"github.com/tsoniclang/gotots/internal/identity"
)

// PackageDigest is the canonical content digest of one logical package graph.
// It is independent of acquisition paths and provider-container layout.
func PackageDigest(pkg PackageGraph) string {
	encoder := newPackageDigestEncoder(sha256.New())
	encoder.text("gotots-structure-package/v2")
	encoder.packageID(pkg.ID())
	files := pkg.Files()
	encoder.unsigned(uint64(len(files)))
	for _, file := range files {
		writeFileDigest(encoder, file)
	}
	owners := pkg.SyntheticOwners()
	encoder.unsigned(uint64(len(owners)))
	for _, owner := range owners {
		writeOwnerDigest(encoder, owner)
	}
	writeDefinitionDigest(
		encoder,
		pkg.ownedDefinitions,
		pkg.ownedSites,
		pkg.ownedHeaders,
		pkg.ownedBoundaries,
	)
	return hex.EncodeToString(encoder.digest.Sum(nil))
}

func writeFileDigest(encoder *packageDigestEncoder, file FileGraph) {
	writeOwnerDigest(encoder, file.Owner())
	encoder.unsigned(uint64(file.OccurrenceCount()))
	err := file.VisitOccurrenceRefs(func(occurrence OccurrenceRef) error {
		encoder.text("occurrence")
		encoder.occurrenceID(occurrence.ID())
		encoder.unsigned(uint64(occurrence.Kind()))
		encoder.occurrenceID(occurrence.Parent())
		encoder.unsigned(uint64(occurrence.Edge()))
		encoder.signed(int64(occurrence.Ordinal()))
		writeSpanDigest(encoder, occurrence.Span())
		writeDisplayDigest(encoder, occurrence.Display())
		encoder.unsigned(uint64(occurrence.Token()))
		return nil
	})
	if err != nil {
		panic(err)
	}
	containment := file.Containment()
	encoder.text("containment")
	encoder.ownerRegionID(containment.Owner())
	anchors := containment.Anchors()
	encoder.unsigned(uint64(len(anchors)))
	for _, anchor := range anchors {
		encoder.occurrenceID(anchor)
	}
	writeDefinitionDigest(
		encoder,
		file.Definitions(),
		file.Sites(),
		file.Headers(),
		file.Boundaries(),
	)
	mappings := file.CheckedMappings()
	encoder.unsigned(uint64(len(mappings)))
	for _, mapping := range mappings {
		encoder.text("checked-mapping")
		encoder.definitionID(mapping.Definition())
		encoder.signed(int64(mapping.OriginLine()))
		encoder.signed(int64(mapping.OriginColumn()))
		encoder.signed(int64(mapping.OriginMatch()))
		encoder.text(mapping.CheckedDigest())
	}
}

func writeOwnerDigest(
	encoder *packageDigestEncoder,
	owner OwnerRegion,
) {
	encoder.text("owner")
	encoder.ownerRegionID(owner.ID())
	members := owner.Members()
	encoder.unsigned(uint64(len(members)))
	for _, member := range members {
		encoder.occurrenceID(member)
	}
	directives := owner.Directives()
	encoder.unsigned(uint64(len(directives)))
	for _, directive := range directives {
		encoder.text("directive")
		encoder.unsigned(uint64(directive.Kind()))
		encoder.text(directive.Tool())
		encoder.text(directive.Name())
		encoder.text(directive.Args())
		writeSpanDigest(encoder, directive.Span())
		writeDisplayDigest(encoder, directive.Display())
	}
}

func writeDefinitionDigest(
	encoder *packageDigestEncoder,
	definitions []ImplementationDefinition,
	sites []DefinitionSite,
	headers []HeaderRegion,
	boundaries []ExecutionBoundary,
) {
	encoder.unsigned(uint64(len(definitions)))
	for _, definition := range definitions {
		encoder.text("definition")
		encoder.definitionID(definition.ID())
		encoder.ownerRegionID(definition.Owner())
		encoder.definitionID(definition.Header().Definition())
		encoder.definitionID(definition.Boundary().Definition())
		encoder.text(definition.Name())
	}
	encoder.unsigned(uint64(len(sites)))
	for _, site := range sites {
		encoder.text("site")
		encoder.unsigned(uint64(site.Kind()))
		encoder.definitionID(site.Definition())
		encoder.ownerRegionID(site.Owner())
		encoder.definitionID(site.ParentDefinition())
		encoder.occurrenceID(site.Terminal())
	}
	encoder.unsigned(uint64(len(headers)))
	for _, header := range headers {
		encoder.text("header")
		encoder.definitionID(header.ID().Definition())
		encoder.text(header.Digest())
		members := header.Members()
		encoder.unsigned(uint64(len(members)))
		for _, member := range members {
			encoder.occurrenceID(member)
		}
	}
	encoder.unsigned(uint64(len(boundaries)))
	for _, boundary := range boundaries {
		encoder.text("boundary")
		encoder.definitionID(boundary.ID().Definition())
		encoder.unsigned(uint64(boundary.Kind()))
		encoder.text(boundary.CombinedDigest())
		encoder.unsigned(uint64(boundary.ImplicitOp()))
		encoder.unsigned(uint64(boundary.SyntheticRole()))
		entries := boundary.Entries()
		encoder.unsigned(uint64(len(entries)))
		for _, entry := range entries {
			encoder.occurrenceID(entry.ID())
			encoder.text(entry.Hash())
		}
	}
}

func writeSpanDigest(encoder *packageDigestEncoder, span Span) {
	encoder.signed(int64(span.Start.Line))
	encoder.signed(int64(span.Start.Column))
	encoder.signed(int64(span.Start.Offset))
	encoder.signed(int64(span.End.Line))
	encoder.signed(int64(span.End.Column))
	encoder.signed(int64(span.End.Offset))
}

func writeDisplayDigest(
	encoder *packageDigestEncoder,
	span DisplaySpan,
) {
	encoder.text(span.Start.Filename)
	encoder.signed(int64(span.Start.Line))
	encoder.signed(int64(span.Start.Column))
	encoder.text(span.End.Filename)
	encoder.signed(int64(span.End.Line))
	encoder.signed(int64(span.End.Column))
}

type packageDigestEncoder struct {
	digest  hash.Hash
	number  [binary.MaxVarintLen64]byte
	content [256]byte
}

func newPackageDigestEncoder(
	digest hash.Hash,
) *packageDigestEncoder {
	return &packageDigestEncoder{digest: digest}
}

func (encoder *packageDigestEncoder) unsigned(value uint64) {
	count := binary.PutUvarint(encoder.number[:], value)
	_, _ = encoder.digest.Write(encoder.number[:count])
}

func (encoder *packageDigestEncoder) signed(value int64) {
	count := binary.PutVarint(encoder.number[:], value)
	_, _ = encoder.digest.Write(encoder.number[:count])
}

func (encoder *packageDigestEncoder) text(value string) {
	encoder.unsigned(uint64(len(value)))
	for len(value) != 0 {
		count := min(len(value), len(encoder.content))
		copy(encoder.content[:], value[:count])
		_, _ = encoder.digest.Write(encoder.content[:count])
		value = value[count:]
	}
}

func (encoder *packageDigestEncoder) moduleID(id identity.ModuleID) {
	encoder.text(id.Path())
	encoder.text(id.Version())
}

func (encoder *packageDigestEncoder) owner(id identity.Owner) {
	encoder.unsigned(uint64(id.Class()))
	if id.Class() == identity.OwnerModule {
		encoder.moduleID(id.Module())
	}
}

func (encoder *packageDigestEncoder) packageID(id identity.PackageID) {
	encoder.owner(id.Owner())
	encoder.text(id.ImportPath())
}

func (encoder *packageDigestEncoder) fileID(id identity.FileID) {
	encoder.owner(id.Owner())
	encoder.text(id.Rel())
}

func (encoder *packageDigestEncoder) spanID(id identity.SpanID) {
	encoder.fileID(id.File())
	encoder.signed(int64(id.Start()))
	encoder.signed(int64(id.End()))
}

func (encoder *packageDigestEncoder) occurrenceID(
	id identity.OccurrenceID,
) {
	encoder.spanID(id.Span())
	encoder.unsigned(uint64(id.KindID()))
}

func (encoder *packageDigestEncoder) definitionID(
	id identity.DefinitionID,
) {
	encoder.unsigned(uint64(id.Kind()))
	encoder.occurrenceID(id.Root())
	encoder.packageID(id.Package())
	encoder.unsigned(uint64(id.ImplicitOp()))
	encoder.unsigned(uint64(id.SyntheticRole()))
	encoder.text(id.SyntheticName())
}

func (encoder *packageDigestEncoder) ownerRegionID(
	id OwnerRegionID,
) {
	encoder.unsigned(uint64(id.Kind()))
	encoder.fileID(id.File())
	encoder.packageID(id.Package())
	encoder.unsigned(uint64(id.SyntheticKind()))
}
