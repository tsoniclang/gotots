package verify

import (
	"strings"
	"testing"
)

func assertBoundedProviderConsumption(
	t *testing.T,
	budget stage2CostBudget,
	output string,
) {
	t.Helper()
	providerPackages := parseIntField(
		t,
		output,
		`providerPackages=([0-9]+)`,
	)
	shardLoads := parseIntField(
		t,
		output,
		`providerShardLoads=([0-9]+)`,
	)
	maxResident := parseIntField(
		t,
		output,
		`maxProviderPackagesResident=([0-9]+)`,
	)
	largestBytes := parseInt64Field(
		t,
		output,
		`(?m)^provider-manifest-tail rank=1 [^\n]*bytes=([0-9]+)`,
	)
	largestRecords := parseIntField(
		t,
		output,
		`(?m)^provider-manifest-tail rank=1 [^\n]*records=([0-9]+)`,
	)
	residentDefinitions := parseIntField(
		t,
		output,
		`residentDefinitions=([0-9]+)`,
	)
	assertTailLineCount(
		t,
		output,
		"provider-manifest-tail ",
		minimum(providerPackages, 20),
	)
	assertTailLineCount(
		t,
		output,
		"provider-projection-tail ",
		0,
	)
	assertTailLineCount(
		t,
		output,
		"header-tail ",
		minimum(residentDefinitions, 20),
	)
	assertSemanticMetricTails(
		t, output, "local-production",
	)
	assertSemanticConsumptionMetrics(
		t, output, "checker-consumption",
	)
	assertSemanticManifestMetrics(t, output)
	assertSemanticConsumptionMetrics(
		t, output, "provider-consumption",
	)
	assertSemanticWorkOutput(
		t, output, "local-production",
	)
	semanticPackages := parseIntField(
		t,
		output,
		`(?m)^semantic-provider-manifest: packages=([0-9]+)`,
	)
	semanticConsumed := parseIntField(
		t,
		output,
		`(?m)^semantic-provider-consumption: packages=([0-9]+)`,
	)
	semanticProjected := parseIntField(
		t,
		output,
		`(?m)^semantic-consumption-residency: packages=([0-9]+)`,
	)
	semanticLocal := parseIntField(
		t,
		output,
		`(?m)^semantic-consumption-residency: [^\n]*localPackages=([0-9]+)`,
	)
	semanticCertified := parseIntField(
		t,
		output,
		`(?m)^semantic-consumption-residency: [^\n]*certifiedPackages=([0-9]+)`,
	)
	semanticMixed := parseIntField(
		t,
		output,
		`(?m)^semantic-consumption-residency: [^\n]*mixedPackages=([0-9]+)`,
	)
	checkerLoads := parseIntField(
		t,
		output,
		`(?m)^semantic-consumption-residency: [^\n]*checkerShardLoads=([0-9]+)`,
	)
	providerLoads := parseIntField(
		t,
		output,
		`(?m)^semantic-consumption-residency: [^\n]*providerShardLoads=([0-9]+)`,
	)
	logicalLoads := parseIntField(
		t,
		output,
		`(?m)^semantic-consumption-residency: [^\n]*logicalPackageLoads=([0-9]+)`,
	)
	checkerResident := parseIntField(
		t,
		output,
		`(?m)^semantic-consumption-residency: [^\n]*maxCheckerPackagesResident=([0-9]+)`,
	)
	providerResident := parseIntField(
		t,
		output,
		`(?m)^semantic-consumption-residency: [^\n]*maxProviderPackagesResident=([0-9]+)`,
	)
	logicalResident := parseIntField(
		t,
		output,
		`(?m)^semantic-consumption-residency: [^\n]*maxLogicalPackagesResident=([0-9]+)`,
	)
	semanticLargestBytes := parseInt64Field(
		t,
		output,
		`(?m)^semantic-provider-manifest-package-tail rank=1 [^\n]*encodedBytes=([0-9]+)`,
	)
	semanticLargestRecords := parseIntField(
		t,
		output,
		`(?m)^semantic-provider-manifest-package-tail rank=1 [^\n]*records=([0-9]+)`,
	)
	if providerPackages == 0 ||
		shardLoads != 0 ||
		maxResident != 0 ||
		largestBytes == 0 ||
		largestBytes > budget.largestStructureShardBytes ||
		largestRecords == 0 ||
		largestRecords >
			budget.largestStructurePackageRecords ||
		semanticPackages != providerPackages ||
		semanticCertified != semanticPackages ||
		semanticProjected < semanticCertified ||
		semanticLocal == 0 ||
		checkerLoads != semanticLocal ||
		logicalLoads != semanticLocal ||
		semanticConsumed != semanticMixed ||
		providerLoads != semanticMixed ||
		checkerResident != 1 ||
		logicalResident != 1 ||
		(providerResident != 0 && providerResident != 1) ||
		semanticLargestBytes == 0 ||
		semanticLargestBytes >
			budget.largestSemanticShardBytes ||
		semanticLargestRecords == 0 ||
		semanticLargestRecords >
			budget.largestSemanticPackageRecords {
		t.Fatalf(
			"%s unbounded provider consumption structure=%d/%d/%d/%d/%d semantic=%d/%d/%d/%d/%d/%d/%d/%d/%d/%d/%d/%d/%d/%d",
			budget.name,
			providerPackages,
			shardLoads,
			maxResident,
			largestBytes,
			largestRecords,
			semanticPackages,
			semanticProjected,
			semanticLocal,
			semanticCertified,
			semanticMixed,
			semanticConsumed,
			checkerLoads,
			providerLoads,
			logicalLoads,
			checkerResident,
			providerResident,
			logicalResident,
			semanticLargestBytes,
			semanticLargestRecords,
		)
	}
}

