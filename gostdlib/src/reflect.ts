import type { GoComplex128 } from "@gotots/runtime/complex.js";
import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type {
  bool,
  float64,
  gostring,
  int,
  int64,
  uint,
  uint8,
  uint64,
  uintptr,
} from "@gotots/gostdlib/internal/scalars.js";
import type { GoUnsafePointer } from "@gotots/runtime/unsafe-pointer.js";

import { hostInteger } from "./internal/host-integer.js";

import {
  getStructTag,
  lookupStructTag,
} from "./internal/portable/reflect/struct-tag.js";
import { ProviderError } from "./internal/runtime/error.js";
import { providerPlaceholder } from "./internal/runtime/placeholder.js";
import {
  pointerDescriptorFor,
  resolveRuntimeType,
  runtimeValueOperations,
  type RuntimeValueLocation,
  type RuntimeValueOperations,
} from "./internal/portable/reflect/runtime-value.js";
import type { Seq } from "./iter.js";

export class ChanDir {
  constructor(readonly value: int) {}
}

export class Kind {
  constructor(readonly value: uint) {}

  String(): gostring {
    return kindNames[hostInteger(this.value)] ?? `Kind(${this.value})`;
  }
}

export const Invalid = new Kind(0n);
export const Bool = new Kind(1n);
export const Int = new Kind(2n);
export const Int8 = new Kind(3n);
export const Int16 = new Kind(4n);
export const Int32 = new Kind(5n);
export const Int64 = new Kind(6n);
export const Uint = new Kind(7n);
export const Uint8 = new Kind(8n);
export const Uint16 = new Kind(9n);
export const Uint32 = new Kind(10n);
export const Uint64 = new Kind(11n);
export const Uintptr = new Kind(12n);
export const Float32 = new Kind(13n);
export const Float64 = new Kind(14n);
export const Complex64 = new Kind(15n);
export const Complex128 = new Kind(16n);
export const Array = new Kind(17n);
export const Chan = new Kind(18n);
export const Func = new Kind(19n);
export const Interface = new Kind(20n);
export const Map = new Kind(21n);
export const Pointer = new Kind(22n);
export const Ptr = Pointer;
export const Slice = new Kind(23n);
export const String = new Kind(24n);
export const Struct = new Kind(25n);
export const UnsafePointer = new Kind(26n);

export abstract class Value {
  protected constructor(
    private readonly stored?: GoInterfaceValue,
    private readonly location?: RuntimeValueLocation,
    private readonly addressable: bool = false,
  ) {}

  // source reads live through the location so mutations that replace the
  // represented value (Set, SetLen, Grow, SetBytes) stay visible to every
  // subsequent read of the same Value, exactly like Go's pointer-backed
  // addressable values.
  protected get source(): GoInterfaceValue | undefined {
    return this.location === undefined
      ? this.stored
      : this.location.get();
  }

  private static located(location: RuntimeValueLocation): Value {
    return new LocatedValue(location.get(), location, true);
  }

  static $append(target: Value, values: RuntimeSlice<Value>): Value {
    const operation = target.operations()?.append;
    if (operation === undefined || target.source === undefined) {
      return GoPanic.raise(
        new ProviderError(
          `reflect: call of reflect.Append on ${target.kindText()} Value`,
        ),
      );
    }
    const boxes: GoInterfaceValue[] = [];
    for (let index = 0; index < values.length; index++) {
      const element = values.get(index);
      const box = element.source;
      if (box === undefined) {
        return GoPanic.raise(
          new ProviderError(
            "reflect: call of reflect.Append on zero Value",
          ),
        );
      }
      boxes.push(box);
    }
    return new InterfaceValue(operation(target.source, boxes));
  }

  private resolvedType(): Type | undefined {
    return this.source === undefined
      ? undefined
      : resolveRuntimeType(this.source);
  }

  private operations(): RuntimeValueOperations | undefined {
    return runtimeValueOperations(this.resolvedType());
  }

  private kindText(): string {
    const type = this.resolvedType();
    return type === undefined ? "zero" : `${type.Kind().String()}`;
  }

