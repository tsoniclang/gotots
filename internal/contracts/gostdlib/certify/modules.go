package certify

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

const moduleMapSchemaVersion = 1

type moduleMapDocument struct {
	SchemaVersion int          `json:"schemaVersion"`
	Modules       []moduleSeed `json:"modules"`
}

func readModuleSeeds(sourcePath string) ([]moduleSeed, error) {
	file, err := os.Open(sourcePath)
	if err != nil {
		return nil, certifyError("read module map", sourcePath, err.Error())
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var document moduleMapDocument
	if err := decoder.Decode(&document); err != nil {
		return nil, certifyError("read module map", sourcePath, err.Error())
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("multiple JSON values")
		}
		return nil, certifyError("read module map", sourcePath, err.Error())
	}
	if document.SchemaVersion != moduleMapSchemaVersion {
		return nil, certifyError("read module map", sourcePath, "schema is unsupported")
	}
	return validateSeeds(document.Modules)
}
