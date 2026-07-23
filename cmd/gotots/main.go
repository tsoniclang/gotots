// Command gotots is the single fail-closed GoToTS binary.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/tsoniclang/gotots/internal/compiler"
	"github.com/tsoniclang/gotots/internal/scope/sourceplan"
	"github.com/tsoniclang/gotots/internal/source"
)

const usage = `usage: gotots <command> [args]

commands:
  inspect constructs -contract <id> [-contract-artifact <path>] [-dir <dir>]
                     [-provider <artifact> -provider-digest <hex>] [pattern ...]
  audit catalog -contract <id> [-contract-artifact <path>] -o <artifact>
                [-dir <dir>] [pattern ...]
  audit verify -contract <id> [-contract-artifact <path>] -a <artifact>
               [-dir <dir>] [pattern ...]`

type UnsupportedCommandError struct{ Command string }

func (e *UnsupportedCommandError) Error() string {
	return fmt.Sprintf(
		"GOTOTS_UNSUPPORTED_COMMAND: %q is not supported\n\n%s",
		e.Command,
		usage,
	)
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return &UnsupportedCommandError{}
	}
	switch args[0] {
	case "inspect":
		return runInspect(args[1:], stdout)
	case "audit":
		return runAudit(args[1:], stdout)
	default:
		return &UnsupportedCommandError{
			Command: strings.Join(args, " "),
		}
	}
}

type commonFlags struct {
	request  source.Request
	patterns []string
}

func parseCommon(
	command string,
	arguments []string,
	extra map[string]*string,
) (commonFlags, error) {
	out := commonFlags{request: source.Request{Dir: "."}}
	for index := 0; index < len(arguments); index++ {
		argument := arguments[index]
		if target, known := extra[argument]; known {
			if index+1 >= len(arguments) {
				return out, &UnsupportedCommandError{
					Command: command + " " + argument + " (missing value)",
				}
			}
			index++
			*target = arguments[index]
			continue
		}
		switch argument {
		case "-dir":
			if index+1 >= len(arguments) {
				return out, &UnsupportedCommandError{
					Command: command + " -dir (missing value)",
				}
			}
			index++
			out.request.Dir = arguments[index]
		case "-contract":
			if index+1 >= len(arguments) {
				return out, &UnsupportedCommandError{
					Command: command + " -contract (missing value)",
				}
			}
			index++
			out.request.ProviderContract = arguments[index]
		case "-contract-digest":
			if index+1 >= len(arguments) {
				return out, &UnsupportedCommandError{
					Command: command +
						" -contract-digest (missing value)",
				}
			}
			index++
			out.request.ProviderContractDigest = arguments[index]
		case "-contract-artifact":
			if index+1 >= len(arguments) {
				return out, &UnsupportedCommandError{
					Command: command +
						" -contract-artifact (missing value)",
				}
			}
			index++
			out.request.ProviderContractArtifact = arguments[index]
		default:
			if strings.HasPrefix(argument, "-") {
				return out, &UnsupportedCommandError{
					Command: command + " " + argument,
				}
			}
			out.patterns = append(out.patterns, argument)
		}
	}
	out.request.Patterns = append([]string(nil), out.patterns...)
	return out, nil
}

func runInspect(arguments []string, stdout io.Writer) error {
	if len(arguments) == 0 || arguments[0] != "constructs" {
		return &UnsupportedCommandError{
			Command: strings.TrimSpace(
				"inspect " + strings.Join(arguments, " "),
			),
		}
	}
	providerPath := ""
	providerDigest := ""
	common, err := parseCommon(
		"inspect constructs",
		arguments[1:],
		map[string]*string{
			"-provider":        &providerPath,
			"-provider-digest": &providerDigest,
		},
	)
	if err != nil {
		return err
	}
	common.request.AuditArtifact = providerPath
	common.request.AuditArtifactDigest = providerDigest
	inspection, err := compiler.InspectConstructs(common.request)
	if err != nil {
		return err
	}
	return printInspection(stdout, inspection)
}

func runAudit(arguments []string, stdout io.Writer) error {
	if len(arguments) == 0 {
		return &UnsupportedCommandError{Command: "audit"}
	}
	switch arguments[0] {
	case "catalog":
		return runAuditCatalog(arguments[1:], stdout)
	case "verify":
		return runAuditVerify(arguments[1:], stdout)
	default:
		return &UnsupportedCommandError{
			Command: strings.TrimSpace(
				"audit " + strings.Join(arguments, " "),
			),
		}
	}
}

