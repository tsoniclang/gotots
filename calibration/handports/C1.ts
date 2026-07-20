// Hand port C1 — OrderedMap.Set, primitive-key binding family
// (pass-specialized planning exercise). The JS Map representation owns
// insertion order and allocation, so the Go keys-slice bookkeeping and
// lazy make() belong to the selected representation, not the body:
// Map.set keeps an existing key's original position, exactly matching
// the Go append-only-when-absent bookkeeping.
set(key: K, value: V): void {
  this.mp.set(key, value);
}
