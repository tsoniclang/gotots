// Package oracle executes Go semantic differential tests: a fixture
// package is compiled and run with the Go toolchain, translated with the
// GoToTS pipeline, executed as generated TypeScript under Node, and the
// outputs are compared exactly — return values, panic categories, and
// termination.
//
// The fixture protocol: a module of one or more packages translated as a
// unit, whose "fixture" package's exported zero-parameter functions are
// the cases. Each case's results (or its panic) are printed one line per
// case in sorted name order by generated drivers on both sides.
package oracle

import (
	"bytes"
	"fmt"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"golang.org/x/tools/go/packages"

	"github.com/tsoniclang/gotots/internal/goenv"
	"github.com/tsoniclang/gotots/internal/profile"
	"github.com/tsoniclang/gotots/internal/translate"
	"github.com/tsoniclang/gotots/internal/typedload"
)

// Result is one differential run's evidence.
type Result struct {
	GoOutput string
	TSOutput string
	Cases    int
}

// Match reports byte-identical outputs.
func (r *Result) Match() bool { return r.GoOutput == r.TSOutput }

var bootstrapOnce = sync.OnceValues(func() (*goenv.Resolved, error) {
	goExecutable, err := goenv.Locate()
	if err != nil {
		return nil, err
	}
	return goenv.Bootstrap(goExecutable)
})

// tscOnce locates the pinned TypeScript compiler and the strict
// configuration in the repository (resolved from this source file's
// location, so tests in temp working directories still find it).
var tscOnce = sync.OnceValues(func() (*strictTsc, error) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return nil, fmt.Errorf("cannot resolve the oracle package path")
	}
	repo := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	tscJs := filepath.Join(repo, "product", "node_modules", "typescript", "lib", "tsc.js")
	if _, err := os.Stat(tscJs); err != nil {
		return nil, fmt.Errorf("pinned tsc.js: %w", err)
	}
	strictConfig, err := os.ReadFile(filepath.Join(repo, "product", "tsconfig.strict.json"))
	if err != nil {
		return nil, fmt.Errorf("strict tsconfig: %w", err)
	}
	return &strictTsc{tscJs: tscJs, strictConfig: strictConfig}, nil
})

type strictTsc struct {
	tscJs        string
	strictConfig []byte
}

// typecheckFixture strict-typechecks one differential fixture's complete
// generated tree (including assembled implementation modules) with the
// pinned TypeScript compiler before any execution: a fixture that does
// not typecheck must never be executed under type stripping, where type
// errors are silently ignored.
func typecheckFixture(nodeExecutable, workDir string) error {
	tsc, err := tscOnce()
	if err != nil {
		return err
	}
	generatedDir := filepath.Join(workDir, "generated")
	if err := os.WriteFile(filepath.Join(generatedDir, "tsconfig.json"), tsc.strictConfig, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(generatedDir, "package.json"), []byte("{\n  \"type\": \"module\"\n}\n"), 0o644); err != nil {
		return err
	}
	command := exec.Command(nodeExecutable, tsc.tscJs, "-p", ".")
	command.Dir = generatedDir
	out, err := command.CombinedOutput()
	if err != nil {
		text := string(out)
		if len(text) > 4000 {
			text = text[:4000] + "\n... (truncated)"
		}
		return fmt.Errorf("fixture failed the strict typecheck (fix the generated types, never weaken the check):\n%s", text)
	}
	return nil
}

// Translate materializes a fixture module in workDir — one Go package per
// entry, mapping the package directory to its single-file source, with
// "fixture" as the case package — loads it through the shared typed
// loader, and translates every package as one unit.
func Translate(workDir string, packageSources map[string]string) (*translate.Generated, *packages.Package, error) {
	resolved, err := bootstrapOnce()
	if err != nil {
		return nil, nil, err
	}
	if _, hasFixture := packageSources["fixture"]; !hasFixture {
		return nil, nil, fmt.Errorf("fixture module needs a %q case package", "fixture")
	}
	if err := os.WriteFile(filepath.Join(workDir, "go.mod"), []byte("module oracle.fixture\n\ngo 1.26\n"), 0o644); err != nil {
		return nil, nil, err
	}
	ownedRoots := make([]string, 0, len(packageSources))
	for dir := range packageSources {
		ownedRoots = append(ownedRoots, dir)
	}
	sort.Strings(ownedRoots)
	for _, dir := range ownedRoots {
		if err := os.MkdirAll(filepath.Join(workDir, dir), 0o755); err != nil {
			return nil, nil, err
		}
		file := filepath.Base(dir) + ".go"
		if err := os.WriteFile(filepath.Join(workDir, dir, file), []byte(packageSources[dir]), 0o644); err != nil {
			return nil, nil, err
		}
	}

	prof := &profile.Profile{
		GoModule:   "oracle.fixture",
		OwnedRoots: ownedRoots,
	}
	hostEnv, err := hostEnviron(resolved)
	if err != nil {
		return nil, nil, err
	}
	loaded, err := typedload.Load(prof, hostEnv, workDir)
	if err != nil {
		return nil, nil, err
	}
	var unit []*packages.Package
	var fixture *packages.Package
	for _, p := range loaded {
		if typedload.RoleOf(p, workDir) != typedload.RoleProduction {
			continue
		}
		unit = append(unit, p)
		if p.PkgPath == "oracle.fixture/fixture" {
			fixture = p
		}
	}
	if fixture == nil {
		return nil, nil, fmt.Errorf("fixture package did not load")
	}

	generated, err := translate.Packages(unit, workDir, translate.Options{
		SourceRevision: "oracle-fixture",
		ProfileHash:    "oracle",
	})
	if err != nil {
		return nil, nil, err
	}
	if len(generated.Withheld) > 0 {
		// Oracles execute complete runnable fixtures; a withheld package
		// surfaces every recorded unsupported site.
		var lines []string
		for _, support := range generated.Support {
			for _, site := range support.Sites {
				lines = append(lines, fmt.Sprintf("%s:\n%s at %s:%d:%d",
					site.Code, site.Construct, site.Span.File, site.Span.Line, site.Span.Col))
			}
		}
		sort.Strings(lines)
		return nil, nil, fmt.Errorf("fixture translation withheld:\n%s", strings.Join(lines, "\n"))
	}
	return generated, fixture, nil
}