func runAuditCatalog(arguments []string, stdout io.Writer) error {
	output := ""
	common, err := parseCommon(
		"audit catalog",
		arguments,
		map[string]*string{"-o": &output},
	)
	if err != nil {
		return err
	}
	if output == "" {
		return &UnsupportedCommandError{
			Command: "audit catalog (missing -o)",
		}
	}
	result, err := compiler.AuditCatalog(common.request, output)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(
		stdout,
		"provider audit: packageContexts=%d files=%d syntheticPackages=%d facts=%d encodedBytes=%d unknown=0 -> %s\ncertifiedDigest=%s\n",
		result.PackageContexts,
		result.Files,
		result.SyntheticPackages,
		result.Facts,
		result.EncodedBytes,
		output,
		result.Digest,
	)
	return err
}

func runAuditVerify(arguments []string, stdout io.Writer) error {
	artifactPath := ""
	common, err := parseCommon(
		"audit verify",
		arguments,
		map[string]*string{"-a": &artifactPath},
	)
	if err != nil {
		return err
	}
	if artifactPath == "" {
		return &UnsupportedCommandError{
			Command: "audit verify (missing -a)",
		}
	}
	if err := compiler.AuditVerify(
		common.request, artifactPath,
	); err != nil {
		return err
	}
	_, err = fmt.Fprintf(
		stdout,
		"audit verify: %s exact-joins independent Stage-1 derivation\n",
		artifactPath,
	)
	return err
}

func printInspection(
	stdout io.Writer,
	inspection *compiler.Inspection,
) error {
	print := func(format string, arguments ...any) error {
		_, err := fmt.Fprintf(stdout, format, arguments...)
		return err
	}
	workspace := inspection.Workspace()
	if err := print(
		"toolchain %s digest=%s config=%s\n",
		workspace.Toolchain().Version(),
		workspace.Toolchain().BinaryDigest()[:12],
		workspace.Toolchain().BuildConfigurationDigest()[:12],
	); err != nil {
		return err
	}
	localSourceFiles := 0
	certifiedSourceFiles := 0
	for _, decision := range inspection.SourcePlan().Files() {
		switch decision.Kind() {
		case sourceplan.KindLocalSyntax:
			localSourceFiles++
		case sourceplan.KindCertifiedGraph:
			certifiedSourceFiles++
		default:
			return fmt.Errorf(
				"verified inspection has invalid source authority %s",
				decision.Kind(),
			)
		}
	}
	selections := inspection.Selections()
	executableInventory := inspection.Executable()
	definitionCount := 0
	headerOccurrences := 0
	boundaryEntries := 0
	for _, pkg := range inspection.Structure().Packages() {
		for _, definition := range pkg.Definitions() {
			definitionCount++
			_, selected := selections.For(definition.ID())
			if !selected {
				return fmt.Errorf(
					"verified inspection lost selection %s",
					definition.ID(),
				)
			}
		}
		for _, header := range pkg.Headers() {
			headerOccurrences += len(header.Members())
		}
		for _, boundary := range pkg.Boundaries() {
			boundaryEntries += len(boundary.Entries())
		}
	}
	executableOccurrences := 0
	for _, region := range executableInventory.Regions() {
		executableOccurrences += len(region.Members())
	}
	hydration := inspection.Hydration()
	return print(
		"denominators: closurePackages=%d structuralFiles=%d localAuthorityFiles=%d certifiedAuthorityFiles=%d hydratedPackages=%d localSyntaxFiles=%d localSyntaxBytes=%d checkedViewPackages=%d definitions=%d selections=%d headers=%d headerOccurrences=%d boundaries=%d boundaryEntries=%d executableRegions=%d executableOccurrences=%d selectionFacts=%d unknownConstructs=0 unknownDirectives=0\n",
		len(workspace.Packages()),
		len(inspection.SourcePlan().Files()),
		localSourceFiles,
		certifiedSourceFiles,
		hydration.SemanticPackages,
		hydration.LocalFiles,
		hydration.LocalBytes,
		hydration.CheckedPackages,
		definitionCount,
		len(selections.Records()),
		definitionCount,
		headerOccurrences,
		definitionCount,
		boundaryEntries,
		len(executableInventory.Regions()),
		executableOccurrences,
		len(inspection.SelectionFacts().Facts()),
	)
}
