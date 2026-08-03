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
export type InvalidEffectCallable = () => Promise<void> | void;
export type GenericAsyncCallable<
  Value extends (() => Promise<void>) | undefined =
    (() => Promise<void>) | undefined,
> = Value;

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
  Box,
  Effects,
  GenericAsyncCallable,
  InvalidEffectCallable,
  MemberAccess,
  Shape,
  SyncCallable,
  Value,
} from "./implementation.js";
export const count: number = 1;
export const state: { Count: number } = { Count: 0 };
`)
	markerPath := filepath.Join(projectDirectory, "effect-marker.ts")
	writeProjectFile(t, markerPath, `export type AsyncEffectMarker = () => Promise<never>;
`)
	facetPath := filepath.Join(projectDirectory, "facets.ts")
	writeProjectFile(t, facetPath, `type AsyncABI = (() => Promise<void>) | undefined;

export type AsyncFacet<Value = AsyncABI> = Value;
export type ConstrainedFacet<Value extends AsyncABI = AsyncABI> = Value;
export type MissingDefaultFacet<Value> = Value;
export type NonCallableDefaultFacet<Value = number> = Value;
export type WrappedFacet<Value = AsyncABI> = [Value][0];
`)
	facetEntryPath := filepath.Join(projectDirectory, "facet-entry.ts")
	writeProjectFile(t, facetEntryPath, `export { AsyncFacet } from "./facets.js";
`)
	renamedPath := filepath.Join(projectDirectory, "renamed.ts")
	writeProjectFile(t, renamedPath, `export { Value as Other } from "./implementation.js";
`)

	client, err := StartClient(repositoryRoot(), projectDirectory)
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
		case "AsyncCallable", "Effects", "GenericAsyncCallable", "InvalidEffectCallable", "SyncCallable":
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
	if !slices.Equal(
		names,
		[]string{
			"AsyncCallable",
			"Box",
			"Effects",
			"GenericAsyncCallable",
			"InvalidEffectCallable",
			"MemberAccess",
			"Shape",
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
	async := projectExportByName(t, exports, "AsyncCallable")
	genericAsync := projectExportByName(t, exports, "GenericAsyncCallable")
	sync := projectExportByName(t, exports, "SyncCallable")
	asyncEffect, err := project.CallableEffect(async, marker)
	if err != nil || asyncEffect != CallableEffectAsynchronous {
		t.Fatalf("async effect = %v, %v", asyncEffect, err)
	}
	asyncEffect, err = project.CallableEffect(genericAsync, marker)
	if err != nil || asyncEffect != CallableEffectAsynchronous {
		t.Fatalf("generic async effect = %v, %v", asyncEffect, err)
	}
	facetExports, err := project.Exports(facetPath)
	if err != nil {
		t.Fatal(err)
	}
	asyncFacet := projectExportByName(t, facetExports, "AsyncFacet")
	asyncEffect, err = project.CallableValueFacetEffect(asyncFacet, marker)
	if err != nil || asyncEffect != CallableEffectAsynchronous {
		t.Fatalf("async facet effect = %v, %v", asyncEffect, err)
	}
	reexportedFacets, err := project.Exports(facetEntryPath)
	if err != nil {
		t.Fatal(err)
	}
	reexportedFacet := projectExportByName(t, reexportedFacets, "AsyncFacet")
	asyncEffect, err = project.CallableValueFacetEffect(reexportedFacet, marker)
	if err != nil || asyncEffect != CallableEffectAsynchronous {
		t.Fatalf("re-exported async facet effect = %v, %v", asyncEffect, err)
	}
	for _, name := range []string{
		"ConstrainedFacet",
		"MissingDefaultFacet",
		"NonCallableDefaultFacet",
		"WrappedFacet",
	} {
		selected := projectExportByName(t, facetExports, name)
		if _, err := project.CallableValueFacetEffect(selected, marker); err == nil {
			t.Fatalf("malformed callable facet %s was accepted", name)
		}
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
	fingerprint := func(prefix string) (string, string) {
		t.Helper()
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
		writeProjectFile(t, implementation, prefix+`
export const ProfileNameKey: unique symbol = Symbol("profile-name");
export class Profile {
  readonly [ProfileNameKey]: string = "profile";
}
`)
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
		exports, err := project.Exports(implementation)
		if err != nil {
			t.Fatal(err)
		}
		profile := projectExportByName(t, exports, "Profile")
		members := profile.TypeMembers()
		if len(members) != 1 {
			t.Fatalf("Profile members = %#v", members)
		}
		return profile.Fingerprint(), members[0].Name()
	}

	baselineFingerprint, baselineDisplay := fingerprint("")
	shiftedFingerprint, shiftedDisplay := fingerprint(
		`export const EarlierKey: unique symbol = Symbol("earlier");
export class Earlier {
  readonly [EarlierKey]: string = "earlier";
}
`,
	)
	if baselineDisplay == shiftedDisplay {
		t.Fatalf(
			"unique-symbol allocation-order control did not change display spelling %q",
			baselineDisplay,
		)
	}
	if baselineFingerprint != shiftedFingerprint {
		t.Fatalf(
			"computed unique-symbol fingerprint drifted: %s != %s",
			baselineFingerprint,
			shiftedFingerprint,
		)
	}
}

func writeProjectFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
