export class GoArray {
  constructor(values, length) {
    this.values = values;
    this.length = length;
  }

  static zero(length, zero) {
    return new GoArray(new Array(Number(length)).fill(zero), length);
  }

  static literal(length, zero, indexes, values) {
    const result = GoArray.zero(length, zero);
    for (let entry = 0; entry < indexes.length; entry += 1) {
      result.set(indexes[entry], values[entry]);
    }
    return result;
  }

  copy() {
    return new GoArray([...this.values], this.length);
  }

  get(index) {
    const numericIndex = Number(index);
    if (numericIndex < 0 || numericIndex >= this.length) {
      throw new RangeError("array index out of bounds");
    }
    return this.values[numericIndex];
  }

  set(index, value) {
    const numericIndex = Number(index);
    if (numericIndex < 0 || numericIndex >= this.length) {
      throw new RangeError("array index out of bounds");
    }
    this.values[numericIndex] = value;
  }
}
