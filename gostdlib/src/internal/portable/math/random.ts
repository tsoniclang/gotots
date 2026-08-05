import type { uint64 } from "@gotots/gostdlib/internal/scalars.js";

const words = new Uint32Array(2);

export function Uint64(): uint64 {
  globalThis.crypto.getRandomValues(words);
  return BigInt(words[0] ?? 0) * 4_294_967_296n + BigInt(words[1] ?? 0);
}
