package tsgo

import (
	"context"
	"go/importer"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestBoundaryPolicyIsDerivedAndExecutesBothEffects(t *testing.T) {
	projectDirectory, source, project, client := boundaryPolicyProject(
		t,
		boundaryPolicyContract(t),
	)
	exports, err := project.Exports(source)
	if err != nil {
		t.Fatal(err)
	}
	policy, err := project.BoundaryPolicy(
		projectExportByName(t, exports, "WalkDir"),
		projectExportByName(t, exports, "CanonicalBoundaryPolicy"),
		projectExportByName(t, exports, "FromProviderRequest"),
		projectExportByName(t, exports, "InterfaceGuardRequest"),
	)
	if err != nil {
		t.Fatal(err)
	}
	sourcePackage, err := importer.Default().Import("io/fs")
	if err != nil {
		t.Fatal(err)
	}
	sourceCallable, ok := sourcePackage.Scope().Lookup("WalkDir").(*types.Func)
	if !ok {
		t.Fatal("selected Go io/fs.WalkDir is absent")
	}
	sourceSignature, ok := sourceCallable.Type().(*types.Signature)
	if !ok {
		t.Fatal("selected Go io/fs.WalkDir has no signature")
	}
	if policy.Parameter() != sourceSignature.Params().Len() {
		t.Fatalf(
			"policy parameter = %d, Go parameters = %d",
			policy.Parameter(),
			sourceSignature.Params().Len(),
		)
	}
	if !policy.Source().Matches(projectExportByName(t, exports, "SourceWalkDir")) {
		t.Fatal("boundary policy does not name its exact source callable symbol")
	}
	expected := []struct {
		member string
		kind   BoundaryCapabilityKind
		source string
	}{
		{"entryBridge", BoundaryCapabilityFromProvider, "ProviderDirEntry"},
		{"errorBridge", BoundaryCapabilityFromProvider, "ProviderError"},
		{"isReadDirFS", BoundaryCapabilityInterfaceGuard, "ReadDirFSIdentity"},
		{"isReadDirFile", BoundaryCapabilityInterfaceGuard, "ReadDirFileIdentity"},
		{"isStatFS", BoundaryCapabilityInterfaceGuard, "StatFSIdentity"},
	}
	capabilities := policy.Capabilities()
	if len(capabilities) != len(expected) {
		t.Fatalf("capabilities = %#v", capabilities)
	}
	for index, want := range expected {
		actual := capabilities[index]
		if actual.Member() != want.member ||
			actual.Kind() != want.kind ||
			actual.Source().Name() != want.source {
			t.Fatalf("capability %d = %#v, want %#v", index, actual, want)
		}
		if !actual.Source().Matches(projectExportByName(t, exports, want.source)) {
			t.Fatalf("capability %s source does not match its TS symbol", want.member)
		}
		if !slices.Equal(
			actual.Source().ImplementationOwners(),
			[]string{"contract.ts"},
		) {
			t.Fatalf(
				"capability %s owners = %v",
				want.member,
				actual.Source().ImplementationOwners(),
			)
		}
	}
	adapter := boundaryPolicyAdapter(t, capabilities, "")
	printed, err := client.PrintNode(adapter, PrintOptions{})
	if err != nil {
		t.Fatal(err)
	}
	const expectedAdapter = `import { entryBridge, errorBridge, isReadDirFS, isReadDirFile, isStatFS, runProof } from "./contract.js";
const policy = {
    entryBridge: entryBridge,
    errorBridge: errorBridge,
    isReadDirFS: isReadDirFS,
    isReadDirFile: isReadDirFile,
    isStatFS: isStatFS
};
console.log(await runProof(policy));
`
	if printed != expectedAdapter {
		t.Fatalf("generated boundary adapter:\n%s", printed)
	}
	adapterPath := filepath.Join(projectDirectory, "adapter.ts")
	writeProjectFile(t, adapterPath, printed)
	compileBoundaryPolicyProject(t, projectDirectory, adapterPath, true)
	command := exec.Command(
		"node",
		filepath.Join(projectDirectory, "out", "adapter.js"),
	)
	command.Env = append(os.Environ(), "NODE_OPTIONS=--max-old-space-size=512")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("execute boundary policy: %v\n%s", err, output)
	}
	referenceContext, referenceCancel := context.WithTimeout(
		context.Background(),
		time.Minute,
	)
	defer referenceCancel()
	reference := exec.CommandContext(
		referenceContext,
		"go",
		"run",
		filepath.Join("testdata", "boundary-policy", "reference.go"),
	)
	referenceOutput, err := reference.CombinedOutput()
	if err != nil {
		t.Fatalf("execute Go boundary reference: %v\n%s", err, referenceOutput)
	}
	if string(output) != string(referenceOutput) {
		t.Fatalf(
			"boundary policy differential:\nGo:\n%s\nTypeScript:\n%s",
			referenceOutput,
			output,
		)
	}
	for _, forbidden := range []string{
		" as any",
		" as unknown",
		".call(",
		".apply(",
		".bind(",
	} {
		if strings.Contains(printed, forbidden) {
			t.Fatalf("generated adapter contains %q:\n%s", forbidden, printed)
		}
	}
	missing := boundaryPolicyAdapter(t, capabilities, "entryBridge")
	missingText, err := client.PrintNode(missing, PrintOptions{})
	if err != nil {
		t.Fatal(err)
	}
	missingPath := filepath.Join(projectDirectory, "missing.ts")
	writeProjectFile(t, missingPath, missingText)
	compileBoundaryPolicyProject(t, projectDirectory, missingPath, false)
}

