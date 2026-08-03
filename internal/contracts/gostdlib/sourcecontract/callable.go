package sourcecontract

import "go/types"

func DirectCallableParameterSignature(
	source types.Type,
) (*types.Signature, bool) {
	signature, ok := types.Unalias(source).(*types.Signature)
	return signature, ok
}
