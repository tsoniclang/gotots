// Command gotots is the single GoToTS binary. It parses flags and invokes the
// compiler; there is one compilation route. Unrecognized commands fail closed
// with a typed error and a non-zero exit — never a partial or best-effort run.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/tsoniclang/gotots/internal/compiler"
	"github.com/tsoniclang/gotots/internal/language/analyze"
	"github.com/tsoniclang/gotots/internal/source"
)

const usage = `usage: gotots <command> [args]

commands:
  inspect constructs -contract <id> [-contract-artifact <path>] [-dir <dir>]
                     [-manifest <artifact> -manifest-digest <hex>] [pattern ...]
      report canonical construct occurrences, evidence-depth partition,
      directives, and exact denominators for the selected workspace
      (default pattern ./...). The request selects its provider contract
      by identity (-contract-artifact supplies file acquisition for that
      identity); -manifest selects the produced audit/unit-manifest
      artifact with its certified digest — required whenever the closure
      contains provider-owned files
  audit catalog -contract <id> [-contract-artifact <path>] -o <artifact>
                [-dir <dir>] [pattern ...]
      run the manifest-producing streaming catalog-coverage audit and
      write the versioned, fingerprinted gate artifact plus its certified
      digest (run with the toolchain patterns "std cmd" to produce the
      complete selected-toolchain artifact)
  audit verify -contract <id> [-contract-artifact <path>] -a <artifact>
               [-dir <dir>] [pattern ...]
      exact-join a stored artifact bidirectionally against a freshly and
      recursively re-derived universe of the same patterns (the
      certification gate)`

// UnsupportedCommandError reports a command line the binary does not implement.
// It is the CLI-level typed error; its string is rendered only here.
type UnsupportedCommandError struct {
	Command string
}

func (e *UnsupportedCommandError) Error() string {
	return fmt.Sprintf("GOTOTS_UNSUPPORTED_COMMAND: %q is not a supported command\n\n%s", e.Command, usage)
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run dispatches one command. It is separated from main for testability and
// returns a typed error rather than exiting.
func run(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return &UnsupportedCommandError{Command: ""}
	}
	switch args[0] {
	case "inspect":
		return runInspect(args[1:], stdout)
	case "audit":
		return runAudit(args[1:], stdout)
	default:
		return &UnsupportedCommandError{Command: strings.Join(args, " ")}
	}
}

// parseCommon collects the flags every command shares. The provider contract
// is an explicit request selection — there is no implicit default.
type commonFlags struct {
	request  source.Request
	patterns []string
}

func parseCommon(command string, rest []string, extra map[string]*string) (commonFlags, error) {
	out := commonFlags{request: source.Request{Dir: "."}}
	for i := 0; i < len(rest); i++ {
		if target, known := extra[rest[i]]; known {
			if i+1 >= len(rest) {
				return out, &UnsupportedCommandError{Command: command + " " + rest[i] + " (missing value)"}
			}
			i++
			*target = rest[i]
			continue
		}
		switch {
		case rest[i] == "-dir":
			if i+1 >= len(rest) {
				return out, &UnsupportedCommandError{Command: command + " -dir (missing directory)"}
			}
			i++
			out.request.Dir = rest[i]
		case rest[i] == "-contract":
			if i+1 >= len(rest) {
				return out, &UnsupportedCommandError{Command: command + " -contract (missing identity)"}
			}
			i++
			out.request.ProviderContract = rest[i]
		case rest[i] == "-contract-artifact":
			if i+1 >= len(rest) {
				return out, &UnsupportedCommandError{Command: command + " -contract-artifact (missing path)"}
			}
			i++
			out.request.ProviderContractArtifact = rest[i]
		case strings.HasPrefix(rest[i], "-"):
			return out, &UnsupportedCommandError{Command: command + " " + rest[i]}
		default:
			out.patterns = append(out.patterns, rest[i])
		}
	}
	out.request.Patterns = out.patterns
	return out, nil
}

func runInspect(args []string, stdout io.Writer) error {
	if len(args) == 0 || args[0] != "constructs" {
		return &UnsupportedCommandError{Command: strings.TrimSpace("inspect " + strings.Join(args, " "))}
	}
	manifestPath, manifestDigest := "", ""
	common, err := parseCommon("inspect constructs", args[1:], map[string]*string{
		"-manifest": &manifestPath, "-manifest-digest": &manifestDigest,
	})
	if err != nil {
		return err
	}
	common.request.AuditArtifact = manifestPath
	common.request.AuditArtifactDigest = manifestDigest
	inspection, err := compiler.InspectConstructs(common.request)
	if err != nil {
		return err
	}
	manifestState := "none"
	if manifestPath != "" {
		manifestState = "consumed"
	}
	return printInspection(stdout, inspection, manifestState)
}

func runAudit(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return &UnsupportedCommandError{Command: "audit"}
	}
	switch args[0] {
	case "catalog":
		return runAuditCatalog(args[1:], stdout)
	case "verify":
		return runAuditVerify(args[1:], stdout)
	}
	return &UnsupportedCommandError{Command: strings.TrimSpace("audit " + strings.Join(args, " "))}
}