// Run translates a fixture module (package directory -> single-file
// source, "fixture" being the case package) and executes both sides. The
// Node toolchain is required: its absence blocks the gate rather than
// skipping it.
func Run(workDir string, packageSources map[string]string) (*Result, error) {
	return RunAssembled(workDir, packageSources, nil)
}

// RunAssembled additionally assembles reviewed implementation modules in
// place of generated external stubs — keyed by external package path,
// each module exports exactly the stub's typed symbols — mirroring the
// product's static assembly step. No runtime registration exists.
func RunAssembled(workDir string, packageSources map[string]string, implementations map[string]string) (*Result, error) {
	resolved, err := bootstrapOnce()
	if err != nil {
		return nil, err
	}
	nodeExecutable, err := exec.LookPath("node")
	if err != nil {
		return nil, fmt.Errorf("node toolchain is required for differential oracles: %w", err)
	}

	generated, fixture, err := Translate(workDir, packageSources)
	if err != nil {
		return nil, err
	}
	cases, err := fixtureCases(fixture)
	if err != nil {
		return nil, err
	}
	if len(cases) == 0 {
		return nil, fmt.Errorf("fixture has no exported zero-parameter case functions")
	}

	generatedDir := filepath.Join(workDir, "generated")
	for path, content := range generated.Files {
		full := filepath.Join(generatedDir, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			return nil, err
		}
	}
	// Static assembly: each implementation module replaces its stub at
	// the same path, so every import site binds the reviewed behavior.
	for externalPkg, moduleTS := range implementations {
		full := filepath.Join(generatedDir, "external-stubs", filepath.FromSlash(externalPkg), "package.ts")
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(full, []byte(moduleTS), 0o644); err != nil {
			return nil, err
		}
	}

	nodeForTsc, err := exec.LookPath("node")
	if err != nil {
		return nil, err
	}
	if err := typecheckFixture(nodeForTsc, workDir); err != nil {
		return nil, err
	}

	hostEnv, err := hostEnviron(resolved)
	if err != nil {
		return nil, err
	}
	goOutput, err := runGoDriver(resolved, hostEnv, workDir, cases)
	if err != nil {
		return nil, err
	}
	tsOutput, err := runNodeDriver(nodeExecutable, workDir, cases)
	if err != nil {
		return nil, err
	}
	return &Result{GoOutput: goOutput, TSOutput: tsOutput, Cases: len(cases)}, nil
}

// fixtureCase is one exported zero-parameter function with the formatting
// kind of each result. Floats compare by exact IEEE-754 bit pattern so
// driver number formatting can never mask or invent a difference.
type fixtureCase struct {
	Name        string
	ResultKinds []string // "float" | "value"
}

func fixtureCases(p *packages.Package) ([]fixtureCase, error) {
	scope := p.Types.Scope()
	var cases []fixtureCase
	for _, name := range scope.Names() {
		object := scope.Lookup(name)
		function, ok := object.(*types.Func)
		if !ok || !function.Exported() {
			continue
		}
		signature := function.Type().(*types.Signature)
		if signature.Params().Len() != 0 || signature.Recv() != nil {
			return nil, fmt.Errorf("fixture case %s must take no parameters", name)
		}
		c := fixtureCase{Name: name}
		results := signature.Results()
		for i := range results.Len() {
			kind := "value"
			if basic, ok := results.At(i).Type().Underlying().(*types.Basic); ok && basic.Info()&types.IsFloat != 0 {
				kind = "float"
			}
			c.ResultKinds = append(c.ResultKinds, kind)
		}
		cases = append(cases, c)
	}
	sort.Slice(cases, func(i, j int) bool { return cases[i].Name < cases[j].Name })
	return cases, nil
}

// hostEnviron builds the hermetic environment for the host platform.
func hostEnviron(resolved *goenv.Resolved) ([]string, error) {
	command := exec.Command(resolved.GoExecutable, "env", "GOHOSTOS", "GOHOSTARCH")
	command.Env = goenv.BootstrapEnviron()
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("go env: %w", err)
	}
	fields := strings.Fields(string(output))
	if len(fields) != 2 {
		return nil, fmt.Errorf("unexpected go env output %q", output)
	}
	options := goenv.EnvOptions{GOOS: fields[0], GOARCH: fields[1]}
	if options.GOARCH == "amd64" {
		options.GOAMD64 = "v1"
	}
	if options.GOARCH == "arm64" {
		options.GOARM64 = "v8.0"
	}
	return resolved.Environ(options), nil
}

func runCommand(executable string, args []string, dir string, env []string) (string, error) {
	command := exec.Command(executable, args...)
	command.Dir = dir
	if env != nil {
		command.Env = env
	}
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return "", fmt.Errorf("%s %s: %v\nstdout:\n%s\nstderr:\n%s",
			executable, strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String(), nil
}
