package stagecheck

import (
	"crypto/sha256"
	"fmt"
	"go/ast"

	"github.com/tsoniclang/gotots/internal/language/structure"
)

func independentHeaderDigest(
	raw []byte,
	span func(ast.Node) structure.Span,
	node ast.Node,
	members []derivedOccurrence,
) (string, error) {
	root := span(node)
	end := root.End.Offset
	switch typed := node.(type) {
	case *ast.FuncDecl:
		if typed.Body != nil {
			end = span(typed.Body).Start.Offset
		}
	case *ast.FuncLit:
		end = span(typed.Body).Start.Offset
	case *ast.ValueSpec:
		if len(typed.Values) == 0 {
			return "", fmt.Errorf(
				"independent package initializer has no values",
			)
		}
		end = span(typed.Values[0]).Start.Offset
	}
	if root.Start.Offset < 0 ||
		end < root.Start.Offset ||
		end > len(raw) {
		return "", fmt.Errorf(
			"independent header range exceeds selected bytes",
		)
	}
	hash := sha256.New()
	fmt.Fprintf(hash, "bytes:%d:", end-root.Start.Offset)
	hash.Write(raw[root.Start.Offset:end])
	for _, occurrence := range members {
		fmt.Fprintf(
			hash,
			"|%s|%d|%s|%d|%d",
			occurrence.id,
			uint16(occurrence.kind),
			occurrence.edge,
			occurrence.ordinal,
			uint16(occurrence.token),
		)
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func independentDigestSpan(
	raw []byte,
	span structure.Span,
) (string, error) {
	if span.Start.Offset < 0 ||
		span.End.Offset < span.Start.Offset ||
		span.End.Offset > len(raw) {
		return "", fmt.Errorf(
			"independent digest span exceeds source",
		)
	}
	return fmt.Sprintf(
		"%x", sha256.Sum256(raw[span.Start.Offset:span.End.Offset]),
	), nil
}

func independentDigest(parts ...string) string {
	hash := sha256.New()
	for _, part := range parts {
		fmt.Fprintf(hash, "%d:%s|", len(part), part)
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}
