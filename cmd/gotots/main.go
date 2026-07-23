// Command gotots is the single fail-closed GoToTS binary.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/tsoniclang/gotots/internal/compiler"
	"github.com/tsoniclang/gotots/internal/scope/contract"
	"github.com/tsoniclang/gotots/internal/scope/sourceplan"
	"github.com/tsoniclang/gotots/internal/source"
)

const usage = `usage: gotots <command> [args]

commands:
  inspect constructs -contract <id> [-contract-artifact <path>] [-dir <dir>]
                     [-provider-structure <artifact>
                      -provider-structure-digest <hex>
                      -provider-semantics <artifact>
                      -provider-semantics-digest <hex>] [pattern ...]
  audit catalog -contract <id> [-contract-artifact <path>]
                -structure <artifact> -semantics <artifact>
                [-dir <dir>] [pattern ...]
  audit verify -contract <id> [-contract-artifact <path>]
               -structure <artifact> -semantics <artifact>
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
	structurePath := ""
	structureDigest := ""
	semanticPath := ""
	semanticDigest := ""
	common, err := parseCommon(
		"inspect constructs",
		arguments[1:],
		map[string]*string{
			"-provider-structure":        &structurePath,
			"-provider-structure-digest": &structureDigest,
			"-provider-semantics":        &semanticPath,
			"-provider-semantics-digest": &semanticDigest,
		},
	)
	if err != nil {
		return err
	}
	common.request.ProviderStructureArtifact = structurePath
	common.request.ProviderStructureDigest = structureDigest
	common.request.ProviderSemanticArtifact = semanticPath
	common.request.ProviderSemanticDigest = semanticDigest
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
	structureOutput := ""
	semanticOutput := ""
	common, err := parseCommon(
		"audit catalog",
		arguments,
		map[string]*string{
			"-structure": &structureOutput,
			"-semantics": &semanticOutput,
		},
	)
	if err != nil {
		return err
	}
	if structureOutput == "" || semanticOutput == "" {
		return &UnsupportedCommandError{
			Command: "audit catalog (missing -structure or -semantics)",
		}
	}
	result, err := compiler.AuditCatalog(
		common.request, structureOutput, semanticOutput,
	)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(
		stdout,
		"provider structure: packageContexts=%d files=%d syntheticPackages=%d definitions=%d headerOccurrences=%d boundaryEntries=%d facts=%d largestShardBytes=%d largestPackageRecords=%d encodedBytes=%d unknown=0 -> %s\ndigest=%s\nprovider semantics: packages=%d definitions=%d resolutions=%d declarations=%d bindings=%d types=%d operations=%d unsupported=%d typeClosureDuplicates=%d largestShardBytes=%d largestPackageRecords=%d encodedBytes=%d -> %s\ndigest=%s\n",
		result.Structure.PackageContexts,
		result.Structure.Files,
		result.Structure.SyntheticPackages,
		result.Structure.Definitions,
		result.Structure.HeaderOccurrences,
		result.Structure.BoundaryEntries,
		result.Structure.Facts,
		result.Structure.LargestShardBytes,
		result.Structure.LargestPackageRecords,
		result.Structure.EncodedBytes,
		structureOutput,
		result.Structure.Digest,
		result.Semantic.Packages,
		result.Semantic.Definitions,
		result.Semantic.Resolutions,
		result.Semantic.Declarations,
		result.Semantic.Bindings,
		result.Semantic.Types,
		result.Semantic.Operations,
		result.Semantic.Unsupported,
		result.Semantic.TypeClosureDuplicates,
		result.Semantic.LargestShardBytes,
		result.Semantic.LargestPackageRecords,
		result.Semantic.EncodedBytes,
		semanticOutput,
		result.Semantic.Digest,
	)
	if err != nil {
		return err
	}
	for index, record := range result.Structure.LargestPackages() {
		if _, err := fmt.Fprintf(
			stdout,
			"provider-production-tail rank=%d package=%s bytes=%d records=%d\n",
			index+1,
			record.Package,
			record.Bytes,
			record.Records,
		); err != nil {
			return err
		}
	}
	for index, record := range result.Structure.LargestHeaders() {
		if _, err := fmt.Fprintf(
			stdout,
			"provider-header-tail rank=%d header=%s encodedBytes=%d occurrences=%d\n",
			index+1,
			record.Header,
			record.EncodedBytes,
			record.Occurrences,
		); err != nil {
			return err
		}
	}
	return nil
}

func runAuditVerify(arguments []string, stdout io.Writer) error {
	structurePath := ""
	semanticPath := ""
	common, err := parseCommon(
		"audit verify",
		arguments,
		map[string]*string{
			"-structure": &structurePath,
			"-semantics": &semanticPath,
		},
	)
	if err != nil {
		return err
	}
	if structurePath == "" || semanticPath == "" {
		return &UnsupportedCommandError{
			Command: "audit verify (missing -structure or -semantics)",
		}
	}
	if err := compiler.AuditVerify(
		common.request, structurePath, semanticPath,
	); err != nil {
		return err
	}
	_, err = fmt.Fprintf(
		stdout,
		"audit verify: structure=%s semantics=%s exact-join independent Stage-1 and Stage-2 derivations\n",
		structurePath,
		semanticPath,
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
	fullSemanticDefinitions := 0
	declarationContractDefinitions := 0
	externalBoundaryDefinitions := 0
	intrinsicDefinitions := 0
	headerOccurrences := 0
	boundaryEntries := 0
	for _, record := range inspection.Structure().DefinitionCensus() {
		definition := record.ID()
		definitionCount++
		if _, selected := selections.For(definition); !selected {
			return fmt.Errorf(
				"verified inspection lost selection %s",
				definition,
			)
		}
	}
	for _, selection := range selections.Records() {
		switch selection.Depth() {
		case contract.DepthFullSemantic:
			fullSemanticDefinitions++
		case contract.DepthDeclarationContract:
			declarationContractDefinitions++
		case contract.DepthExternalBoundary:
			externalBoundaryDefinitions++
		case contract.DepthIntrinsic:
			intrinsicDefinitions++
		default:
			return fmt.Errorf(
				"verified inspection has invalid evidence depth %s",
				selection.Depth(),
			)
		}
	}
	if fullSemanticDefinitions+
		declarationContractDefinitions+
		externalBoundaryDefinitions+
		intrinsicDefinitions != definitionCount {
		return fmt.Errorf(
			"verified inspection evidence-depth partition is %d/%d",
			fullSemanticDefinitions+
				declarationContractDefinitions+
				externalBoundaryDefinitions+
				intrinsicDefinitions,
			definitionCount,
		)
	}
	headerOccurrences = inspection.Structure().HeaderOccurrenceCount()
	boundaryEntries = inspection.Structure().BoundaryEntryCount()
	provider := inspection.Structure().ProviderManifestStats()
	projection := inspection.Structure().ProviderProjectionStats()
	executableOccurrences := 0
	for _, region := range executableInventory.Regions() {
		executableOccurrences += len(region.Members())
	}
	hydration := inspection.Hydration()
	if err := print(
		"denominators: closurePackages=%d structuralFiles=%d localAuthorityFiles=%d certifiedAuthorityFiles=%d hydratedPackages=%d localSyntaxFiles=%d localSyntaxBytes=%d checkedViewPackages=%d definitions=%d residentDefinitions=%d selections=%d fullSemanticDefinitions=%d declarationContractDefinitions=%d externalBoundaryDefinitions=%d intrinsicDefinitions=%d headers=%d headerOccurrences=%d boundaries=%d boundaryEntries=%d residentOccurrences=%d executableRegions=%d executableOccurrences=%d selectionFacts=%d providerPackages=%d providerFiles=%d providerDefinitions=%d providerFacts=%d largestProviderShardBytes=%d largestManifestPackageRecords=%d providerShardLoads=%d providerCacheHits=%d maxProviderPackagesResident=%d largestProjectedPackageBytes=%d largestProjectedPackageRecords=%d unknownConstructs=0 unknownDirectives=0\n",
		len(workspace.Packages()),
		len(inspection.SourcePlan().Files()),
		localSourceFiles,
		certifiedSourceFiles,
		hydration.SemanticPackages,
		hydration.LocalFiles,
		hydration.LocalBytes,
		hydration.CheckedPackages,
		definitionCount,
		len(inspection.Structure().ResidentDefinitions()),
		len(selections.Records()),
		fullSemanticDefinitions,
		declarationContractDefinitions,
		externalBoundaryDefinitions,
		intrinsicDefinitions,
		definitionCount,
		headerOccurrences,
		definitionCount,
		boundaryEntries,
		len(inspection.Structure().ResidentOccurrences()),
		len(executableInventory.Regions()),
		executableOccurrences,
		len(inspection.SelectionFacts().Facts()),
		provider.PackageContexts,
		provider.Files,
		provider.Definitions,
		provider.SelectionFacts,
		provider.LargestShardBytes,
		provider.LargestPackageRecords,
		projection.ShardLoads,
		projection.CacheHits,
		projection.MaxResidentPackages,
		projection.LargestPackageBytes,
		projection.LargestPackageRecords,
	); err != nil {
		return err
	}
	for index, record := range provider.LargestShards() {
		if err := print(
			"provider-manifest-tail rank=%d package=%s bytes=%d records=%d\n",
			index+1,
			record.Package,
			record.Bytes,
			record.Records,
		); err != nil {
			return err
		}
	}
	for index, record := range projection.LargestPackages() {
		if err := print(
			"provider-projection-tail rank=%d package=%s bytes=%d records=%d\n",
			index+1,
			record.Package,
			record.Bytes,
			record.Records,
		); err != nil {
			return err
		}
	}
	for index, record := range inspection.Structure().LargestHeaderArtifacts() {
		if err := print(
			"header-tail rank=%d header=%s encodedBytes=%d occurrences=%d\n",
			index+1,
			record.Header,
			record.EncodedBytes,
			record.Occurrences,
		); err != nil {
			return err
		}
	}
	return nil
}
