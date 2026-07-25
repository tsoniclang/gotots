package semantic

import "io"

func decodeSemanticShard(
	input io.Reader,
	authority Authority,
	entry packageShardManifest,
) (Package, error) {
	return decodeBinarySemanticShard(input, authority, entry)
}
