package shapegate

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// The typed-AST path is the shape AUTHORITY: modules parse through the
// pinned TypeScript compiler and verdicts join on AST facts. The regex
// detectors in shapegate.go are non-authoritative diagnostics retained
// for fast local signal only.

//go:embed extract-shape.mjs
var extractScript embed.FS

// FileShape is one parsed module's facts.
type FileShape struct {
	File         string        `json:"file"`
	Declarations []Declaration `json:"declarations"`
	Calls        []CallFact    `json:"calls"`
	Aliases      []AliasFact   `json:"aliases"`
}

type Declaration struct {
	Kind     string `json:"kind"` // function | class | method
	Name     string `json:"name"`
	Exported bool   `json:"exported"`
	Params   int    `json:"params"`
}

type CallFact struct {
	Callee string `json:"callee"`
	// ResolvedName/File/Line identify the callee's ORIGINAL declaration
	// through the TypeChecker — aliases and const-bound references
	// resolve here and cannot evade the joins.
	ResolvedName string `json:"resolvedName"`
	ResolvedFile string `json:"resolvedFile"`
	ResolvedLine int    `json:"resolvedLine"`
	Args         int    `json:"args"`
	Line         int    `json:"line"`
}

type AliasFact struct {
	Name     string `json:"name"`
	Exported bool   `json:"exported"`
}

// ExtractShapes parses the given TypeScript files with the pinned
// compiler (typescriptDir is the pinned module directory) and returns
// their AST facts. Node is required; a parse failure fails closed.
func ExtractShapes(nodeExecutable, typescriptDir string, files []string) ([]FileShape, error) {
	script, err := extractScript.ReadFile("extract-shape.mjs")
	if err != nil {
		return nil, err
	}
	scratch, err := os.CreateTemp("", "extract-shape-*.mjs")
	if err != nil {
		return nil, err
	}
	defer os.Remove(scratch.Name())
	if _, err := scratch.Write(script); err != nil {
		return nil, err
	}
	if err := scratch.Close(); err != nil {
		return nil, err
	}
	absTS, err := filepath.Abs(typescriptDir)
	if err != nil {
		return nil, err
	}
	args := append([]string{scratch.Name(), absTS}, files...)
	command := exec.Command(nodeExecutable, args...)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return nil, fmt.Errorf("shape extraction: %w (%s)", err, stderr.String())
	}
	var shapes []FileShape
	if err := json.Unmarshal(stdout.Bytes(), &shapes); err != nil {
		return nil, fmt.Errorf("shape extraction output: %w", err)
	}
	return shapes, nil
}

// DuplicateDefinitions joins declarations across modules by name and
// returns every definition identity declared in more than one module —
// the AST-authoritative replacement for the textual counter. Aliases
// participate: a type alias re-declared per consumer module is the
// measured interface-duplication class.
func DuplicateDefinitions(shapes []FileShape) map[string][]string {
	owners := map[string][]string{}
	for _, shape := range shapes {
		for _, decl := range shape.Declarations {
			if decl.Kind == "function" || decl.Kind == "class" {
				owners[decl.Kind+"::"+decl.Name] = append(owners[decl.Kind+"::"+decl.Name], shape.File)
			}
		}
		for _, alias := range shape.Aliases {
			owners["alias::"+alias.Name] = append(owners["alias::"+alias.Name], shape.File)
		}
	}
	out := map[string][]string{}
	for identity, files := range owners {
		if len(files) > 1 {
			out[identity] = files
		}
	}
	return out
}

// CallArgSurplus reports calls to the named callee whose argument count
// exceeds the source arity — the AST-authoritative hidden-argument
// check for a known call fact (source arities come from the Go side).
func CallArgSurplus(shapes []FileShape, callee string, sourceArity int) []CallFact {
	var out []CallFact
	for _, shape := range shapes {
		for _, call := range shape.Calls {
			if call.ResolvedName == callee && call.Args > sourceArity {
				out = append(out, call)
			}
		}
	}
	return out
}
