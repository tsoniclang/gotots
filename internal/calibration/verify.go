package calibration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Port verification, stage 1 of the strict/differential harness: every
// authored hand port must PARSE cleanly through the pinned TypeScript
// compiler. Method-body snippets (the corpus convention for receiver
// methods) wrap in a fixture class before parsing. Strict semantic
// typechecking and differential execution are later stages gated on
// the reviewed ambient-declaration scheme; their absence is recorded
// per fixture, never implied away.

// PortVerification is one fixture's harness state.
type PortVerification struct {
	FixtureID   string   `json:"fixtureId"`
	Wrapped     bool     `json:"wrappedAsMethod"`
	ParseClean  bool     `json:"parseClean"`
	Diagnostics []string `json:"diagnostics,omitempty"`
	// Pending names the harness stages not yet run for this fixture.
	Pending []string `json:"pending"`
}

const parseScript = `
const { createRequire } = require("node:module");
const { readFileSync } = require("node:fs");
const ts = require(process.argv[2]);
const out = [];
for (const file of process.argv.slice(3)) {
  const text = readFileSync(file, "utf8");
  const source = ts.createSourceFile(file, text, ts.ScriptTarget.ES2022, true);
  const diagnostics = source.parseDiagnostics.map((d) =>
    file + ":" + ts.getLineAndCharacterOfPosition(source, d.start).line + ": " +
    ts.flattenDiagnosticMessageText(d.messageText, " "));
  out.push({ file, diagnostics });
}
process.stdout.write(JSON.stringify(out));
`

// stripReviewHeader removes the leading //-comment reviewer header (the
// same convention the measurer uses).
func stripReviewHeader(source string) string {
	lines := strings.SplitAfter(source, "\n")
	start := 0
	for start < len(lines) {
		trimmed := strings.TrimSpace(lines[start])
		if strings.HasPrefix(trimmed, "//") || trimmed == "" {
			start++
			continue
		}
		break
	}
	return strings.Join(lines[start:], "")
}

// isMethodSnippet reports whether the port's first code line is method
// syntax (no `function`/`const`/`class` keyword before the signature).
func isMethodSnippet(code string) bool {
	first := strings.TrimSpace(strings.SplitN(code, "\n", 2)[0])
	if strings.HasPrefix(first, "function ") || strings.HasPrefix(first, "export ") ||
		strings.HasPrefix(first, "const ") || strings.HasPrefix(first, "class ") ||
		strings.HasPrefix(first, "let ") {
		return false
	}
	if !strings.HasSuffix(first, "{") {
		// A bare statement (module-init effect fixtures) is not a
		// method signature.
		return false
	}
	open := strings.Index(first, "(")
	return open > 0
}

// VerifyPorts runs the parse stage over every authored port and
// returns per-fixture verifications sorted by fixture ID.
func VerifyPorts(handportDir, nodeExecutable, typescriptDir string) ([]PortVerification, error) {
	entries, err := filepath.Glob(filepath.Join(handportDir, "*.ts"))
	if err != nil {
		return nil, err
	}
	scratch, err := os.MkdirTemp("", "port-verify-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(scratch)

	var files []string
	var verifications []PortVerification
	for _, entry := range entries {
		fixture := strings.TrimSuffix(filepath.Base(entry), ".ts")
		data, err := os.ReadFile(entry)
		if err != nil {
			return nil, err
		}
		code := stripReviewHeader(string(data))
		verification := PortVerification{FixtureID: fixture,
			Pending: []string{"strict-typecheck", "differential-execution"}}
		if isMethodSnippet(code) {
			verification.Wrapped = true
			code = "class $Fixture {\n" + code + "\n}\n"
		}
		wrapped := filepath.Join(scratch, fixture+".ts")
		if err := os.WriteFile(wrapped, []byte(code), 0o644); err != nil {
			return nil, err
		}
		files = append(files, wrapped)
		verifications = append(verifications, verification)
	}

	script := filepath.Join(scratch, "parse.cjs")
	if err := os.WriteFile(script, []byte(parseScript), 0o644); err != nil {
		return nil, err
	}
	absTS, err := filepath.Abs(typescriptDir)
	if err != nil {
		return nil, err
	}
	command := exec.Command(nodeExecutable, append([]string{script, absTS}, files...)...)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return nil, fmt.Errorf("parse harness: %w (%s)", err, stderr.String())
	}
	var results []struct {
		File        string   `json:"file"`
		Diagnostics []string `json:"diagnostics"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil {
		return nil, err
	}
	byFile := map[string][]string{}
	for _, result := range results {
		byFile[strings.TrimSuffix(filepath.Base(result.File), ".ts")] = result.Diagnostics
	}
	for i := range verifications {
		diagnostics := byFile[verifications[i].FixtureID]
		verifications[i].ParseClean = len(diagnostics) == 0
		verifications[i].Diagnostics = diagnostics
	}
	return verifications, nil
}
