export class GoMap {
  constructor(zeroValue, values) {
    this.zeroValue = zeroValue;
    this.values = values;
  }

  static nil(zeroValue) {
    return new GoMap(zeroValue, undefined);
  }

  static make(zeroValue, size, entries) {
    return new GoMap(zeroValue, new Map(entries));
  }

  lookup(key) {
    return this.values?.get(key) ?? this.zeroValue;
  }

  lookupOk(key) {
    return this.values?.has(key) ? [this.values.get(key), true] : [this.zeroValue, false];
  }

  store(key, value) {
    if (this.values === undefined) {
      throw new TypeError("assignment to entry in nil map");
    }
    this.values.set(key, value);
  }

  delete(key) {
    this.values?.delete(key);
  }

  length() {
    return this.values?.size ?? 0;
  }

  isNil() {
    return this.values === undefined;
  }

  clear() {
    this.values?.clear();
  }

  keys() {
    return this.values === undefined ? [] : [...this.values.keys()];
  }
}
