package structure

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestProviderProjectionStatsAreBoundedAndNonSemantic(t *testing.T) {
	fixture := writeProviderStoreFixture(
		t,
		filepath.Join(t.TempDir(), "provider.gotots"),
	)
	artifact, err := DecodeProviderArtifact(
		fixture.path,
		fixture.digest,
	)
	if err != nil {
		t.Fatal(err)
	}
	if stats := artifact.ProjectionStats(); !reflect.DeepEqual(
		stats,
		ProviderProjectionStats{},
	) {
		t.Fatalf("unprojected artifact has stats %+v", stats)
	}
	if _, present, err := artifact.SyntheticPackageGraph(
		fixture.pkg,
	); err != nil || !present {
		t.Fatalf("first projection present=%t err=%v", present, err)
	}
	if _, present, err := artifact.SyntheticPackageGraph(
		fixture.pkg,
	); err != nil || !present {
		t.Fatalf("cached projection present=%t err=%v", present, err)
	}
	stats := artifact.ProjectionStats()
	if stats.ShardLoads != 1 ||
		stats.CacheHits != 1 ||
		stats.MaxResidentPackages != 1 ||
		stats.ProjectedPackages != 1 ||
		stats.LargestPackageBytes != int64(len(fixture.shard)) ||
		stats.LargestPackageRecords == 0 {
		t.Fatalf("projection stats = %+v", stats)
	}
	tail := stats.LargestPackages()
	if len(tail) != 1 ||
		tail[0].Package != fixture.pkg ||
		tail[0].Bytes != int64(len(fixture.shard)) ||
		tail[0].Records != stats.LargestPackageRecords {
		t.Fatalf("projection tail = %+v", tail)
	}
	headers := stats.LargestHeaders()
	if len(headers) != 1 ||
		headers[0].Header.IsZero() ||
		headers[0].EncodedBytes == 0 {
		t.Fatalf("projection header tail = %+v", headers)
	}
	tail[0] = ProviderPackageSize{}
	headers[0] = HeaderArtifactSize{}
	if artifact.ProjectionStats().LargestPackages()[0].Package.IsZero() {
		t.Fatal("projection stats exposed backing storage")
	}
	if artifact.ProjectionStats().LargestHeaders()[0].Header.IsZero() {
		t.Fatal("projection header stats exposed backing storage")
	}
}

func TestProviderStorageHasExactlyOnePayloadSlot(t *testing.T) {
	artifactType := reflect.TypeOf((*ProviderArtifact)(nil))
	storageType := reflect.TypeOf(providerStorage{})
	payloadSlots := 0
	for index := 0; index < storageType.NumField(); index++ {
		field := storageType.Field(index)
		if field.Type == artifactType {
			payloadSlots++
			continue
		}
		switch field.Type.Kind() {
		case reflect.Array, reflect.Map, reflect.Slice:
			if containsProviderArtifact(field.Type) {
				t.Fatalf(
					"provider storage field %s can retain multiple payloads",
					field.Name,
				)
			}
		}
	}
	if payloadSlots != 1 {
		t.Fatalf(
			"provider storage payload slots=%d, want exactly one",
			payloadSlots,
		)
	}
}

func TestProviderManifestContainsNoDetailedGraphRecords(t *testing.T) {
	fixture := writeProviderStoreFixture(
		t,
		filepath.Join(t.TempDir(), "provider.gotots"),
	)
	raw, err := json.Marshal(fixture.manifest)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{
		`"occurrences"`,
		`"owners"`,
		`"sites"`,
		`"headers"`,
		`"boundaries"`,
		`"checkedMappings"`,
	} {
		if strings.Contains(string(raw), key) {
			t.Fatalf(
				"provider manifest duplicates detailed graph field %s",
				key,
			)
		}
	}
}

func containsProviderArtifact(typ reflect.Type) bool {
	artifactType := reflect.TypeOf(ProviderArtifact{})
	for typ.Kind() == reflect.Array ||
		typ.Kind() == reflect.Map ||
		typ.Kind() == reflect.Pointer ||
		typ.Kind() == reflect.Slice {
		if typ.Kind() == reflect.Map {
			if typ.Key() == artifactType ||
				typ.Key() == reflect.PointerTo(artifactType) {
				return true
			}
			typ = typ.Elem()
			continue
		}
		typ = typ.Elem()
	}
	return typ == artifactType
}
