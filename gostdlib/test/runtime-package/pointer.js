const roots = new WeakMap();

function root(owner) {
  let address = roots.get(owner);
  if (address === undefined) {
    address = {};
    roots.set(owner, address);
  }
  return address;
}

export class GoPointer {
  constructor(address, read, write) {
    this.$go$address = address;
    this.read = read;
    this.write = write;
  }

  static cell(value) {
    const storage = [value];
    return new GoPointer(storage, () => storage[0], (next) => {
      storage[0] = next;
    });
  }

  static objectField(owner, key) {
    return new GoPointer(root(owner), () => owner[key], (next) => {
      owner[key] = next;
    });
  }

  static equal(left, right) {
    return left === right || (left !== undefined && right !== undefined && left.$go$address === right.$go$address);
  }

  static dereference(pointer) {
    if (pointer === undefined) {
      throw new TypeError("nil pointer dereference");
    }
    return pointer;
  }

  static direct(pointer) {
    if (pointer === undefined) {
      throw new TypeError("nil pointer dereference");
    }
    return pointer;
  }

  static view(pointer) {
    if (pointer === undefined) {
      return undefined;
    }
    return new GoPointer(pointer.$go$address, pointer.read, pointer.write);
  }

  get value() {
    return this.read();
  }

  set value(value) {
    this.write(value);
  }
}

export function goPointerHash(pointer) {
  return pointer === undefined ? 0 : 1;
}