func TestBoundaryPolicySemanticMutationsAreCaught(t *testing.T) {
	referenceContext, referenceCancel := context.WithTimeout(
		context.Background(),
		time.Minute,
	)
	defer referenceCancel()
	reference := exec.CommandContext(
		referenceContext,
		"go",
		"run",
		filepath.Join("testdata", "boundary-policy", "reference.go"),
	)
	referenceOutput, err := reference.CombinedOutput()
	if err != nil {
		t.Fatalf("execute Go boundary reference: %v\n%s", err, referenceOutput)
	}
	tests := []struct {
		name         string
		mutate       func(string) string
		strictReject bool
	}{
		{
			name: "synchronous contract narrowed to promise",
			mutate: func(value string) string {
				return strings.Replace(
					value,
					"type Awaitable<Value> = Value | Promise<Value>;",
					"type Awaitable<Value> = Promise<Value>;",
					1,
				)
			},
			strictReject: true,
		},
		{
			name: "bridge discards provider error",
			mutate: func(value string) string {
				return strings.Replace(
					value,
					"return value === undefined ? undefined : new CanonicalFailure(value, true);",
					"return undefined;",
					1,
				)
			},
		},
		{
			name: "optional interface guard lies",
			mutate: func(value string) string {
				return strings.Replace(
					value,
					"(value): value is SyncFS => value instanceof SyncFS;",
					"(_value): _value is SyncFS => false;",
					1,
				)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			directory, source, project, client := boundaryPolicyProject(
				t,
				test.mutate(boundaryPolicyContract(t)),
			)
			exports, err := project.Exports(source)
			if err != nil {
				t.Fatal(err)
			}
			policy, err := project.BoundaryPolicy(
				projectExportByName(t, exports, "WalkDir"),
				projectExportByName(t, exports, "CanonicalBoundaryPolicy"),
				projectExportByName(t, exports, "FromProviderRequest"),
				projectExportByName(t, exports, "InterfaceGuardRequest"),
			)
			if err != nil {
				t.Fatal(err)
			}
			adapter := boundaryPolicyAdapter(t, policy.Capabilities(), "")
			printed, err := client.PrintNode(adapter, PrintOptions{})
			if err != nil {
				t.Fatal(err)
			}
			adapterPath := filepath.Join(directory, "adapter.ts")
			writeProjectFile(t, adapterPath, printed)
			compileBoundaryPolicyProject(
				t,
				directory,
				adapterPath,
				!test.strictReject,
			)
			if test.strictReject {
				return
			}
			command := exec.Command(
				"node",
				filepath.Join(directory, "out", "adapter.js"),
			)
			command.Env = append(
				os.Environ(),
				"NODE_OPTIONS=--max-old-space-size=512",
			)
			output, err := command.CombinedOutput()
			if err == nil && string(output) == string(referenceOutput) {
				t.Fatalf("semantic mutation preserved the Go differential:\n%s", output)
			}
		})
	}
}

