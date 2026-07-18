package abi

// goexternSource carries the external-contract support types: the
// branded opaque handle and the fail-closed throw every unresolved stub
// body uses. There is no registry, lookup, or runtime registration —
// stubs are typed static exports, and assembly replaces a stub module
// with its reviewed implementation module at the same path.
const goexternSource = `// External-contract support: the branded opaque handle and the
// fail-closed unimplemented panic. Stubs are static typed exports;
// assembly replaces stub modules with implementation modules.
import { GoPanic } from "./gopanic.js";

// An external value: an opaque handle whose behavior the assembled
// implementation supplies per contract. The brand keeps distinct
// external types from cross-assigning inside generated code.
export type GoExtern<T extends string> = { readonly goExternType$?: T };

// goExternalUnimplemented is the body of every unresolved stub: the
// contract exists and is typed, but no implementation is assembled.
export function goExternalUnimplemented(id: string): never {
  throw new GoPanic("GOTOTS_EXTERNAL_UNIMPLEMENTED: " + id);
}
`
