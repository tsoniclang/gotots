package emit_test

import (
	"strings"
	"testing"
)

func assertWaveNineGenericArtifactBudget(
	t *testing.T,
	artifacts []artifactSize,
) {
	t.Helper()
	concretizations := 0
	concretizationBytes := 0
	capabilities := 0
	capabilityBytes := 0
	for _, artifact := range artifacts {
		switch {
		case strings.HasPrefix(
			artifact.path,
			"support/generics/concretizations/",
		):
			concretizations++
			concretizationBytes += artifact.bytes
		case strings.HasPrefix(
			artifact.path,
			"support/generics/capabilities/",
		):
			capabilities++
			capabilityBytes += artifact.bytes
		}
	}
	// Exact concrete operations and the deferred-callable registry are reusable
	// declarations grouped into their two operation-family modules.
	if concretizations != 6 || concretizationBytes > 6_200 ||
		capabilities != 2 || capabilityBytes > 4_000 {
		t.Fatalf(
			"Wave 9 generic artifact bounds exceeded: concretizations=%d/%d capabilities=%d/%d",
			concretizations,
			concretizationBytes,
			capabilities,
			capabilityBytes,
		)
	}
}