  private operationPanic(operation: string): never {
    return GoPanic.raise(
      new ProviderError(
        `reflect: call of reflect.Value.${operation} on ${this.kindText()} Value`,
      ),
    );
  }

  Addr(): Value {
    if (!this.addressable || this.location === undefined) {
      return GoPanic.raise(
        new ProviderError(
          "reflect.Value.Addr of unaddressable value",
        ),
      );
    }
    return new AddressValue(this.location);
  }

  Bool(): bool {
    const operation = this.operations()?.bool;
    if (operation === undefined || this.source === undefined) {
      return this.operationPanic("Bool");
    }
    return operation(this.source);
  }
  Bytes(): RuntimeSlice<uint8> {
    const operations = this.operations();
    const box = this.source;
    if (operations === undefined || box === undefined) {
      return this.operationPanic("Bytes");
    }
    const bytes = operations.bytes;
    if (bytes === undefined) {
      const type = this.resolvedType();
      if (type !== undefined && type.Kind().value === Slice.value) {
        return GoPanic.raise(
          new ProviderError("reflect.Value.Bytes of non-byte slice"),
        );
      }
      return this.operationPanic("Bytes");
    }
    return bytes(box);
  }
  CanInt(): bool {
    return this.operations()?.int !== undefined;
  }
  CanSet(): bool {
    return this.addressable &&
      this.location !== undefined &&
      this.location.settable;
  }
  Cap(): int {
    const operation = this.operations()?.cap;
    if (operation === undefined || this.source === undefined) {
      return this.operationPanic("Cap");
    }
    return operation(this.source);
  }
  Convert(target: Type | undefined): Value {
    const operations = this.operations();
    const type = this.resolvedType();
    if (operations === undefined || type === undefined) {
      return this.operationPanic("Convert");
    }
    if (target === undefined) {
      return GoPanic.raise(
        new ProviderError("reflect: Value.Convert(nil)"),
      );
    }
    const targetOperations = runtimeValueOperations(target);
    const converted =
      targetOperations === undefined
        ? undefined
        : this.convertedBox(operations, targetOperations);
    if (converted === undefined) {
      return GoPanic.raise(
        new ProviderError(
          `reflect.Value.Convert: value of type ${type.String()} cannot be converted to type ${target.String()}`,
        ),
      );
    }
    return new InterfaceValue(converted);
  }

  private convertedBox(
    operations: RuntimeValueOperations,
    target: RuntimeValueOperations,
  ): GoInterfaceValue | undefined {
    const box = this.source;
    if (box === undefined) {
      return undefined;
    }
    if (operations.int !== undefined || operations.uint !== undefined) {
      const wide =
        operations.int !== undefined
          ? operations.int(box)
          : operations.uint!(box);
      if (target.boxInt !== undefined) {
        return target.boxInt(wide);
      }
      if (target.boxUint !== undefined) {
        return target.boxUint(wide);
      }
      if (target.boxFloat !== undefined) {
        return target.boxFloat(globalThis.Number(wide));
      }
      if (target.boxString !== undefined) {
        return target.boxString(stringFromRune(wide));
      }
      return undefined;
    }
    if (operations.float !== undefined) {
      const wide = operations.float(box);
      if (target.boxFloat !== undefined) {
        return target.boxFloat(wide);
      }
      if (target.boxInt !== undefined) {
        return target.boxInt(globalThis.BigInt(globalThis.Math.trunc(wide)));
      }
      if (target.boxUint !== undefined) {
        return target.boxUint(globalThis.BigInt(globalThis.Math.trunc(wide)));
      }
      return undefined;
    }
    if (operations.string !== undefined) {
      if (target.boxString !== undefined) {
        return target.boxString(operations.string(box));
      }
      return undefined;
    }
    return undefined;
  }
  Elem(): Value {
    const operation = this.operations()?.elem;
    if (operation === undefined || this.source === undefined) {
      return this.operationPanic("Elem");
    }
    const location = operation(this.source);
    return location === undefined
      ? new LocatedValue(undefined)
      : Value.located(location);
  }
  Field(index: int): Value {
    const operations = this.operations();
    const field = operations?.field;
    const count = operations?.numField;
    if (
      field === undefined ||
      count === undefined ||
      this.source === undefined
    ) {
      return this.operationPanic("Field");
    }
    if (index < 0n || index >= count) {
      return GoPanic.raise(
        new ProviderError("reflect: Field index out of range"),
      );
    }
    const location = field(this.source, index);
    return new LocatedValue(location.get(), location, this.addressable);
  }
  Float(): float64 {
    const operation = this.operations()?.float;
    if (operation === undefined || this.source === undefined) {
      return this.operationPanic("Float");
    }
    return operation(this.source);
  }
  Grow(count: int): void {
    const target = this.settableLocation("Grow");
    const grown = this.operations()?.grown;
    const box = this.source;
    if (grown === undefined || box === undefined) {
      return this.operationPanic("Grow");
    }
    if (count < 0n) {
      return GoPanic.raise(
        new ProviderError("reflect.Value.Grow: negative len"),
      );
    }
    target.set(grown(box, count));
  }
  Index(index: int): Value {
    const operation = this.operations()?.index;
    if (operation === undefined || this.source === undefined) {
      return this.operationPanic("Index");
    }
    if (index < 0n || index >= this.Len()) {
      return GoPanic.raise(
        new ProviderError("reflect: slice index out of range"),
      );
    }
    const location = operation(this.source, index);
    return new LocatedValue(location.get(), location, true);
  }
  Int(): int64 {
    const operation = this.operations()?.int;
    if (operation === undefined || this.source === undefined) {
      return this.operationPanic("Int");
    }
    return operation(this.source);
  }

