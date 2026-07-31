package certify

import (
	"crypto/sha256"
	"encoding/hex"
	"os"

	runtimecontract "github.com/tsoniclang/gotots/internal/contracts/runtime"
)

func readRuntimeContract(
	sourcePath string,
) (runtimecontract.Requirements, string, error) {
	payload, err := os.ReadFile(sourcePath)
	if err != nil {
		return runtimecontract.Requirements{}, "", certifyError(
			"read runtime contract",
			sourcePath,
			err.Error(),
		)
	}
	requirements, err := runtimecontract.Decode(payload)
	if err != nil {
		return runtimecontract.Requirements{}, "", certifyError(
			"read runtime contract",
			sourcePath,
			err.Error(),
		)
	}
	digest := sha256.Sum256(payload)
	return requirements, hex.EncodeToString(digest[:]), nil
}