func assertSemanticConsumptionMetrics(
	t *testing.T,
	output string,
	category string,
) {
	t.Helper()
	line := `(?m)^semantic-` + category + `: [^\n]*`
	packages := parseIntField(
		t, output, line+`packages=([0-9]+)`,
	)
	assertTailLineCount(
		t,
		output,
		"semantic-"+category+"-package-tail ",
		minimum(packages, 20),
	)
	for _, class := range []string{
		"definition", "operation", "type",
	} {
		assertTailLineCount(
			t,
			output,
			"semantic-"+category+"-"+class+"-tail ",
			0,
		)
	}
}

func assertSemanticManifestMetrics(
	t *testing.T,
	output string,
) {
	t.Helper()
	line := `(?m)^semantic-provider-manifest: [^\n]*`
	packages := parseIntField(
		t, output, line+`packages=([0-9]+)`,
	)
	assertTailLineCount(
		t,
		output,
		"semantic-provider-manifest-package-tail ",
		minimum(packages, 20),
	)
	for _, class := range []string{
		"definition", "operation", "type",
	} {
		assertTailLineCount(
			t,
			output,
			"semantic-provider-manifest-"+class+"-tail ",
			0,
		)
	}
}

func assertSemanticMetricTails(
	t *testing.T,
	output string,
	category string,
) {
	t.Helper()
	line := `(?m)^semantic-` + category + `: [^\n]*`
	for field, class := range map[string]string{
		"packages":    "package",
		"definitions": "definition",
		"operations":  "operation",
		"types":       "type",
	} {
		total := parseIntField(
			t,
			output,
			line+field+`=([0-9]+)`,
		)
		assertTailLineCount(
			t,
			output,
			"semantic-"+category+"-"+class+"-tail ",
			minimum(total, 20),
		)
	}
}

func assertSemanticWorkOutput(
	t *testing.T,
	output string,
	category string,
) {
	t.Helper()
	line := `(?m)^semantic-` + category + `-work: [^\n]*`
	occurrences := parseIntField(
		t, output, line+`inputOccurrences=([0-9]+)`,
	)
	contexts := parseIntField(
		t, output, line+`contexts=([0-9]+)`,
	)
	objectVisits := parseIntField(
		t, output, line+`objectVisits=([0-9]+)`,
	)
	checkerDefinitionVisits := parseIntField(
		t, output, line+`checkerDefinitionVisits=([0-9]+)`,
	)
	checkerImplicitVisits := parseIntField(
		t, output, line+`checkerImplicitVisits=([0-9]+)`,
	)
	checkerSignatureBindingVisits := parseIntField(
		t, output, line+`checkerSignatureBindingVisits=([0-9]+)`,
	)
	checkerScopeEvidenceVisits := parseIntField(
		t, output, line+`checkerScopeEvidenceVisits=([0-9]+)`,
	)
	bindingVisits := parseIntField(
		t, output, line+`implicitBindingVisits=([0-9]+)`,
	)
	captureVisits := parseIntField(
		t, output, line+`captureVisits=([0-9]+)`,
	)
	resolutionVisits := parseIntField(
		t, output, line+`resolutionVisits=([0-9]+)`,
	)
	resolutions := parseIntField(
		t, output, line+`resolutions=([0-9]+)`,
	)
	containmentVisits := parseIntField(
		t, output, line+`containmentVisits=([0-9]+)`,
	)
	containmentEdges := parseIntField(
		t, output, line+`containmentEdges=([0-9]+)`,
	)
	containmentEntries := parseIntField(
		t, output, line+`containmentEntries=([0-9]+)`,
	)
	linear := parseIntField(
		t, output, line+`linearOperations=([0-9]+)`,
	)
	if occurrences == 0 ||
		contexts != occurrences ||
		objectVisits != occurrences ||
		checkerDefinitionVisits > occurrences ||
		checkerImplicitVisits > occurrences ||
		checkerSignatureBindingVisits > occurrences ||
		checkerScopeEvidenceVisits > occurrences ||
		bindingVisits != occurrences ||
		captureVisits != occurrences ||
		resolutionVisits != occurrences ||
		resolutions != occurrences ||
		containmentVisits != 2*containmentEntries ||
		containmentEdges > containmentEntries ||
		linear > 25*occurrences+4096 {
		t.Fatalf(
			"semantic %s work is not fixed-coefficient linear: occurrences=%d contexts=%d objects=%d checkerEvidence=%d/%d/%d/%d bindings=%d captures=%d resolutionVisits=%d resolutions=%d containment=%d/%d/%d linear=%d",
			category,
			occurrences,
			contexts,
			objectVisits,
			checkerDefinitionVisits,
			checkerImplicitVisits,
			checkerSignatureBindingVisits,
			checkerScopeEvidenceVisits,
			bindingVisits,
			captureVisits,
			resolutionVisits,
			resolutions,
			containmentVisits,
			containmentEdges,
			containmentEntries,
			linear,
		)
	}
}

func assertTailLineCount(
	t *testing.T,
	output string,
	prefix string,
	want int,
) {
	t.Helper()
	count := 0
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, prefix) {
			count++
		}
	}
	if count != want {
		t.Fatalf(
			"%s lines=%d, want %d:\n%s",
			strings.TrimSpace(prefix),
			count,
			want,
			strings.TrimSpace(output),
		)
	}
}

func minimum(left int, right int) int {
	if left < right {
		return left
	}
	return right
}