func TestBoundaryPolicyRejectsStructuralSpoofs(t *testing.T) {
	source := boundaryPolicyContract(t)
	tests := []struct {
		name     string
		mutate   func(string) string
		contains string
	}{
		{
			name: "policy marker spelling",
			mutate: func(value string) string {
				return strings.Replace(
					value,
					"> extends CanonicalBoundaryPolicy<typeof SourceWalkDir> {",
					"> {\n  readonly $go$canonicalBoundarySource?: typeof SourceWalkDir;",
					1,
				)
			},
			contains: "does not inherit the canonical policy marker",
		},
		{
			name: "capability marker spelling",
			mutate: func(value string) string {
				return strings.Replace(
					value,
					"readonly errorBridge: FromProviderRequest<ProviderError, Failure>;",
					"readonly errorBridge: {\n"+
						"    readonly $go$fromProviderSource?: ProviderError;\n"+
						"    $from(value: ProviderError | undefined): Failure | undefined;\n"+
						"  };",
					1,
				)
			},
			contains: "has no recognized capability marker",
		},
		{
			name: "capability wrapper",
			mutate: func(value string) string {
				value = strings.Replace(
					value,
					"export interface WalkPolicy<",
					"interface WrappedErrorBridge<Failure extends BoundaryFailure>\n"+
						"  extends FromProviderRequest<ProviderError, Failure> {}\n\n"+
						"export interface WalkPolicy<",
					1,
				)
				return strings.Replace(
					value,
					"readonly errorBridge: FromProviderRequest<ProviderError, Failure>;",
					"readonly errorBridge: WrappedErrorBridge<Failure>;",
					1,
				)
			},
			contains: "is not a direct marker instantiation",
		},
		{
			name: "unclassified policy field",
			mutate: func(value string) string {
				return strings.Replace(
					value,
					"> extends CanonicalBoundaryPolicy<typeof SourceWalkDir> {",
					"> extends CanonicalBoundaryPolicy<typeof SourceWalkDir> {\n"+
						"  readonly unrelated: { readonly value: number };",
					1,
				)
			},
			contains: "has no recognized capability marker",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, path, project, _ := boundaryPolicyProject(t, test.mutate(source))
			exports, err := project.Exports(path)
			if err != nil {
				t.Fatal(err)
			}
			_, err = project.BoundaryPolicy(
				projectExportByName(t, exports, "WalkDir"),
				projectExportByName(t, exports, "CanonicalBoundaryPolicy"),
				projectExportByName(t, exports, "FromProviderRequest"),
				projectExportByName(t, exports, "InterfaceGuardRequest"),
			)
			if err == nil || !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("error = %v, want %q", err, test.contains)
			}
		})
	}
}

