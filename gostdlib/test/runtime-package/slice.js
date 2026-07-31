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

  static make(length, capacity, zero) {
    const numericLength = Number(length);
    const numericCapacity = capacity === null ? numericLength : Number(capacity);
    if (!Number.isSafeInteger(numericLength) || numericLength < 0 || numericCapacity < numericLength) {
      throw new RangeError("slice bounds out of range");
    }
    const values = new Array(numericCapacity).fill(zero);
    const result = new RuntimeSlice(values);
    result.length = numericLength;
    result.capacity = numericCapacity;
    return result;
  }

  static copy(target, source) {
    const count = Math.min(target.length, source.length);
    const values = Array.from({ length: count }, (_, index) => source.get(index));
    for (let index = 0; index < count; index += 1) {
      target.set(index, values[index]);
    }
    return count;
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

  slice(low, high, max) {
    const numericLow = Number(low);
    const numericHigh = high === null ? this.length : Number(high);
    const numericMax = max === null ? this.capacity : Number(max);
    if (numericLow < 0 || numericHigh < numericLow || numericMax < numericHigh || numericMax > this.capacity) {
      throw new RangeError("slice bounds out of range");
    }
    const result = new RuntimeSlice(this.#values.slice(numericLow, numericMax), this.#nil);
    result.length = numericHigh - numericLow;
    result.capacity = numericMax - numericLow;
    return result;
  }

  append(zero, values) {
    return RuntimeSlice.literal([...this.toArray(), ...values]);
  }

  appendSlice(zero, source) {
    return this.append(zero, source.toArray());
  }

  clear(zero) {
    for (let index = 0; index < this.length; index += 1) {
      this.#values[index] = zero;
    }
  }

  toArray() {
    return this.#values.slice(0, this.length);
  }
}
