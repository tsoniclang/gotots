package callableimplementation

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"go/ast"
	"go/format"
	"go/token"
)

type SourceBodyError struct {
	Reason string
}

func (e *SourceBodyError) Error() string {
	return "digest callable source body: " + e.Reason
}

func SourceBodyDigest(fileSet *token.FileSet, body *ast.BlockStmt) (string, error) {
	if fileSet == nil || body == nil || !body.Pos().IsValid() || !body.End().IsValid() {
		return "", &SourceBodyError{Reason: "source body is invalid"}
	}
	var canonical bytes.Buffer
	if err := format.Node(&canonical, fileSet, body); err != nil {
		return "", &SourceBodyError{Reason: err.Error()}
	}
	digest := sha256.Sum256(canonical.Bytes())
	return hex.EncodeToString(digest[:]), nil
}