func boundaryPolicyProject(
	t *testing.T,
	contract string,
) (string, string, *ProjectInspection, *Client) {
	t.Helper()
	directory := t.TempDir()
	writeProjectFile(t, filepath.Join(directory, "package.json"), `{"type":"module"}`)
	writeProjectFile(t, filepath.Join(directory, "tsconfig.json"), `{
  "compilerOptions": {
    "target": "ES2022",
    "module": "NodeNext",
    "moduleResolution": "NodeNext",
    "strict": true,
    "noUncheckedIndexedAccess": true,
    "outDir": "out"
  },
  "include": ["*.ts"]
}
`)
	source := filepath.Join(directory, "contract.ts")
	writeProjectFile(t, source, contract)
	client, err := StartClient(repositoryRoot(), directory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close client: %v", err)
		}
	})
	project, err := client.OpenProject(filepath.Join(directory, "tsconfig.json"))
	if err != nil {
		t.Fatal(err)
	}
	return directory, source, project, client
}

func boundaryPolicyContract(t *testing.T) string {
	t.Helper()
	payload, err := os.ReadFile(filepath.Join(
		"testdata",
		"boundary-policy",
		"contract.ts",
	))
	if err != nil {
		t.Fatal(err)
	}
	return string(payload)
}

func boundaryPolicyAdapter(
	t *testing.T,
	capabilities []BoundaryCapability,
	omit string,
) SourceFile {
	t.Helper()
	factory := NewFactory()
	imports := make([]ImportSpecifier, 0, len(capabilities)+1)
	properties := make([]ObjectLiteralElementLike, 0, len(capabilities))
	for _, capability := range capabilities {
		if capability.Member() == omit {
			continue
		}
		name := factory.Identifier(capability.Member())
		imports = append(imports, factory.ImportSpecifier(false, nil, name))
		properties = append(
			properties,
			factory.PropertyAssignment(
				nil,
				name,
				nil,
				factory.TypeQueryNode(name, nil),
				name,
			),
		)
	}
	imports = append(imports, factory.ImportSpecifier(
		false,
		nil,
		factory.Identifier("runProof"),
	))
	importStatement := factory.ImportDeclaration(
		nil,
		factory.ImportClause(
			ImportPhaseModifierSyntaxKind(0),
			nil,
			factory.NamedImports(imports),
		),
		factory.StringLiteral("./contract.js", TokenFlagsNone),
		nil,
	)
	policyName := factory.Identifier("policy")
	policyStatement := factory.VariableStatement(
		nil,
		factory.VariableDeclarationList(
			[]VariableDeclaration{factory.VariableDeclaration(
				policyName,
				nil,
				nil,
				factory.ObjectLiteralExpression(properties, true),
			)},
			NodeFlagsConst,
		),
	)
	proof := factory.AwaitExpression(factory.CallExpression(
		factory.Identifier("runProof"),
		nil,
		nil,
		[]Expression{policyName},
		NodeFlagsNone,
	))
	printStatement := factory.ExpressionStatement(factory.CallExpression(
		factory.PropertyAccessExpression(
			factory.Identifier("console"),
			nil,
			factory.Identifier("log"),
			NodeFlagsNone,
		),
		nil,
		nil,
		[]Expression{proof},
		NodeFlagsNone,
	))
	path, err := NewPath("/adapter.ts")
	if err != nil {
		t.Fatal(err)
	}
	return factory.SourceFile(
		[]Statement{importStatement, policyStatement, printStatement},
		factory.EndOfFile(),
		SourceFileData{
			FileName:        path,
			Path:            path,
			LanguageVariant: LanguageVariantStandard,
			ScriptKind:      ScriptKindTS,
		},
	)
}

func compileBoundaryPolicyProject(
	t *testing.T,
	directory string,
	entry string,
	wantSuccess bool,
) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	err := Compile(
		ctx,
		repositoryRoot(),
		directory,
		[]string{
			"--ignoreConfig",
			"--target", "es2022",
			"--module", "nodenext",
			"--moduleResolution", "nodenext",
			"--strict",
			"--noUncheckedIndexedAccess",
			"--outDir", filepath.Join(directory, "out"),
			entry,
		},
	)
	if wantSuccess && err != nil {
		t.Fatal(err)
	}
	if !wantSuccess && err == nil {
		t.Fatal("strict TS-Go accepted an incomplete generated policy")
	}
}
