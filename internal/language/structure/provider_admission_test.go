package structure

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestProviderShardAdmissionRejectsRedigestedStructuralCorruption(
	t *testing.T,
) {
	fixture := writeProviderStoreFixture(
		t,
		filepath.Join(t.TempDir(), "provider.gotots"),
	)
	decoded, err := decodeProviderStream(bytes.NewReader(fixture.shard))
	if err != nil {
		t.Fatal(err)
	}
	synthetic := decoded.packageGraphs[fixture.pkg]
	synthetic.ownedSites[0].kind = DefinitionSiteSource
	decoded.packageGraphs[fixture.pkg] = synthetic

	var shard bytes.Buffer
	if err := encodeProviderArtifact(&shard, decoded); err != nil {
		t.Fatal(err)
	}
	shardBytes := shard.Bytes()
	manifest := fixture.manifest
	manifest.Packages = append(
		[]providerManifestPackage(nil),
		manifest.Packages...,
	)
	manifest.Packages[0].ShardBytes = int64(len(shardBytes))
	manifest.Packages[0].ShardDigest = fmt.Sprintf(
		"%x",
		sha256.Sum256(shardBytes),
	)

	spool, err := os.CreateTemp(t.TempDir(), "provider-shard-*")
	if err != nil {
		t.Fatal(err)
	}
	defer spool.Close()
	if _, err := spool.Write(shardBytes); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "redigested.gotots")
	digest, err := writeProviderContainer(path, spool, manifest)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := DecodeProviderArtifact(path, digest)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := artifact.SyntheticPackageGraph(
		fixture.pkg,
	); err == nil {
		t.Fatal("redigested structural corruption was exposed")
	}
}

func TestProviderAdmissionRejectsEmptyPackageShard(t *testing.T) {
	fixture := writeProviderStoreFixture(
		t,
		filepath.Join(t.TempDir(), "provider.gotots"),
	)
	admission, err := newProviderAdmission(fixture.context)
	if err != nil {
		t.Fatal(err)
	}
	if err := admission.addPackage(artifactPackage{
		Package:     fixture.pkg.String(),
		InputDigest: testProviderDigest("empty-package"),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := admission.finish(); err == nil {
		t.Fatal("provider shard without a structural owner passed admission")
	}
}