  Interface(): GoInterfaceValue | undefined {
    if (this.source === undefined) {
      return providerPlaceholder("reflect.Value.Interface on an invalid value requires generated reflection metadata");
    }
    return this.source;
  }

  IsNil(): bool {
    const operation = this.operations()?.isNil;
    if (operation === undefined || this.source === undefined) {
      return this.operationPanic("IsNil");
    }
    return operation(this.source);
  }
  IsValid(): bool { return this.source !== undefined; }
  IsZero(): bool {
    const operation = this.operations()?.isZero;
    if (operation === undefined || this.source === undefined) {
      return this.operationPanic("IsZero");
    }
    return operation(this.source);
  }

  Kind(): Kind {
    const type = this.resolvedType();
    return type === undefined ? Invalid : type.Kind();
  }

  Len(): int {
    const operation = this.operations()?.len;
    if (operation === undefined || this.source === undefined) {
      return this.operationPanic("Len");
    }
    return operation(this.source);
  }
  MapIndex(key: Value): Value {
    const operation = this.operations()?.mapIndex;
    const box = this.source;
    const keyBox = key.source;
    if (
      operation === undefined ||
      box === undefined ||
      keyBox === undefined
    ) {
      return this.operationPanic("MapIndex");
    }
    return new InterfaceValue(operation(box, keyBox));
  }
  MapRange(): MapIter | undefined {
    const operations = this.operations();
    const keys = operations?.mapKeys;
    const lookup = operations?.mapIndex;
    const box = this.source;
    if (keys === undefined || lookup === undefined || box === undefined) {
      return this.operationPanic("MapRange");
    }
    return new MapIterator(keys(box), (key) => lookup(box, key));
  }
  NumField(): int {
    const count = this.operations()?.numField;
    if (count === undefined || this.source === undefined) {
      return this.operationPanic("NumField");
    }
    return count;
  }
  Set(value: Value): void {
    const target = this.settableLocation("Set");
    const payload = value.source;
    if (payload === undefined) {
      return GoPanic.raise(
        new ProviderError("reflect: Set using zero Value argument"),
      );
    }
    target.set(payload);
  }

