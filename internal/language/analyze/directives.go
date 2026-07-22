// Directive inventory: every comment directive in a file joins the closed
// directive catalog. Unknown go:* directives fail closed; non-go tool
// directives are inventoried under the external-tool member with their tool
// recorded.
package analyze

import (
	"go/ast"
	"strings"

	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/source"
)

// scanDirectives inventories the comment directives of one file. It scans the
// file's complete comment set (File.Comments indexes every group, including
// doc comments).
func scanDirectives(b *builder, file *source.File) ([]DirectiveRecord, error) {
	syntax, ok := file.FullSyntax()
	if !ok {
		return nil, nil
	}
	var records []DirectiveRecord
	for _, group := range syntax.Comments {
		for _, comment := range group.List {
			record, ok, err := b.directiveOf(comment)
			if err != nil {
				return nil, err
			}
			if ok {
				records = append(records, record)
			}
		}
	}
	return records, nil
}

// directiveOf classifies one comment line against the directive catalog.
func (b *builder) directiveOf(comment *ast.Comment) (DirectiveRecord, bool, error) {
	text := comment.Text
	span := b.physicalSpan(comment)
	display := b.displaySpan(comment)
	record := func(kind catalog.DirectiveKind, tool, name, args string) (DirectiveRecord, bool, error) {
		return DirectiveRecord{kind: kind, tool: tool, name: name, args: args, span: span, display: display}, true, nil
	}
	// //line and /*line position directives adjust display positions only.
	if strings.HasPrefix(text, "//line ") || strings.HasPrefix(text, "/*line ") {
		return record(catalog.DirectiveLine, "line", "line", strings.TrimSpace(text[len("//line "):]))
	}
	// Legacy build tags predate the //go:build form.
	if strings.HasPrefix(text, "// +build") || strings.HasPrefix(text, "//+build") {
		return record(catalog.DirectiveLegacyBuildTag, "+build", "+build", strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(text, "// +build"), "//+build")))
	}
	directive, ok := ast.ParseDirective(comment.Slash, text)
	if !ok {
		return DirectiveRecord{}, false, nil
	}
	if directive.Tool == "go" {
		kind := catalog.GoDirectiveByName(directive.Name)
		if !kind.Valid() {
			return DirectiveRecord{}, false, newUnknownDirectiveError(directive.Tool, directive.Name, b.file, span)
		}
		return record(kind, directive.Tool, directive.Name, directive.Args)
	}
	return record(catalog.DirectiveExternalTool, directive.Tool, directive.Name, directive.Args)
}
