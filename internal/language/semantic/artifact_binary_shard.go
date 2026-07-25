package semantic

import (
	"bytes"
	"fmt"
	"io"

	"github.com/tsoniclang/gotots/internal/identity"
)

var semanticShardMagic = [8]byte{
	'G', 'T', 'S', 'S', 'H', 'R', 'D', '6',
}

type binarySemanticShard struct {
	decoder    *binaryShardDecoder
	identities admittedPackageIdentityTable
	pkgRef     packageRef
	pkg        identity.PackageID
	provenance PackageProvenance
}

func writeBinarySemanticShard(
	output io.Writer,
	pkg Package,
) (semanticShardMeasurement, error) {
	if output == nil || pkg.ID().IsZero() || !pkg.Provenance().Valid() {
		return semanticShardMeasurement{}, fmt.Errorf(
			"semantic binary shard writer requires package and output",
		)
	}
	measurement := newSemanticShardMeasurement(pkg.ID())
	encoder := newBinaryShardEncoder(output)
	encoder.raw(semanticShardMagic[:])
	encoder.unsigned(ProviderArtifactVersion)
	encoder.unsigned(uint64(pkg.Provenance()))
	writeBinaryIdentityTables(encoder, pkg.identities)
	packageReference := pkg.identities.packageReference(pkg.ID())
	if packageReference == 0 {
		return semanticShardMeasurement{}, fmt.Errorf(
			"semantic package identity is absent from normalized dictionary",
		)
	}
	encoder.unsigned(uint64(packageReference))
	writeBinaryDefinitions(encoder, pkg, &measurement)
	writeBinaryResolutions(encoder, pkg.resolutions)
	writeBinaryDeclarations(encoder, pkg.declarations)
	writeBinaryBindings(encoder, pkg.bindings)
	writeBinaryTypes(encoder, pkg, &measurement)
	writeBinaryOperations(encoder, pkg, &measurement)
	writeBinaryUnsupported(encoder, pkg.unsupported)
	encodedBytes, err := encoder.finish()
	if err != nil {
		return semanticShardMeasurement{}, err
	}
	measurement.encodedBytes = encodedBytes
	return measurement, nil
}

func decodeBinarySemanticShard(
	input io.Reader,
	authority Authority,
	entry packageShardManifest,
) (Package, error) {
	if input == nil || !authority.Valid() {
		return Package{}, fmt.Errorf(
			"semantic binary shard decoder requires input and authority",
		)
	}
	shard, err := beginBinarySemanticShard(input, entry)
	if err != nil {
		return Package{}, err
	}
	const selectedAuthority authorityRef = 1
	stores := normalizedPackageStores{
		identities:  shard.identities,
		authorities: packageAuthorityTable{records: []Authority{authority}},
	}
	stores.definitions, err = readBinaryDefinitions(
		shard.decoder,
		entry.DefinitionCount,
		selectedAuthority,
	)
	if err != nil {
		return Package{}, err
	}
	stores.resolutions, err = readBinaryResolutions(
		shard.decoder, entry.ResolutionCount,
	)
	if err != nil {
		return Package{}, err
	}
	stores.declarations, err = readBinaryDeclarations(
		shard.decoder,
		entry.DeclarationCount,
		selectedAuthority,
	)
	if err != nil {
		return Package{}, err
	}
	stores.bindings, err = readBinaryBindings(
		shard.decoder,
		entry.BindingCount,
		selectedAuthority,
	)
	if err != nil {
		return Package{}, err
	}
	stores.types, err = readBinaryTypes(
		shard.decoder, entry.TypeCount,
	)
	if err != nil {
		return Package{}, err
	}
	stores.witnesses = binaryTypeWitnesses(
		stores.types,
		shard.pkgRef,
		selectedAuthority,
	)
	stores.operations, err = readBinaryOperations(
		shard.decoder, entry.OperationCount,
	)
	if err != nil {
		return Package{}, err
	}
	stores.unsupported, err = readBinaryUnsupported(
		shard.decoder,
		entry.UnsupportedCount,
		selectedAuthority,
	)
	if err != nil {
		return Package{}, err
	}
	if err := shard.decoder.identityUses.complete(); err != nil {
		return Package{}, err
	}
	if err := shard.decoder.finish(); err != nil {
		return Package{}, err
	}
	return newPackageFromStores(
		shard.pkg,
		shard.provenance,
		stores,
	)
}

func beginBinarySemanticShard(
	input io.Reader,
	entry packageShardManifest,
) (binarySemanticShard, error) {
	decoder, err := newBinaryShardDecoder(input, entry.ShardBytes)
	if err != nil {
		return binarySemanticShard{}, err
	}
	magic, err := decoder.raw(len(semanticShardMagic))
	if err != nil {
		return binarySemanticShard{}, err
	}
	if !bytes.Equal(magic, semanticShardMagic[:]) {
		return binarySemanticShard{}, fmt.Errorf(
			"semantic binary shard magic is invalid",
		)
	}
	version, err := decoder.unsigned("shard version")
	if err != nil {
		return binarySemanticShard{}, err
	}
	if version != ProviderArtifactVersion {
		return binarySemanticShard{}, fmt.Errorf(
			"semantic binary shard version %d, want %d",
			version,
			ProviderArtifactVersion,
		)
	}
	provenance, err := readUnsignedAs[PackageProvenance](
		decoder, "package provenance",
	)
	if err != nil {
		return binarySemanticShard{}, err
	}
	if !provenance.Valid() ||
		uint8(provenance) != entry.Provenance {
		return binarySemanticShard{}, fmt.Errorf(
			"semantic binary shard provenance is invalid",
		)
	}
	identities, err := readBinaryIdentityTables(decoder)
	if err != nil {
		return binarySemanticShard{}, err
	}
	admittedIdentities, err := admitPackageIdentityTable(identities)
	if err != nil {
		return binarySemanticShard{}, err
	}
	decoder.identityUses = newBinaryIdentityUses(
		admittedIdentities.table,
	)
	packageReference, err := readIdentityReference[packageRef](
		decoder, "package identity",
	)
	if err != nil {
		return binarySemanticShard{}, err
	}
	projection := newPackageIdentityProjection(
		admittedIdentities.table,
	)
	pkg := projection.packageID(packageReference)
	manifestPackage, err := identity.ParsePackageID(entry.Package)
	if err != nil {
		return binarySemanticShard{}, err
	}
	if pkg.IsZero() || pkg != manifestPackage {
		return binarySemanticShard{}, fmt.Errorf(
			"semantic binary shard package %s disagrees with manifest %s",
			pkg,
			manifestPackage,
		)
	}
	return binarySemanticShard{
		decoder: decoder, identities: admittedIdentities,
		pkgRef: packageReference, pkg: pkg,
		provenance: provenance,
	}, nil
}

func binaryTypeWitnesses(
	types packageTypeStore,
	pkg packageRef,
	authority authorityRef,
) packageTypeWitnessStore {
	records := make([]storedTypeWitness, len(types.records))
	for index, record := range types.records {
		records[index] = storedTypeWitness{
			pkg: pkg, typeID: record.id, authority: authority,
		}
	}
	return packageTypeWitnessStore{records: records}
}
