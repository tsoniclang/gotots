package structure

import (
	"fmt"
	"go/ast"
	"strings"

	"github.com/tsoniclang/gotots/internal/language/catalog"
)

// NewDirective validates one closed directive payload.
func NewDirective(
	kind catalog.DirectiveKind,
	tool string,
	name string,
	args string,
	span Span,
	display DisplaySpan,
) (Directive, error) {
	if !kind.Valid() ||
		span.Start.Line <= 0 ||
		span.Start.Column <= 0 ||
		span.Start.Offset < 0 ||
		span.End.Line <= 0 ||
		span.End.Column <= 0 ||
		span.End.Offset < span.Start.Offset ||
		display.Start.Filename == "" ||
		display.Start.Line <= 0 ||
		display.Start.Column <= 0 ||
		display.End.Filename == "" ||
		display.End.Line <= 0 ||
		display.End.Column <= 0 {
		return Directive{}, fmt.Errorf("directive has invalid canonical payload")
	}
	switch kind {
	case catalog.DirectiveLine:
		if tool != "line" || name != "line" {
			return Directive{}, fmt.Errorf("line directive has invalid identity")
		}
	case catalog.DirectiveLegacyBuildTag:
		if tool != "+build" || name != "+build" {
			return Directive{}, fmt.Errorf(
				"legacy build directive has invalid identity",
			)
		}
	case catalog.DirectiveExternalTool:
		if tool == "" || tool == "go" || name == "" {
			return Directive{}, fmt.Errorf(
				"external directive has invalid tool or name",
			)
		}
	default:
		if tool != "go" || name != kind.Name() {
			return Directive{}, fmt.Errorf(
				"go directive does not match its catalog identity",
			)
		}
	}
	return Directive{
		kind:    kind,
		tool:    tool,
		name:    name,
		args:    args,
		span:    span,
		display: display,
	}, nil
}

func (b *fileBuilder) scanDirectives(syntax *ast.File) ([]Directive, error) {
	var out []Directive
	for _, group := range syntax.Comments {
		for _, comment := range group.List {
			record, present, err := b.directive(comment)
			if err != nil {
				return nil, err
			}
			if present {
				out = append(out, record)
				b.work.RecordAppends++
			}
		}
	}
	return out, nil
}

func (b *fileBuilder) directive(comment *ast.Comment) (Directive, bool, error) {
	text := comment.Text
	span := b.physicalSpan(comment)
	display := b.displaySpan(comment)
	record := func(kind catalog.DirectiveKind, tool, name, args string) (Directive, bool, error) {
		directive, err := NewDirective(
			kind, tool, name, args, span, display,
		)
		return directive, err == nil, err
	}
	if strings.HasPrefix(text, "//line ") || strings.HasPrefix(text, "/*line ") {
		return record(
			catalog.DirectiveLine,
			"line",
			"line",
			strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(text, "//line "), "/*line ")),
		)
	}
	if strings.HasPrefix(text, "// +build") || strings.HasPrefix(text, "//+build") {
		args := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(text, "// +build"), "//+build"))
		return record(catalog.DirectiveLegacyBuildTag, "+build", "+build", args)
	}
	parsed, present := ast.ParseDirective(comment.Slash, text)
	if !present {
		return Directive{}, false, nil
	}
	if parsed.Tool == "go" {
		kind := catalog.GoDirectiveByName(parsed.Name)
		if !kind.Valid() {
			return Directive{}, false, &Error{
				Phase: "UNKNOWN_DIRECTIVE", File: b.file.ID(),
				Span: span, Reason: "//go:" + parsed.Name + " is absent from the catalog",
			}
		}
		return record(kind, parsed.Tool, parsed.Name, parsed.Args)
	}
	return record(catalog.DirectiveExternalTool, parsed.Tool, parsed.Name, parsed.Args)
}