  private settableLocation(operation: string): RuntimeValueLocation {
    if (this.location === undefined || this.source === undefined) {
      return this.operationPanic(operation);
    }
    if (!this.addressable || !this.location.settable) {
      return GoPanic.raise(
        new ProviderError(
          `reflect: reflect.Value.${operation} using unaddressable value`,
        ),
      );
    }
    return this.location;
  }
  SetBool(value: bool): void {
    const target = this.settableLocation("SetBool");
    const box = this.operations()?.boxBool;
    if (box === undefined) {
      return this.operationPanic("SetBool");
    }
    target.set(box(value));
  }
  SetBytes(value: RuntimeSlice<uint8>): void {
    const target = this.settableLocation("SetBytes");
    const operations = this.operations();
    const box = this.source;
    if (operations === undefined || box === undefined) {
      return this.operationPanic("SetBytes");
    }
    const boxBytes = operations.boxBytes;
    if (boxBytes === undefined) {
      const type = this.resolvedType();
      if (type !== undefined && type.Kind().value === Slice.value) {
        return GoPanic.raise(
          new ProviderError(
            "reflect.Value.SetBytes of non-byte slice",
          ),
        );
      }
      return this.operationPanic("SetBytes");
    }
    target.set(boxBytes(value));
  }
  SetFloat(value: float64): void {
    const target = this.settableLocation("SetFloat");
    const box = this.operations()?.boxFloat;
    if (box === undefined) {
      return this.operationPanic("SetFloat");
    }
    target.set(box(value));
  }
  SetInt(value: int64): void {
    const target = this.settableLocation("SetInt");
    const box = this.operations()?.boxInt;
    if (box === undefined) {
      return this.operationPanic("SetInt");
    }
    target.set(box(value));
  }
  SetIterKey(iterator: MapIter | undefined): void {
    const target = this.settableLocation("SetIterKey");
    const state = mapIteratorState(iterator, "SetIterKey");
    target.set(state.currentKey());
  }
  SetIterValue(iterator: MapIter | undefined): void {
    const target = this.settableLocation("SetIterValue");
    const state = mapIteratorState(iterator, "SetIterValue");
    const value = state.currentValue();
    if (value === undefined) {
      return GoPanic.raise(
        new ProviderError(
          "reflect: map entry deleted during iteration",
        ),
      );
    }
    target.set(value);
  }
  SetLen(length: int): void {
    const target = this.settableLocation("SetLen");
    const operations = this.operations();
    const resliced = operations?.resliced;
    const capacity = operations?.cap;
    const box = this.source;
    if (
      resliced === undefined ||
      capacity === undefined ||
      box === undefined
    ) {
      return this.operationPanic("SetLen");
    }
    if (length < 0n || length > capacity(box)) {
      return GoPanic.raise(
        new ProviderError(
          "reflect: slice length out of range in SetLen",
        ),
      );
    }
    target.set(resliced(box, length));
  }
  SetMapIndex(key: Value, element: Value): void {
    const operation = this.operations()?.mapStore;
    const box = this.source;
    const keyBox = key.source;
    if (
      operation === undefined ||
      box === undefined ||
      keyBox === undefined
    ) {
      return this.operationPanic("SetMapIndex");
    }
    operation(box, keyBox, element.source);
  }
  SetString(value: gostring): void {
    const target = this.settableLocation("SetString");
    const box = this.operations()?.boxString;
    if (box === undefined) {
      return this.operationPanic("SetString");
    }
    target.set(box(value));
  }
  SetUint(value: uint64): void {
    const target = this.settableLocation("SetUint");
    const box = this.operations()?.boxUint;
    if (box === undefined) {
      return this.operationPanic("SetUint");
    }
    target.set(box(value));
  }
  SetZero(): void {
    const target = this.settableLocation("SetZero");
    const zero = this.operations()?.zero;
    if (zero === undefined) {
      return this.operationPanic("SetZero");
    }
    target.set(zero());
  }

  String(): gostring {
    if (this.source === undefined) {
      return "<invalid Value>";
    }
    const operation = this.operations()?.string;
    if (operation !== undefined) {
      return operation(this.source);
    }
    const type = this.resolvedType();
    return type === undefined
      ? "<invalid Value>"
      : `<${type.String()} Value>`;
  }

