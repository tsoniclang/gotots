// Hand port B1 — width-specific arithmetic exception: uint32 reinterpret
// and byte extraction are the attributed exception bytes; the call shape
// stays ordinary.
export function hashWrite32(h: xxh3.Hasher, value: GoInt32 | GoUint32): void {
  const v = value >>> 0;
  h.Write(Uint8Array.of(v & 0xff, (v >>> 8) & 0xff, (v >>> 16) & 0xff, (v >>> 24) & 0xff));
}
