export class RuntimeSlice {
  #values;
  #nil;

  constructor(values, nil = false) {
    this.#values = values;
    this.#nil = nil;
    this.length = values.length;
    this.capacity = values.length;
  }

  static nil() {
    return new RuntimeSlice([], true);
  }

  static literal(values) {
    return new RuntimeSlice([...values]);
  }

  isNil() {
    return this.#nil;
  }

  get(index) {
    const value = this.#values[Number(index)];
    if (value === undefined && Number(index) >= this.length) {
      throw new RangeError("slice index out of range");
    }
    return value;
  }

  set(index, value) {
    const numericIndex = Number(index);
    if (numericIndex < 0 || numericIndex >= this.length) {
      throw new RangeError("slice index out of range");
    }
    this.#values[numericIndex] = value;
    return value;
  }

  toArray() {
    return [...this.#values];
  }
}