  Type(): Type | undefined {
    if (this.source === undefined) {
      return GoPanic.raise(
        new ProviderError(
          "reflect: call of reflect.Value.Type on zero Value",
        ),
      );
    }
    const type = this.resolvedType();
    if (type === undefined) {
      return GoPanic.raiseRuntime(
        "reflect: value type has no registered canonical descriptor",
      );
    }
    return type;
  }
  Uint(): uint64 {
    const operation = this.operations()?.uint;
    if (operation === undefined || this.source === undefined) {
      return this.operationPanic("Uint");
    }
    return operation(this.source);
  }
  UnsafePointer(): GoUnsafePointer | undefined {
    return providerPlaceholder("reflect.Value.UnsafePointer requires generated reflection metadata");
  }
}

class InterfaceValue extends Value {
  constructor(source: GoInterfaceValue | undefined) {
    super(source);
  }
}

class LocatedValue extends Value {
  constructor(
    source: GoInterfaceValue | undefined,
    location?: RuntimeValueLocation,
    addressable: bool = false,
  ) {
    super(source, location, addressable);
  }
}

// AddressValue is the exact pointer view produced by Addr: it has no
// interface box of its own, is always valid and non-nil, and Elem returns
// the held addressable location.
class AddressValue extends Value {
  constructor(private readonly held: RuntimeValueLocation) {
    super(undefined, undefined, false);
  }

  override IsValid(): bool {
    return true;
  }

  override IsNil(): bool {
    return false;
  }

  override Kind(): Kind {
    return Pointer;
  }

  override Elem(): Value {
    return new LocatedValue(this.held.get(), this.held, true);
  }
}

export function ValueOf(value: GoInterfaceValue | undefined): Value {
  return new InterfaceValue(value);
}

class MapIterator {
  position = -1;

  constructor(
    readonly keys: readonly GoInterfaceValue[],
    readonly valueAt: (key: GoInterfaceValue) => GoInterfaceValue | undefined,
  ) {}

  currentKey(): GoInterfaceValue {
    const key = this.keys[this.position];
    if (key === undefined) {
      return GoPanic.raise(
        new ProviderError("reflect: map iterator has no current entry"),
      );
    }
    return key;
  }

  currentValue(): GoInterfaceValue | undefined {
    return this.valueAt(this.currentKey());
  }
}

// mapIteratorState validates one positioned live iterator with the exact
// Go panic messages for premature and exhausted access.
function mapIteratorState(
  receiver: MapIterator | undefined,
  operation: string,
): MapIterator {
  if (receiver === undefined) {
    return GoPanic.raiseRuntime(
      "invalid memory address or nil pointer dereference",
    );
  }
  if (receiver.position < 0) {
    return GoPanic.raise(
      new ProviderError(`MapIter.${operation} called before Next`),
    );
  }
  if (receiver.position >= receiver.keys.length) {
    return GoPanic.raise(
      new ProviderError(
        `MapIter.${operation} called on exhausted iterator`,
      ),
    );
  }
  return receiver;
}

export type MapIter = MapIterator;

export const MapIter = Object.freeze({
  Key(receiver: MapIter | undefined): Value {
    return new InterfaceValue(
      mapIteratorState(receiver, "Key").currentKey(),
    );
  },
  Next(receiver: MapIter | undefined): bool {
    if (receiver === undefined) {
      return GoPanic.raiseRuntime(
        "invalid memory address or nil pointer dereference",
      );
    }
    if (receiver.position >= receiver.keys.length) {
      return GoPanic.raise(
        new ProviderError("MapIter.Next called on exhausted iterator"),
      );
    }
    receiver.position++;
    while (
      receiver.position < receiver.keys.length &&
      receiver.currentValue() === undefined
    ) {
      receiver.position++;
    }
    return receiver.position < receiver.keys.length;
  },
  Value(receiver: MapIter | undefined): Value {
    const value = mapIteratorState(receiver, "Value").currentValue();
    if (value === undefined) {
      return GoPanic.raise(
        new ProviderError(
          "reflect: map entry deleted during iteration",
        ),
      );
    }
    return new InterfaceValue(value);
  },
});

export class Method {
  Name: gostring;
  PkgPath: gostring;
  Type: Type | undefined;
  Func: Value;
  Index: int;

  constructor(fields: {
    Name: gostring;
    PkgPath: gostring;
    Type: Type | undefined;
    Func: Value;
    Index: int;
  }) {
    this.Name = fields.Name;
    this.PkgPath = fields.PkgPath;
    this.Type = fields.Type;
    this.Func = fields.Func;
    this.Index = fields.Index;
  }
}

