// Hand port B6 — structural-map-key exception at struct-key
// instantiations: the map representation owns key semantics; callers
// pass only source arguments.
Set(key: K, value: V): void {
  if (this.mp === undefined) {
    this.mp = new GoStructuralMap<K, V>(this.keyPlan);
  }
  if (!this.mp.has(key)) {
    this.keys.push(key);
  }
  this.mp.set(key, value);
}
