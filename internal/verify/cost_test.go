package verify

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/language/semantic"
)

const stage2CertificationEnvironment = "GOTOTS_CERTIFY_STAGE2"
const stage2ProjectionHelperEnvironment = "GOTOTS_STAGE2_PROJECT_LARGEST"

type stage2CostBudget struct {
	name                           string
	directory                      string
	auditWallSeconds               float64
	auditPeakRSSKiB                int
	inspectWallSeconds             float64
	inspectPeakRSSKiB              int
	structureArtifactBytes         int64
	semanticArtifactBytes          int64
	largestStructureShardBytes     int64
	largestSemanticShardBytes      int64
	largestStructurePackageRecords int
	largestSemanticPackageRecords  int
}

type processMeasurement struct {
	wallSeconds float64
	peakRSSKiB  int
	output      string
}

func TestIsolatedStage2CostGate(t *testing.T) {
	if os.Getenv(stage2CertificationEnvironment) != "1" {
		t.Skip(
			"set GOTOTS_CERTIFY_STAGE2=1 for the fail-closed phase-exit cost gate",
		)
	}
	setsidBinary := requiredCostTool(t, "setsid")
	timeBinary := requiredCostTool(t, "/usr/bin/time")
	timeoutBinary := requiredCostTool(t, "timeout")
	prlimitBinary := requiredCostTool(t, "prlimit")
	root := repoRoot(t)
	binary := filepath.Join(t.TempDir(), "gotots-stage2-gate")
	runBoundedCostCommand(
		t,
		"build",
		root,
		setsidBinary,
		prlimitBinary,
		timeoutBinary,
		120,
		"go",
		"build",
		"-o",
		binary,
		"./cmd/gotots",
	)

	budgets := []stage2CostBudget{
		{
			name: "webshop",
			directory: filepath.Join(
				root,
				"testdata",
				"projects",
				"webshop",
			),
			auditWallSeconds:               25,
			auditPeakRSSKiB:                700 * 1024,
			inspectWallSeconds:             5,
			inspectPeakRSSKiB:              350 * 1024,
			structureArtifactBytes:         12 * 1024 * 1024,
			semanticArtifactBytes:          64 * 1024 * 1024,
			largestStructureShardBytes:     4 * 1024 * 1024,
			largestSemanticShardBytes:      32 * 1024 * 1024,
			largestStructurePackageRecords: 180000,
			largestSemanticPackageRecords:  180000,
		},
		{
			name: "textindex",
			directory: filepath.Join(
				root,
				"testdata",
				"projects",
				"textindex",
			),
			auditWallSeconds:               25,
			auditPeakRSSKiB:                700 * 1024,
			inspectWallSeconds:             5,
			inspectPeakRSSKiB:              350 * 1024,
			structureArtifactBytes:         12 * 1024 * 1024,
			semanticArtifactBytes:          64 * 1024 * 1024,
			largestStructureShardBytes:     4 * 1024 * 1024,
			largestSemanticShardBytes:      32 * 1024 * 1024,
			largestStructurePackageRecords: 180000,
			largestSemanticPackageRecords:  180000,
		},
		{
			name:                           "self",
			directory:                      root,
			auditWallSeconds:               40,
			auditPeakRSSKiB:                700 * 1024,
			inspectWallSeconds:             20,
			inspectPeakRSSKiB:              900 * 1024,
			structureArtifactBytes:         20 * 1024 * 1024,
			semanticArtifactBytes:          128 * 1024 * 1024,
			largestStructureShardBytes:     4 * 1024 * 1024,
			largestSemanticShardBytes:      64 * 1024 * 1024,
			largestStructurePackageRecords: 180000,
			largestSemanticPackageRecords:  300000,
		},
	}
	for _, budget := range budgets {
		t.Run(budget.name, func(t *testing.T) {
			runStage2CostCorpus(
				t,
				binary,
				setsidBinary,
				timeBinary,
				timeoutBinary,
				prlimitBinary,
				budget,
			)
		})
	}
}

