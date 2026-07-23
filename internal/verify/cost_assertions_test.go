package verify

import (
	"strings"
	"testing"
)

func assertBoundedProviderConsumption(
	t *testing.T,
	budget stage1CostBudget,
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
		`largestProjectedPackageBytes=([0-9]+)`,
	)
	largestRecords := parseIntField(
		t,
		output,
		`largestProjectedPackageRecords=([0-9]+)`,
	)
	definitions := parseIntField(
		t,
		output,
		`definitions=([0-9]+)`,
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
		minimum(providerPackages, 20),
	)
	assertTailLineCount(
		t,
		output,
		"header-tail ",
		minimum(definitions, 20),
	)
	if providerPackages == 0 ||
		shardLoads != providerPackages ||
		maxResident != 1 ||
		largestBytes == 0 ||
		largestBytes > budget.largestShardBytes ||
		largestRecords == 0 ||
		largestRecords > budget.largestPackageRecords {
		t.Fatalf(
			"%s unbounded provider consumption packages=%d loads=%d maxResident=%d largestBytes=%d largestRecords=%d",
			budget.name,
			providerPackages,
			shardLoads,
			maxResident,
			largestBytes,
			largestRecords,
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