export class StructField {
  Name: gostring;
  PkgPath: gostring;
  Type: Type | undefined;
  Tag: StructTag;
  Offset: uintptr;
  Index: RuntimeSlice<int>;
  Anonymous: bool;

  constructor(fields: {
    Name: gostring;
    PkgPath: gostring;
    Type: Type | undefined;
    Tag: StructTag;
    Offset: uintptr;
    Index: RuntimeSlice<int>;
    Anonymous: bool;
  }) {
    this.Name = fields.Name;
    this.PkgPath = fields.PkgPath;
    this.Type = fields.Type;
    this.Tag = fields.Tag;
    this.Offset = fields.Offset;
    this.Index = fields.Index;
    this.Anonymous = fields.Anonymous;
  }

  IsExported(): bool {
    return this.PkgPath === "";
  }
}

export class StructTag {
  constructor(readonly value: gostring) {}

  Get(key: gostring): gostring {
    return getStructTag(this.value, key);
  }

  Lookup(key: gostring): [gostring, bool] {
    return lookupStructTag(this.value, key);
  }
}

export interface Type extends GoInterfaceValue {
  Align(): int;
  AssignableTo(u: Type | undefined): bool;
  Bits(): int;
  CanSeq(): bool;
  CanSeq2(): bool;
  ChanDir(): ChanDir;
  Comparable(): bool;
  ConvertibleTo(u: Type | undefined): bool;
  Elem(): Type | undefined;
  Field(i: int): StructField;
  FieldAlign(): int;
  FieldByIndex(index: RuntimeSlice<int>): StructField;
  FieldByName(name: gostring): [StructField, bool];
  FieldByNameFunc(
    match: ((name: gostring) => bool) | undefined,
  ): [StructField, bool];
  Fields(): Seq<StructField>;
  Implements(u: Type | undefined): bool;
  In(i: int): Type | undefined;
  Ins(): Seq<Type | undefined>;
  IsVariadic(): bool;
  Key(): Type | undefined;
  Kind(): Kind;
  Len(): int;
  Method(argument0: int): Method;
  MethodByName(argument0: gostring): [Method, bool];
  Methods(): Seq<Method>;
  Name(): gostring;
  NumField(): int;
  NumIn(): int;
  NumMethod(): int;
  NumOut(): int;
  Out(i: int): Type | undefined;
  Outs(): Seq<Type | undefined>;
  OverflowComplex(x: GoComplex128): bool;
  OverflowFloat(x: float64): bool;
  OverflowInt(x: int64): bool;
  OverflowUint(x: uint64): bool;
  PkgPath(): gostring;
  Size(): uintptr;
  String(): gostring;
}

export function Append(slice: Value, values: RuntimeSlice<Value>): Value {
  return Value.$append(slice, values);
}

export function DeepEqual(
  _left: GoInterfaceValue | undefined,
  _right: GoInterfaceValue | undefined,
): bool {
  return providerPlaceholder("reflect.DeepEqual requires generated reflection metadata");
}

export function Indirect(_value: Value): Value {
  return providerPlaceholder("reflect.Indirect requires generated reflection metadata");
}

export function MakeMap(type: Type | undefined): Value {
  if (type === undefined || type.Kind().value !== Map.value) {
    return GoPanic.raise(
      new ProviderError("reflect: MakeMap of non-map type"),
    );
  }
  const operation = runtimeValueOperations(type)?.makeMap;
  if (operation === undefined) {
    return GoPanic.raise(
      new ProviderError(
        `reflect: MakeMap requires a generated value facet for ${type.String()}`,
      ),
    );
  }
  return new InterfaceValue(operation());
}

