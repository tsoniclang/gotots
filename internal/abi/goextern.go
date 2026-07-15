package abi

// goexternSource is the external-contract registry: every external
// package-level function call routes through goExternalCall by canonical
// identity, and fails closed with a stable panic unless the emulation
// layer registered behavior. Registration is explicit and identity-
// keyed — no probing, no fallbacks.
const goexternSource = `// External-contract registry: typed stubs delegate here; the emulation
// layer registers behavior per canonical identity. An unregistered
// contract fails closed at the call.
import { GoPanic } from "./gopanic.js";

const registry = new Map<string, Function>();

// goExternalRegister binds one external contract's behavior. The id is
// the canonical Go identity: "<package path>.<Name>".
export function goExternalRegister(id: string, fn: Function): void {
  registry.set(id, fn);
}

export function goExternalCall(id: string, args: unknown[]): unknown {
  const fn = registry.get(id);
  if (fn === undefined) {
    throw new GoPanic("GOTOTS_EXTERNAL_UNIMPLEMENTED: " + id);
  }
  return fn(...args);
}

// An external value: an opaque handle whose behavior the emulation
// layer supplies per contract. The brand keeps distinct external types
// from cross-assigning inside generated code.
export type GoExtern<T extends string> = { readonly goExternType$?: T };
`
