package structure

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
)

func checkedNodeDigest(
	fset *token.FileSet,
	node ast.Node,
) (string, error) {
	if fset == nil || node == nil {
		return "", fmt.Errorf("checked syntax digest requires syntax and file set")
	}
	var rendered bytes.Buffer
	config := printer.Config{Mode: printer.RawFormat, Tabwidth: 8}
	if err := config.Fprint(&rendered, fset, node); err != nil {
		return "", fmt.Errorf("checked syntax is not printable: %w", err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(rendered.Bytes())), nil
}

func syntheticContentDigests(
	fset *token.FileSet,
	node ast.Node,
	nameIndex int,
) (string, string, error) {
	full, err := checkedNodeDigest(fset, node)
	if err != nil {
		return "", "", err
	}
	headerNode := node
	if function, ok := node.(*ast.FuncDecl); ok {
		copy := *function
		copy.Body = nil
		headerNode = &copy
	}
	header, err := checkedNodeDigest(fset, headerNode)
	if err != nil {
		return "", "", err
	}
	index := fmt.Sprint(nameIndex)
	return digestStrings(header, index), digestStrings(full, index), nil
}
