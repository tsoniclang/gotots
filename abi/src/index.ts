import { createHash } from "node:crypto";
import { createSourceSemanticsVirtualModuleProvider } from "@tsonic/source-core/extension";
import type { TsonicDataLayoutRegistration } from "@tsonic/source-core";
import type { TargetSourceCompilerContributions, TsonicTargetCapabilityPlugin } from "@tsonic/target-api/provider";
import type { CompilerExtension, ProviderExportDeclaration } from "@tsonic/tsts";

export const goAbiModule = "@gotots/abi/layout.js";
const providerId = "gotots.source-abi";
const providerVersion = "1";

const layouts = Object.freeze([
  Object.freeze({ name: "little32", byteOrder: "little", addressWidth: 32 } as const),
  Object.freeze({ name: "little64", byteOrder: "little", addressWidth: 64 } as const),
  Object.freeze({ name: "big32", byteOrder: "big", addressWidth: 32 } as const),
  Object.freeze({ name: "big64", byteOrder: "big", addressWidth: 64 } as const),
]);

export function goAbiCompilerContributions(): TargetSourceCompilerContributions {
  const declarations: readonly ProviderExportDeclaration[] = layouts.map((layout) => ({
    id: layout.name,
    name: layout.name,
    kind: "value",
    type: { kind: "provider-ref", moduleSpecifier: "@tsonic/core/types.js", exportName: "DataLayout" },
  }));
  const provider = createSourceSemanticsVirtualModuleProvider({
    id: providerId,
    version: providerVersion,
    displayName: "Selected Go source ABI",
    virtualDirectory: "gotots-source-abi",
    modules: [{ moduleSpecifier: goAbiModule, exports: [] }],
    evidenceMessage: "Explicit source byte order and address width; independent of the target host",
    importsForModule: () => [{
      moduleSpecifier: "@tsonic/core/types.js",
      namedImports: [{ exportedName: "DataLayout", kind: "type" }],
      typeOnly: true,
    }],
    exportsForModule: () => declarations,
  });
  const extension: CompilerExtension = {
    identity: { id: providerId, version: providerVersion },
    initialize(context) { context.registerSourceDeclarationProvider(provider); },
  };
  const dataLayouts: readonly TsonicDataLayoutRegistration[] = layouts.map((layout) => Object.freeze({
    providerDeclaration: Object.freeze({
      providerId,
      providerVersion,
      providerModuleId: goAbiModule,
      moduleSpecifier: goAbiModule,
      exportId: layout.name,
    }),
    descriptor: Object.freeze({
      fingerprint: createHash("sha256").update(`${providerId}:${providerVersion}:${layout.byteOrder}:${layout.addressWidth}`).digest("hex"),
      byteOrder: layout.byteOrder,
      addressWidth: layout.addressWidth,
    }),
  }));
  return Object.freeze({ extensions: Object.freeze([extension]), dataLayouts: Object.freeze(dataLayouts) });
}

export function createGoAbiCapability(targetId: string): TsonicTargetCapabilityPlugin {
  if (targetId.trim().length === 0) throw new Error("Go ABI capability requires an explicit target identity");
  return Object.freeze({
    kind: "target-capability",
    id: providerId,
    targetId,
    displayName: "Selected Go source ABI",
    moduleOwnership: Object.freeze([{ specifierPrefix: goAbiModule, providerId }]),
    sourceCompilerContributions: goAbiCompilerContributions,
  });
}