export function MakeSlice(
  type: Type | undefined,
  length: int,
  capacity: int,
): Value {
  if (type === undefined || type.Kind().value !== Slice.value) {
    return GoPanic.raise(
      new ProviderError("reflect: MakeSlice of non-slice type"),
    );
  }
  if (length < 0n) {
    return GoPanic.raise(
      new ProviderError(
        "reflect: negative len argument in call to reflect.MakeSlice",
      ),
    );
  }
  if (capacity < 0n) {
    return GoPanic.raise(
      new ProviderError(
        "reflect: negative cap argument in call to reflect.MakeSlice",
      ),
    );
  }
  if (length > capacity) {
    return GoPanic.raise(
      new ProviderError(
        "reflect: len > cap in call to reflect.MakeSlice",
      ),
    );
  }
  const operation = runtimeValueOperations(type)?.makeSlice;
  if (operation === undefined) {
    return GoPanic.raise(
      new ProviderError(
        `reflect: MakeSlice requires a generated value facet for ${type.String()}`,
      ),
    );
  }
  return new InterfaceValue(operation(length, capacity));
}

export function MapOf(
  key: Type | undefined,
  element: Type | undefined,
): Type | undefined {
  return GoPanic.raise(
    new ProviderError(
      `reflect: MapOf requires a generated descriptor for map[${key === undefined ? "?" : key.String()}]${element === undefined ? "?" : element.String()}`,
    ),
  );
}

export function New(type: Type | undefined): Value {
  if (type === undefined) {
    return GoPanic.raise(new ProviderError("reflect: New(nil)"));
  }
  const pointerType = pointerDescriptorFor(type);
  const operation =
    pointerType === undefined
      ? undefined
      : runtimeValueOperations(pointerType)?.newPointer;
  if (operation === undefined) {
    return GoPanic.raise(
      new ProviderError(
        `reflect: New requires a generated pointer facet for ${type.String()}`,
      ),
    );
  }
  return new InterfaceValue(operation());
}

export function PointerTo(type: Type | undefined): Type | undefined {
  if (type === undefined) {
    return GoPanic.raise(new ProviderError("reflect: PointerTo(nil)"));
  }
  const pointerType = pointerDescriptorFor(type);
  if (pointerType === undefined) {
    return GoPanic.raise(
      new ProviderError(
        `reflect: PointerTo requires a generated descriptor for *${type.String()}`,
      ),
    );
  }
  return pointerType;
}

export function SliceOf(type: Type | undefined): Type | undefined {
  return GoPanic.raise(
    new ProviderError(
      `reflect: SliceOf requires a generated descriptor for []${type === undefined ? "?" : type.String()}`,
    ),
  );
}

export function TypeAssert<T>(_value: Value): [T, bool] {
  return providerPlaceholder("reflect.TypeAssert requires generated reflection metadata");
}

export function TypeFor<T>(): Type | undefined {
  return providerPlaceholder("reflect.TypeFor requires generated reflection metadata");
}

export function TypeOf(_value: GoInterfaceValue | undefined): Type | undefined {
  return providerPlaceholder("reflect.TypeOf requires generated reflection metadata");
}

export function Zero(type: Type | undefined): Value {
  if (type === undefined) {
    return GoPanic.raise(new ProviderError("reflect: Zero(nil)"));
  }
  const operation = runtimeValueOperations(type)?.zero;
  if (operation === undefined) {
    return GoPanic.raise(
      new ProviderError(
        `reflect: Zero requires a generated value facet for ${type.String()}`,
      ),
    );
  }
  return new InterfaceValue(operation());
}

// stringFromRune is the exact Go integer-to-string conversion: one rune,
// with values outside the valid Unicode code point range replaced by
// U+FFFD.
function stringFromRune(value: int64 | uint64): gostring {
  const code = globalThis.Number(value);
  if (
    !globalThis.Number.isInteger(code) ||
    code < 0 ||
    code > 0x10ffff ||
    (code >= 0xd800 && code <= 0xdfff)
  ) {
    return "\uFFFD";
  }
  return globalThis.String.fromCodePoint(code);
}

const kindNames: readonly string[] = [
  "invalid", "bool", "int", "int8", "int16", "int32", "int64", "uint",
  "uint8", "uint16", "uint32", "uint64", "uintptr", "float32", "float64",
  "complex64", "complex128", "array", "chan", "func", "interface", "map",
  "ptr", "slice", "string", "struct", "unsafe.Pointer",
];