func runStage2CostCorpus(
	t *testing.T,
	binary string,
	setsidBinary string,
	timeBinary string,
	timeoutBinary string,
	prlimitBinary string,
	budget stage2CostBudget,
) {
	t.Helper()
	var certified stage2ArtifactMeasurement
	for sample := 1; sample <= 3; sample++ {
		artifactDirectory := t.TempDir()
		structureArtifact := filepath.Join(
			artifactDirectory,
			fmt.Sprintf("%s-%d.structure.gotots", budget.name, sample),
		)
		semanticArtifact := filepath.Join(
			artifactDirectory,
			fmt.Sprintf("%s-%d.semantic.gotots", budget.name, sample),
		)
		measurement := measureBoundedCostCommand(
			t,
			fmt.Sprintf("%s-audit-%d", budget.name, sample),
			rootForCostDirectory(t, budget.directory),
			setsidBinary,
			timeBinary,
			prlimitBinary,
			timeoutBinary,
			180,
			binary,
			"audit",
			"catalog",
			"-contract",
			"portable@v1",
			"-dir",
			budget.directory,
			"-structure",
			structureArtifact,
			"-semantics",
			semanticArtifact,
		)
		assertMeasurement(
			t,
			budget.name+" audit",
			sample,
			measurement,
			budget.auditWallSeconds,
			budget.auditPeakRSSKiB,
		)
		current := parseStage2ArtifactMeasurement(
			t, measurement.output,
		)
		packageContexts := parseIntField(
			t,
			measurement.output,
			`(?m)^provider structure: packageContexts=([0-9]+)`,
		)
		structureDefinitions := parseIntField(
			t,
			measurement.output,
			`(?m)^provider structure: [^\n]*definitions=([0-9]+)`,
		)
		assertTailLineCount(
			t,
			measurement.output,
			"provider-production-tail ",
			minimum(packageContexts, 20),
		)
		assertTailLineCount(
			t,
			measurement.output,
			"provider-header-tail ",
			minimum(structureDefinitions, 20),
		)
		assertSemanticMetricTails(
			t, measurement.output, "provider-production",
		)
		assertSemanticWorkOutput(
			t, measurement.output, "provider-production",
		)
		if current.structureBytes >
			budget.structureArtifactBytes ||
			current.semanticBytes >
				budget.semanticArtifactBytes ||
			current.structureShardBytes >
				budget.largestStructureShardBytes ||
			current.semanticShardBytes >
				budget.largestSemanticShardBytes ||
			current.structureRecords >
				budget.largestStructurePackageRecords ||
			current.semanticRecords >
				budget.largestSemanticPackageRecords {
			t.Fatalf(
				"%s audit sample %d exceeds artifact/shard/record budgets: %+v budget=%+v",
				budget.name,
				sample,
				current,
				budget,
			)
		}
		if sample == 1 {
			certified = current
		} else if current != certified {
			t.Fatalf(
				"%s provider production is nondeterministic: first=%+v current=%+v",
				budget.name, certified, current,
			)
		}
		if sample == 3 {
			if budget.name == "self" {
				certifyLargestSemanticProjection(
					t,
					setsidBinary,
					timeBinary,
					timeoutBinary,
					prlimitBinary,
					semanticArtifact,
					certified.semanticDigest,
				)
			}
			for inspectSample := 1; inspectSample <= 3; inspectSample++ {
				inspect := measureBoundedCostCommand(
					t,
					fmt.Sprintf(
						"%s-inspect-%d",
						budget.name,
						inspectSample,
					),
					rootForCostDirectory(t, budget.directory),
					setsidBinary,
					timeBinary,
					prlimitBinary,
					timeoutBinary,
					90,
					binary,
					"inspect",
					"constructs",
					"-contract",
					"portable@v1",
					"-dir",
					budget.directory,
					"-provider-structure",
					structureArtifact,
					"-provider-structure-digest",
					certified.structureDigest,
					"-provider-semantics",
					semanticArtifact,
					"-provider-semantics-digest",
					certified.semanticDigest,
				)
				assertMeasurement(
					t,
					budget.name+" inspect",
					inspectSample,
					inspect,
					budget.inspectWallSeconds,
					budget.inspectPeakRSSKiB,
				)
				assertBoundedProviderConsumption(
					t,
					budget,
					inspect.output,
				)
			}
		}
	}
	t.Logf(
		"%s provider artifacts: %+v", budget.name, certified,
	)
}

