// Hand port C3 — core.Map, parametric slice mapping (planning exercise:
// ADR-0007 forms 1-2 suffice; no specialization is required). The Go
// nil-in/nil-out contract survives as undefined-in/undefined-out; the
// make-plus-range loop is Array.prototype.map.
function Map<T, U>(slice: readonly T[] | undefined, f: (value: T) => U): U[] | undefined {
  if (slice === undefined) {
    return undefined;
  }
  return slice.map(f);
}
