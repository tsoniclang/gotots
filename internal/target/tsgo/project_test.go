package tsgo

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestProjectExportsUseTSGoSymbolsAndDeclarationOwners(t *testing.T) {
	projectDirectory := t.TempDir()
	writeProjectFile(t, filepath.Join(projectDirectory, "tsconfig.json"), `{
  "compilerOptions": {
    "target": "ES2022",
    "module": "NodeNext",
    "moduleResolution": "NodeNext",
    "strict": true
  },
  "include": ["*.ts"]
}
`)
	implementationPath := filepath.Join(projectDirectory, "implementation.ts")
	writeProjectFile(t, implementationPath, `export function Value(input: string): number {
  return input.length;
}

export function Maybe(input: number | undefined): number {
  return input ?? 0;
}

export class Box<Left, Right> {
  static Make(value: number): Box {
    return new Box(value);
  }

  private constructor(readonly value: number) {}
}

export class MemberAccess {
  #ecmaPrivate = 1;
  private tsPrivate = 2;
  protected inherited = 3;
  public visible = 4;

  constructor(private readonly parameterPrivate = 5) {}

  method(): number {
    return this.#ecmaPrivate + this.tsPrivate + this.inherited + this.visible +
      this.parameterPrivate;
  }
}

export interface Shape {
  Get(): number;
}

export type AsyncCallable = (() => Promise<void>) | undefined;
export type SyncCallable = (() => number) | undefined;
export type NumericAlias = number;
export type AwaitableCallable = () => void | Promise<void>;
export type AwaitableUnionCallable = () =>
  string | undefined | Promise<string | undefined>;
export type InvalidEffectCallable = () => string | Promise<void>;
export type GenericAsyncCallable<
  Value extends (() => Promise<void>) | undefined =
    (() => Promise<void>) | undefined,
> = Value;

export function Invoke(
  direct: (value: number) => boolean,
  cooperative: (value: number) => boolean | Promise<boolean>,
): Promise<void> {
  void direct;
  void cooperative;
  return Promise.resolve();
}

export interface SupportContract extends ReadonlyArray<object> {}
export function ConsumeSupport(contract: SupportContract): void {
  void contract;
}

export interface CapabilitySource {
  Base(): string;
}

export interface CapabilityTarget extends CapabilitySource {
  Extra(): number;
}

export function CapabilityView(
  value: CapabilitySource,
): CapabilityTarget | undefined {
  return value.Base() === "target" ? value as CapabilityTarget : undefined;
}

export function RequiredCapabilityView(
  value: CapabilitySource,
): CapabilityTarget {
  return value as CapabilityTarget;
}

export class Effects {
  static Async(): Promise<void> {
    return Promise.resolve();
  }

  static Sync(): void {}
}

const Hidden = 1;
void Hidden;
`)
	entryPath := filepath.Join(projectDirectory, "entry.ts")
	writeProjectFile(t, entryPath, `export {
  AsyncCallable,
  AwaitableCallable,
  AwaitableUnionCallable,
  Box,
  Effects,
  GenericAsyncCallable,
  InvalidEffectCallable,
  Invoke,
  MemberAccess,
  Maybe,
  NumericAlias,
  Shape,
  SupportContract,
  CapabilitySource,
  CapabilityTarget,
  CapabilityView,
  RequiredCapabilityView,
  ConsumeSupport,
  SyncCallable,
  Value,
} from "./implementation.js";
export const count: number = 1;
export const state: { Count: number } = { Count: 0 };
`)
	markerPath := filepath.Join(projectDirectory, "effect-marker.ts")
	writeProjectFile(t, markerPath, `export type AsyncEffectMarker = () => Promise<never>;
`)
	writeProjectFile(t, filepath.Join(projectDirectory, "facets.ts"), `type AsyncABI = (() => Promise<void>) | undefined;

export type AsyncFacet<Value = AsyncABI> = Value;
export type ConstrainedFacet<Value extends AsyncABI = AsyncABI> = Value;
export type MissingDefaultFacet<Value> = Value;
export type NonCallableDefaultFacet<Value = number> = Value;
export type WrappedFacet<Value = AsyncABI> = [Value][0];
`)
	writeProjectFile(t, filepath.Join(projectDirectory, "facet-entry.ts"), `export { AsyncFacet } from "./facets.js";
`)
	renamedPath := filepath.Join(projectDirectory, "renamed.ts")
	writeProjectFile(t, renamedPath, `export { Value as Other } from "./implementation.js";
`)

	client, err := StartClientWithTool(selectedTool(t), projectDirectory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close client: %v", err)
		}
	})
	project, err := client.OpenProject(
		filepath.Join(projectDirectory, "tsconfig.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	exports, err := project.Exports(entryPath)
	if err != nil {
		t.Fatal(err)
	}
	valueExport := projectExportByName(t, exports, "Value")
	parameterType, err := project.CallableParameterTypeString(valueExport, 0)
	if err != nil || parameterType != "string" {
		t.Fatalf("Value parameter type = %q, %v", parameterType, err)
	}
	primitive, ok, err := project.CallableParameterPrimitive(valueExport, 0)
	if err != nil || !ok || primitive.Kind() != ProjectPrimitiveString ||
		primitive.Optional() {
		t.Fatalf("Value parameter primitive = %#v, %v, %v", primitive, ok, err)
	}
	maybeExport := projectExportByName(t, exports, "Maybe")
	primitive, ok, err = project.CallableParameterPrimitive(maybeExport, 0)
	if err != nil || !ok || primitive.Kind() != ProjectPrimitiveNumber ||
		!primitive.Optional() {
		t.Fatalf("Maybe parameter primitive = %#v, %v, %v", primitive, ok, err)
	}
	returnType, err := project.CallableReturnTypeString(valueExport)
	if err != nil || returnType != "number" {
		t.Fatalf("Value return type = %q, %v", returnType, err)
	}
	names := make([]string, len(exports))
	for index, selected := range exports {
		names[index] = selected.Name()
		if selected.Fingerprint() == "" {
			t.Fatalf("%s has no canonical target fingerprint", selected.Name())
		}
		if selected.TypeString() == "" ||
			selected.Flags() == 0 ||
			len(selected.Declarations()) == 0 {
			t.Fatalf("invalid export %#v", selected)
		}
		if selected.Name() == "count" || selected.Name() == "state" {
			if !slices.Equal(
				selected.Declarations(),
				[]string{filepath.ToSlash(entryPath)},
			) {
				t.Fatalf(
					"state declarations = %v",
					selected.Declarations(),
				)
			}
			continue
		}
		switch selected.Name() {
		case "Box":
			if !slices.Equal(
				projectMemberNames(selected.ValueMembers()),
				[]string{"Make"},
			) ||
				!slices.Equal(
					projectMemberNames(selected.TypeMembers()),
					[]string{"value"},
				) {
				t.Fatalf("Box members = %#v", selected)
			}
		case "Shape":
			if !slices.Equal(
				projectMemberNames(selected.TypeMembers()),
				[]string{"Get"},
			) {
				t.Fatalf("Shape members = %#v", selected)
			}
		case "AsyncCallable", "AwaitableCallable", "AwaitableUnionCallable", "CapabilitySource", "CapabilityTarget", "CapabilityView", "ConsumeSupport", "Effects", "GenericAsyncCallable", "InvalidEffectCallable", "Invoke", "NumericAlias", "RequiredCapabilityView", "SupportContract", "SyncCallable":
			continue
		}
		if !slices.Equal(
			selected.Declarations(),
			[]string{filepath.ToSlash(implementationPath)},
		) {
			t.Fatalf(
				"%s declarations = %v",
				selected.Name(),
				selected.Declarations(),
			)
		}
	}
	numeric := projectExportByName(t, exports, "NumericAlias")
	if numeric.TypeString() != "any" || numeric.DeclaredTypeString() != "number" {
		t.Fatalf(
			"numeric alias types = %q/%q, want any/number",
			numeric.TypeString(),
			numeric.DeclaredTypeString(),
		)
	}
	if !slices.Equal(
		names,
		[]string{
			"AsyncCallable",
			"AwaitableCallable",
			"AwaitableUnionCallable",
			"Box",
			"CapabilitySource",
			"CapabilityTarget",
			"CapabilityView",
			"ConsumeSupport",
			"Effects",
			"GenericAsyncCallable",
			"InvalidEffectCallable",
			"Invoke",
			"Maybe",
			"MemberAccess",
			"NumericAlias",
			"RequiredCapabilityView",
			"Shape",
			"SupportContract",
			"SyncCallable",
			"Value",
			"count",
			"state",
		},
	) {
		t.Fatalf("exports = %v", names)
	}
	boxIndex := slices.IndexFunc(exports, func(selected ProjectExport) bool {
		return selected.Name() == "Box"
	})
	if boxIndex < 0 {
		t.Fatal("Box export is absent")
	}
	if exports[boxIndex].TypeParameterCount() != 2 {
		t.Fatalf("Box type parameters = %d", exports[boxIndex].TypeParameterCount())
	}
	makeMember, ok := exports[boxIndex].ValueMember("Make")
	if !ok || makeMember.Fingerprint() == "" || !makeMember.Visible() {
		t.Fatalf("Box.Make member = %#v, %t", makeMember, ok)
	}
	if _, ok := exports[boxIndex].ValueMember("missing"); ok {
		t.Fatal("missing member resolved")
	}
	memberAccess := projectExportByName(t, exports, "MemberAccess")
	visibleMembers := make([]string, 0)
	nonPublicMembers := 0
	for _, member := range memberAccess.TypeMembers() {
		if member.Visible() {
			visibleMembers = append(visibleMembers, member.Name())
			continue
		}
		nonPublicMembers++
	}
	if !slices.Equal(visibleMembers, []string{"method", "visible"}) ||
		nonPublicMembers != 4 {
		t.Fatalf(
			"MemberAccess visibility = public %v, non-public %d; members %#v",
			visibleMembers,
			nonPublicMembers,
			memberAccess.TypeMembers(),
		)
	}
	markerExports, err := project.Exports(markerPath)
	if err != nil {
		t.Fatal(err)
	}
	marker := projectExportByName(t, markerExports, "AsyncEffectMarker")
	invoke := projectExportByName(t, exports, "Invoke")
	directParameterEffect, err := project.CallableParameterEffect(invoke, 0, marker)
	if err != nil || directParameterEffect != CallableEffectSynchronous {
		t.Fatalf("direct parameter effect = %v, %v", directParameterEffect, err)
	}
	cooperativeParameterEffect, err := project.CallableParameterEffect(invoke, 1, marker)
	if err != nil || cooperativeParameterEffect != CallableEffectAwaitable {
		t.Fatalf(
			"cooperative parameter effect = %v, %v",
			cooperativeParameterEffect,
			err,
		)
	}
	consumeSupport := projectExportByName(t, exports, "ConsumeSupport")
	supportContract := projectExportByName(t, exports, "SupportContract")
	supportIdentity, err := project.CallableParameterTypeIdentity(consumeSupport, 0)
	if err != nil || !supportIdentity.Matches(supportContract) {
		t.Fatalf("support parameter identity = %#v, %v", supportIdentity, err)
	}
	capabilitySource := projectExportByName(t, exports, "CapabilitySource")
	capabilityTarget := projectExportByName(t, exports, "CapabilityTarget")
	capabilityView := projectExportByName(t, exports, "CapabilityView")
	capabilityParameter, capabilityResult, err :=
		project.CallableOptionalViewTypes(capabilityView)
	if err != nil || !capabilityParameter.Matches(capabilitySource) ||
		!capabilityResult.Matches(capabilityTarget) {
		t.Fatalf(
			"capability view types = %#v, %#v, %v",
			capabilityParameter,
			capabilityResult,
			err,
		)
	}
	requiredCapabilityView := projectExportByName(
		t,
		exports,
		"RequiredCapabilityView",
	)
	if _, _, err := project.CallableOptionalViewTypes(
		requiredCapabilityView,
	); err == nil {
		t.Fatal("required capability result was accepted as optional")
	}
	async := projectExportByName(t, exports, "AsyncCallable")
	awaitable := projectExportByName(t, exports, "AwaitableCallable")
	awaitableUnion := projectExportByName(t, exports, "AwaitableUnionCallable")
	genericAsync := projectExportByName(t, exports, "GenericAsyncCallable")
	sync := projectExportByName(t, exports, "SyncCallable")
	asyncEffect, err := project.CallableEffect(async, marker)
	if err != nil || asyncEffect != CallableEffectAsynchronous {
		t.Fatalf("async effect = %v, %v", asyncEffect, err)
	}
	awaitableEffect, err := project.CallableEffect(awaitable, marker)
	if err != nil || awaitableEffect != CallableEffectAwaitable {
		t.Fatalf("awaitable effect = %v, %v", awaitableEffect, err)
	}
	awaitableEffect, err = project.CallableEffect(awaitableUnion, marker)
	if err != nil || awaitableEffect != CallableEffectAwaitable {
		t.Fatalf("awaitable union effect = %v, %v", awaitableEffect, err)
	}
	asyncEffect, err = project.CallableEffect(genericAsync, marker)
	if err != nil || asyncEffect != CallableEffectAsynchronous {
		t.Fatalf("generic async effect = %v, %v", asyncEffect, err)
	}
	syncEffect, err := project.CallableEffect(sync, marker)
	if err != nil || syncEffect != CallableEffectSynchronous {
		t.Fatalf("sync effect = %v, %v", syncEffect, err)
	}
	effects := projectExportByName(t, exports, "Effects")
	asyncMember, ok := effects.ValueMember("Async")
	if !ok {
		t.Fatal("Effects.Async is absent")
	}
	asyncEffect, err = project.CallableEffect(asyncMember, marker)
	if err != nil || asyncEffect != CallableEffectAsynchronous {
		t.Fatalf("async member effect = %v, %v", asyncEffect, err)
	}
	invalid := projectExportByName(t, exports, "InvalidEffectCallable")
	if _, err := project.CallableEffect(invalid, marker); err == nil {
		t.Fatal("union effect return was accepted")
	}
	renamed, err := project.Exports(renamedPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(renamed) != 1 ||
		renamed[0].Name() != "Other" ||
		renamed[0].TypeString() != "(input: string) => number" {
		t.Fatalf("renamed exports = %#v", renamed)
	}
}

func projectExportByName(
	t *testing.T,
	exports []ProjectExport,
	name string,
) ProjectExport {
	t.Helper()
	index := slices.IndexFunc(exports, func(selected ProjectExport) bool {
		return selected.Name() == name
	})
	if index < 0 {
		t.Fatalf("export %s is absent", name)
	}
	return exports[index]
}

func projectMemberNames(members []ProjectMember) []string {
	names := make([]string, len(members))
	for index, member := range members {
		names[index] = member.Name()
	}
	return names
}

func TestDeclarationPathRejectsMalformedHandles(t *testing.T) {
	for _, handle := range []string{"", "one", "one.two", "one.two."} {
		if path, ok := declarationPath(handle); ok || path != "" {
			t.Fatalf("declarationPath(%q) = %q, %t", handle, path, ok)
		}
	}
	if path, ok := declarationPath("12.34./project/source.ts"); !ok || path != "/project/source.ts" {
		t.Fatalf("declaration path = %q, %t", path, ok)
	}
}

func TestProjectFingerprintCanonicalizesComputedUniqueSymbolKeys(t *testing.T) {
	directory := t.TempDir()
	writeProjectFile(t, filepath.Join(directory, "tsconfig.json"), `{
  "compilerOptions": {
    "target": "ES2022",
    "module": "NodeNext",
    "moduleResolution": "NodeNext",
    "strict": true
  },
  "include": ["*.ts"]
}
`)
	implementation := filepath.Join(directory, "implementation.ts")
	writeProjectFile(t, implementation, `
export const ProfileNameKey: unique symbol = Symbol("profile-name");
export class Profile {
  readonly [ProfileNameKey]: string = "profile";
}
`)
	client, err := StartClientWithTool(selectedTool(t), directory)
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
	exports, err := project.Exports(implementation)
	if err != nil {
		t.Fatal(err)
	}
	profile := projectExportByName(t, exports, "Profile")
	members := profile.TypeMembers()
	if len(members) != 1 {
		t.Fatalf("Profile members = %#v", members)
	}
	baseline := members[0]
	mutated, err := project.projectMember(
		implementation,
		"computed-key display mutation",
		symbolResponse{
			ID:           baseline.symbolID,
			Name:         baseline.name + "@allocation-shift",
			Flags:        baseline.flags,
			Declarations: baseline.handles,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if baseline.Name() == mutated.Name() {
		t.Fatal("display-spelling mutation did not change the member name")
	}
	if baseline.Fingerprint() != mutated.Fingerprint() {
		t.Fatalf(
			"computed unique-symbol fingerprint drifted: %s != %s",
			baseline.Fingerprint(),
			mutated.Fingerprint(),
		)
	}
}

func writeProjectFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