func runAuditCatalog(rest []string, stdout io.Writer) error {
	out := ""
	common, err := parseCommon("audit catalog", rest, map[string]*string{"-o": &out})
	if err != nil {
		return err
	}
	if out == "" {
		return &UnsupportedCommandError{Command: "audit catalog (missing -o artifact path)"}
	}
	artifact, err := compiler.AuditCatalog(common.request)
	if err != nil {
		return err
	}
	if err := analyze.WriteAuditArtifact(artifact, out); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "catalog audit: toolchain=%s catalog=%s files=%d occurrences=%d directives=%d unknown=0 -> %s\n",
		artifact.Meta.ToolchainVersion, artifact.Meta.CatalogVersion, len(artifact.Files), artifact.Occurrences, artifact.Directives, out); err != nil {
		return err
	}
	// The certified digest is the consumer's external binding; print it in
	// full so the consuming request can select it exactly.
	_, err = fmt.Fprintf(stdout, "certifiedDigest=%s\n", artifact.ArtifactDigest)
	return err
}

func runAuditVerify(rest []string, stdout io.Writer) error {
	artifactPath := ""
	common, err := parseCommon("audit verify", rest, map[string]*string{"-a": &artifactPath})
	if err != nil {
		return err
	}
	if artifactPath == "" {
		return &UnsupportedCommandError{Command: "audit verify (missing -a artifact path)"}
	}
	// The coverage gate independently re-derives every provider interior
	// (recursive policy) and joins the artifact bidirectionally, including
	// per-file unit manifests.
	if err := compiler.AuditVerify(common.request, artifactPath); err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "audit verify: artifact %s exact-joins the independently derived universe\n", artifactPath)
	return err
}

// printInspection renders the resolved universe, the canonical occurrence
// records, and the exact denominators. Write errors propagate; a failing
// writer fails the command.
func printInspection(stdout io.Writer, inspection *compiler.Inspection, manifestState string) error {
	p := func(format string, args ...any) error {
		_, err := fmt.Fprintf(stdout, format, args...)
		return err
	}
	ws := inspection.Workspace()
	if err := p("toolchain %s (%s) binaryDigest=%s\n",
		ws.Toolchain().Version(), ws.Toolchain().Binary(), ws.Toolchain().BinaryDigest()[:12]); err != nil {
		return err
	}
	if err := p("contract %s fingerprint=%s\n",
		inspection.Selection().ContractID(), inspection.Selection().ContractFingerprint()); err != nil {
		return err
	}
	ownerCounts := map[string]int{}
	for _, pkg := range ws.Packages() {
		ownerCounts[pkg.ID().Owner().Class().String()]++
		if err := p("universe %s provenance=%s acquisition=%s disposition=%s moduleGo=%s root=%v\n",
			pkg.ID(), pkg.Provenance(), pkg.Acquisition(), pkg.Disposition(), pkg.ModuleGoVersion(), pkg.RequestedRoot()); err != nil {
			return err
		}
	}
	inventory := inspection.Inventory()
	for _, pkg := range inventory.Packages() {
		if err := p("package %s\n", pkg.ID()); err != nil {
			return err
		}
		for _, file := range pkg.Files() {
			if err := p("  file %s goVersion=%s\n", file.File(), file.EffectiveGoVersion()); err != nil {
				return err
			}
			for _, occurrence := range file.Occurrences() {
				line := fmt.Sprintf("    %s kind=%s", occurrence.ID(), occurrence.Kind())
				if occurrence.Edge().Valid() {
					line += fmt.Sprintf(" edge=%s role=%s", occurrence.Edge(), occurrence.Role())
				}
				if occurrence.Token().Valid() {
					line += " token=" + occurrence.Token().String()
				}
				if occurrence.Variant() != 0 {
					line += " variant=" + occurrence.Variant().String()
				}
				for _, op := range occurrence.Implicit() {
					line += " implicit=" + op.String()
				}
				if err := p("%s\n", line); err != nil {
					return err
				}
			}
			for _, directive := range file.Directives() {
				if err := p("    directive %s tool=%s name=%s disposition=%s\n",
					directive.Kind(), directive.Tool(), directive.Name(), directive.Kind().Disposition()); err != nil {
					return err
				}
			}
		}
	}
	depthCounts := map[string]int{}
	implicitCount := 0
	for _, pkg := range ws.Packages() {
		for _, unit := range pkg.Units() {
			depthCounts[unit.Depth().String()]++
		}
		implicitCount += len(pkg.ImplicitUnits())
	}
	if err := p("depths: fullSemantic=%d declarationContract=%d externalBoundary=%d intrinsic=%d implicitUnits=%d\n",
		depthCounts["full-semantic"], depthCounts["declaration-contract"],
		depthCounts["external-boundary"], depthCounts["intrinsic"], implicitCount); err != nil {
		return err
	}
	d := inventory.Denominators()
	if err := p("universe: closurePackages=%d moduleOwned=%d std=%d toolchain=%d languagePseudo=%d\n",
		len(ws.Packages()), ownerCounts["module"], ownerCounts["standard-library"],
		ownerCounts["toolchain"], ownerCounts["language-pseudo"]); err != nil {
		return err
	}
	return p("denominators: sourcePackages=%d files=%d occurrences=%d directives=%d variantBearing=%d implicitOps=%d unknownConstructs=0 unknownDirectives=0 manifest=%s\n",
		d.Packages, d.Files, d.Occurrences, d.Directives, d.VariantBearing, d.ImplicitOps, manifestState)
}