func certifyLargestSemanticProjection(
	t *testing.T,
	setsidBinary string,
	timeBinary string,
	timeoutBinary string,
	prlimitBinary string,
	artifactPath string,
	artifactDigest string,
) {
	t.Helper()
	t.Setenv(stage2ProjectionHelperEnvironment, "1")
	t.Setenv("GOTOTS_STAGE2_SEMANTIC_ARTIFACT", artifactPath)
	t.Setenv("GOTOTS_STAGE2_SEMANTIC_DIGEST", artifactDigest)
	expected := ""
	for sample := 1; sample <= 3; sample++ {
		measurement := measureBoundedCostCommand(
			t,
			fmt.Sprintf("largest-semantic-projection-%d", sample),
			repoRoot(t),
			setsidBinary,
			timeBinary,
			prlimitBinary,
			timeoutBinary,
			90,
			os.Args[0],
			"-test.run=^TestStage2LargestSemanticProjectionProcess$",
			"-test.count=1",
		)
		assertMeasurement(
			t,
			"largest semantic projection",
			sample,
			measurement,
			30,
			700*1024,
		)
		projection := parseTextField(
			t,
			measurement.output,
			`GOTOTS_PROJECTION ([^\n]+)`,
		)
		if sample == 1 {
			expected = projection
		} else if projection != expected {
			t.Fatalf(
				"largest semantic projection changed: first=%q current=%q",
				expected,
				projection,
			)
		}
	}
}

func TestStage2LargestSemanticProjectionProcess(t *testing.T) {
	if os.Getenv(stage2ProjectionHelperEnvironment) != "1" {
		t.Skip("projection subprocess only")
	}
	artifact, err := semantic.DecodeProviderArtifact(
		os.Getenv("GOTOTS_STAGE2_SEMANTIC_ARTIFACT"),
		os.Getenv("GOTOTS_STAGE2_SEMANTIC_DIGEST"),
	)
	if err != nil {
		t.Fatal(err)
	}
	var largest semantic.ProviderPackageContext
	for _, packageID := range artifact.PackageIDs() {
		context, present, contextErr := artifact.PackageContext(
			packageID,
		)
		if contextErr != nil {
			t.Fatal(contextErr)
		}
		if !present {
			t.Fatalf("manifest package %s has no context", packageID)
		}
		if context.ShardBytes > largest.ShardBytes {
			largest = context
		}
	}
	if largest.Package.IsZero() || largest.ShardBytes <= 0 {
		t.Fatal("semantic provider artifact has no largest package")
	}
	records := largest.DefinitionCount +
		largest.ResolutionCount +
		largest.DeclarationCount +
		largest.BindingCount +
		largest.TypeCount +
		largest.OperationCount +
		largest.UnsupportedCount
	if records <= 0 {
		t.Fatal("largest semantic package has no records")
	}
	if err := artifact.VisitPackage(
		largest.Package,
		func(pkg semantic.Package) error {
			if pkg.ID() != largest.Package {
				return fmt.Errorf(
					"projected package %s, want %s",
					pkg.ID(),
					largest.Package,
				)
			}
			return nil
		},
	); err != nil {
		t.Fatal(err)
	}
	stats := artifact.ReadStats()
	if stats.ShardLoads != 1 ||
		stats.MaxProviderPackagesResident != 1 {
		t.Fatalf("projection residency stats = %+v", stats)
	}
	fmt.Printf(
		"GOTOTS_PROJECTION package=%s shardBytes=%d records=%d\n",
		largest.Package,
		largest.ShardBytes,
		records,
	)
}

type stage2ArtifactMeasurement struct {
	structureDigest     string
	semanticDigest      string
	structureBytes      int64
	semanticBytes       int64
	structureShardBytes int64
	semanticShardBytes  int64
	structureRecords    int
	semanticRecords     int
}

func parseStage2ArtifactMeasurement(
	t *testing.T,
	output string,
) stage2ArtifactMeasurement {
	t.Helper()
	return stage2ArtifactMeasurement{
		structureDigest: parseTextField(
			t, output, `structureDigest=([0-9a-f]{64})`,
		),
		semanticDigest: parseTextField(
			t, output, `semanticDigest=([0-9a-f]{64})`,
		),
		structureBytes: parseInt64Field(
			t,
			output,
			`(?m)^provider structure: [^\n]*encodedBytes=([0-9]+)`,
		),
		semanticBytes: parseInt64Field(
			t,
			output,
			`(?m)^provider semantics: [^\n]*encodedBytes=([0-9]+)`,
		),
		structureShardBytes: parseInt64Field(
			t,
			output,
			`(?m)^provider structure: [^\n]*largestShardBytes=([0-9]+)`,
		),
		semanticShardBytes: parseInt64Field(
			t,
			output,
			`(?m)^provider semantics: [^\n]*largestShardBytes=([0-9]+)`,
		),
		structureRecords: parseIntField(
			t,
			output,
			`(?m)^provider structure: [^\n]*largestPackageRecords=([0-9]+)`,
		),
		semanticRecords: parseIntField(
			t,
			output,
			`(?m)^provider semantics: [^\n]*largestPackageRecords=([0-9]+)`,
		),
	}
}
