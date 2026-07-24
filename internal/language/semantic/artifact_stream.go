package semantic

import "io"

func writeSemanticShard(
	output io.Writer,
	pkg Package,
) (semanticShardMeasurement, error) {
	return writeBinarySemanticShard(output, pkg)
}
