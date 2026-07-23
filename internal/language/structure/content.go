package structure

import (
	"crypto/sha256"
	"fmt"
	"go/ast"

	"github.com/tsoniclang/gotots/internal/identity"
)

func (b *fileBuilder) headerDigest(
	node ast.Node,
	members []identity.OccurrenceID,
) (string, error) {
	span := b.physicalSpan(node)
	end := span.End.Offset
	switch node := node.(type) {
	case *ast.FuncDecl:
		if node.Body != nil {
			end = b.physicalSpan(node.Body).Start.Offset
		}
	case *ast.FuncLit:
		end = b.physicalSpan(node.Body).Start.Offset
	case *ast.ValueSpec:
		if len(node.Values) == 0 {
			return "", fmt.Errorf(
				"package initializer has no execution value",
			)
		}
		end = b.physicalSpan(node.Values[0]).Start.Offset
	}
	if span.Start.Offset < 0 ||
		end < span.Start.Offset ||
		end > len(b.raw) {
		return "", fmt.Errorf(
			"header range %d-%d exceeds %d selected bytes",
			span.Start.Offset, end, len(b.raw),
		)
	}
	hash := sha256.New()
	fmt.Fprintf(
		hash,
		"bytes:%d:",
		end-span.Start.Offset,
	)
	hash.Write(b.raw[span.Start.Offset:end])
	for _, id := range members {
		occurrence, present := b.occurrences[id]
		if !present {
			return "", fmt.Errorf(
				"header member %s has no canonical occurrence", id,
			)
		}
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

func digestSpan(raw []byte, span Span) (string, error) {
	if span.Start.Offset < 0 ||
		span.End.Offset < span.Start.Offset ||
		span.End.Offset > len(raw) {
		return "", fmt.Errorf(
			"span %d-%d exceeds %d selected bytes",
			span.Start.Offset, span.End.Offset, len(raw),
		)
	}
	sum := sha256.Sum256(raw[span.Start.Offset:span.End.Offset])
	return fmt.Sprintf("%x", sum), nil
}

func digestStrings(parts ...string) string {
	hash := sha256.New()
	for _, part := range parts {
		fmt.Fprintf(hash, "%d:%s|", len(part), part)
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}
