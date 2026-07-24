package stagecheck

import (
	"fmt"
	"go/ast"
	"strings"

	"github.com/tsoniclang/gotots/internal/language/catalog"
)

func (b *derivedFile) deriveDirectives(syntax *ast.File) error {
	for _, group := range syntax.Comments {
		for _, comment := range group.List {
			kind, tool, name, args, present, err := independentDirective(comment)
			if err != nil {
				return fmt.Errorf("%s: %w", b.file.ID(), err)
			}
			if !present {
				continue
			}
			addRecord(&b.ledger.directives, directiveLedgerRecord{
				owner:   b.owner,
				kind:    kind,
				tool:    tool,
				name:    name,
				args:    args,
				span:    b.span(comment),
				display: b.display(comment),
			})
		}
	}
	return nil
}

func independentDirective(
	comment *ast.Comment,
) (
	kind catalog.DirectiveKind,
	tool string,
	name string,
	args string,
	present bool,
	err error,
) {
	text := comment.Text
	if strings.HasPrefix(text, "//line ") ||
		strings.HasPrefix(text, "/*line ") {
		return catalog.DirectiveLine,
			"line",
			"line",
			strings.TrimSpace(
				strings.TrimPrefix(
					strings.TrimPrefix(text, "//line "),
					"/*line ",
				),
			),
			true,
			nil
	}
	if strings.HasPrefix(text, "// +build") ||
		strings.HasPrefix(text, "//+build") {
		return catalog.DirectiveLegacyBuildTag,
			"+build",
			"+build",
			strings.TrimSpace(
				strings.TrimPrefix(
					strings.TrimPrefix(text, "// +build"),
					"//+build",
				),
			),
			true,
			nil
	}
	parsed, found := ast.ParseDirective(comment.Slash, text)
	if !found {
		return catalog.DirectiveInvalid, "", "", "", false, nil
	}
	if parsed.Tool != "go" {
		return catalog.DirectiveExternalTool,
			parsed.Tool,
			parsed.Name,
			parsed.Args,
			true,
			nil
	}
	bound := catalog.GoDirectiveByName(parsed.Name)
	if !bound.Valid() {
		return catalog.DirectiveInvalid,
			"",
			"",
			"",
			false,
			fmt.Errorf("unknown //go:%s directive", parsed.Name)
	}
	return bound,
		parsed.Tool,
		parsed.Name,
		parsed.Args,
		true,
		nil
}
