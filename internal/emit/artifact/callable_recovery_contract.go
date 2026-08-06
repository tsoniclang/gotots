package artifact

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func WithCallableRecovery(
	contract Contract,
	factory tsgo.Factory,
	recovery bool,
) (Contract, error) {
	value := "0"
	if recovery {
		value = "1"
	}
	encoded, err := tsgo.EncodeNode(
		factory.NumericLiteral(value, tsgo.TokenFlagsNone),
	)
	if err != nil {
		return Contract{}, err
	}
	return contract.withOwnedFacet(
		api.ArtifactFacetCallableRecovery,
		encoded,
	)
}
