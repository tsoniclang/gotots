// Package staticness implements the generated-output staticness sweep of
// spec chapter 11: every invocation, member selection, and dispatch in
// generated output — the language ABI, generated core, external stubs,
// and helpers — must be a direct statically typed operation or a finite
// closed-branch dynamic switch with a necessity record. Erased dispatch
// (name-selected members, universal operation registries, Function /
// unknown[] call carriers, reflection-like target recovery, typed
// wrappers around erased dispatch) is prohibited. The sweep reports
// every prohibited site; there are no file-local suppressions.
package staticness

import (
	"regexp"
	"sort"
	"strings"
)

// Violation is one prohibited-dispatch site in a generated file.
type Violation struct {
	File    string
	Line    int
	Pattern string // the prohibited-pattern identity
	Detail  string // the offending source line, trimmed
}

// pattern is one prohibited construct, matched structurally against a
// generated line.
type pattern struct {
	id      string
	matches func(line string) bool
}

// contains builds a substring matcher.
func contains(id, needle string) pattern {
	return pattern{id: id, matches: func(line string) bool { return strings.Contains(line, needle) }}
}

// prohibited is the complete set of erased-dispatch constructs. Each has
// one identity; there is no allowlist that exempts a file.
var prohibited = []pattern{
	// Name-selected interface / member dispatch.
	contains("iface-name-dispatch", "goIfaceCall("),
	contains("method-table-index", ".m["),
	// Universal operation registries and erased method tables.
	contains("record-function-table", "Record<string, Function>"),
	contains("map-function-table", "Map<string, Function>"),
	// Erased function-value invocation.
	contains("erased-func-invoke", "goFuncInvoke("),
	// Runtime external dispatch registry.
	contains("external-registry-call", "goExternalCall("),
	contains("external-registry-register", "goExternalRegister("),
	// Erased callable carriers used in call/return position.
	pattern{id: "erased-function-type", matches: func(line string) bool {
		return strings.Contains(line, ": Function") || strings.Contains(line, "| Function") ||
			strings.Contains(line, "as Function") || strings.Contains(line, "<Function>")
	}},
	// Reflection-like recovery.
	contains("eval", "eval("),
	contains("proxy", "new Proxy("),
	contains("function-constructor", "new Function("),
	contains("reflect-construct", "Reflect."),
	contains("constructor-name", ".constructor"),
	// Computed-member invocation: a call through a dynamically selected
	// member is name-selected dispatch. Matches `[<expr>](` and
	// `[<expr>].call(`/`.apply(`.
	pattern{id: "computed-member-call", matches: func(line string) bool {
		return computedMemberCall.MatchString(line)
	}},
	// Erased argument carriers passed positionally to a call.
	pattern{id: "erased-arg-array", matches: func(line string) bool {
		return strings.Contains(line, "...(args as unknown[])") ||
			strings.Contains(line, "as unknown[])(") || strings.Contains(line, ": unknown[]): unknown")
	}},
	// A cast recovering a value from an erased dispatch result.
	contains("erased-result-cast", ") as unknown)("),
}

// computedMemberCall matches a call through a computed (dynamically
// selected) member — obj[expr](...) or obj[expr].call/apply — which is
// name-selected dispatch. It deliberately does not match numeric or
// bigint index reads followed by no call.
var computedMemberCall = regexp.MustCompile(`\][ ]*\(|\][ ]*\.(call|apply)\(`)

// Sweep scans every generated file and returns each prohibited site,
// sorted by file then line. files maps generated paths to source.
func Sweep(files map[string]string) []Violation {
	var out []Violation
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		for i, line := range strings.Split(files[path], "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.HasPrefix(trimmed, "//") {
				continue
			}
			for _, p := range prohibited {
				if p.matches(line) {
					out = append(out, Violation{
						File: path, Line: i + 1, Pattern: p.id, Detail: trimmed,
					})
				}
			}
		}
	}
	return out
}

// Counts summarizes violations per pattern identity in sorted order.
func Counts(violations []Violation) map[string]int {
	counts := map[string]int{}
	for _, v := range violations {
		counts[v.Pattern]++
	}
	return counts
}
