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
import { providerPlaceholder } from "./internal/runtime/placeholder.js";
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
  protected constructor(protected readonly source?: GoInterfaceValue) {}

  Addr(): Value { return providerPlaceholder("reflect.Value.Addr requires generated reflection metadata"); }
  Bool(): bool { return providerPlaceholder("reflect.Value.Bool requires generated reflection metadata"); }
  Bytes(): RuntimeSlice<uint8> { return providerPlaceholder("reflect.Value.Bytes requires generated reflection metadata"); }
  CanInt(): bool { return providerPlaceholder("reflect.Value.CanInt requires generated reflection metadata"); }
  CanSet(): bool { return providerPlaceholder("reflect.Value.CanSet requires generated reflection metadata"); }
  Cap(): int { return providerPlaceholder("reflect.Value.Cap requires generated reflection metadata"); }
  Convert(_target: Type | undefined): Value { return providerPlaceholder("reflect.Value.Convert requires generated reflection metadata"); }
  Elem(): Value { return providerPlaceholder("reflect.Value.Elem requires generated reflection metadata"); }
  Field(_index: int): Value { return providerPlaceholder("reflect.Value.Field requires generated reflection metadata"); }
  Float(): float64 { return providerPlaceholder("reflect.Value.Float requires generated reflection metadata"); }
  Grow(_capacity: int): void { return providerPlaceholder("reflect.Value.Grow requires generated reflection metadata"); }
  Index(_index: int): Value { return providerPlaceholder("reflect.Value.Index requires generated reflection metadata"); }
  Int(): int64 { return providerPlaceholder("reflect.Value.Int requires generated reflection metadata"); }

  Interface(): GoInterfaceValue | undefined {
    if (this.source === undefined) {
      return providerPlaceholder("reflect.Value.Interface on an invalid value requires generated reflection metadata");
    }
    return this.source;
  }

  IsNil(): bool { return providerPlaceholder("reflect.Value.IsNil requires generated reflection metadata"); }
  IsValid(): bool { return this.source !== undefined; }
  IsZero(): bool { return providerPlaceholder("reflect.Value.IsZero requires generated reflection metadata"); }

  Kind(): Kind {
    return this.source === undefined ? Invalid : providerPlaceholder("reflect.Value.Kind requires generated reflection metadata");
  }

  Len(): int { return providerPlaceholder("reflect.Value.Len requires generated reflection metadata"); }
  MapIndex(_key: Value): Value { return providerPlaceholder("reflect.Value.MapIndex requires generated reflection metadata"); }
  MapRange(): MapIter | undefined { return providerPlaceholder("reflect.Value.MapRange requires generated reflection metadata"); }
  NumField(): int { return providerPlaceholder("reflect.Value.NumField requires generated reflection metadata"); }
  Set(_value: Value): void { return providerPlaceholder("reflect.Value.Set requires generated reflection metadata"); }
  SetBool(_value: bool): void { return providerPlaceholder("reflect.Value.SetBool requires generated reflection metadata"); }
  SetBytes(_value: RuntimeSlice<uint8>): void { return providerPlaceholder("reflect.Value.SetBytes requires generated reflection metadata"); }
  SetFloat(_value: float64): void { return providerPlaceholder("reflect.Value.SetFloat requires generated reflection metadata"); }
  SetInt(_value: int64): void { return providerPlaceholder("reflect.Value.SetInt requires generated reflection metadata"); }
  SetIterKey(_iterator: MapIter | undefined): void { return providerPlaceholder("reflect.Value.SetIterKey requires generated reflection metadata"); }
  SetIterValue(_iterator: MapIter | undefined): void { return providerPlaceholder("reflect.Value.SetIterValue requires generated reflection metadata"); }
  SetLen(_length: int): void { return providerPlaceholder("reflect.Value.SetLen requires generated reflection metadata"); }
  SetMapIndex(_key: Value, _element: Value): void { return providerPlaceholder("reflect.Value.SetMapIndex requires generated reflection metadata"); }
  SetString(_value: gostring): void { return providerPlaceholder("reflect.Value.SetString requires generated reflection metadata"); }
  SetUint(_value: uint64): void { return providerPlaceholder("reflect.Value.SetUint requires generated reflection metadata"); }
  SetZero(): void { return providerPlaceholder("reflect.Value.SetZero requires generated reflection metadata"); }

  String(): gostring {
    return this.source === undefined
      ? "<invalid Value>"
      : providerPlaceholder("reflect.Value.String requires generated reflection metadata");
  }

  Type(): Type | undefined { return providerPlaceholder("reflect.Value.Type requires generated reflection metadata"); }
  Uint(): uint64 { return providerPlaceholder("reflect.Value.Uint requires generated reflection metadata"); }
  UnsafePointer(): GoUnsafePointer | undefined {
    return providerPlaceholder("reflect.Value.UnsafePointer requires generated reflection metadata");
  }
}

class InterfaceValue extends Value {
  constructor(source: GoInterfaceValue | undefined) {
    super(source);
  }
}

export function ValueOf(value: GoInterfaceValue | undefined): Value {
  return new InterfaceValue(value);
}

class MapIterator {}

export type MapIter = MapIterator;

export const MapIter = Object.freeze({
  Key(_receiver: MapIter | undefined): Value {
    return providerPlaceholder("reflect.MapIter.Key requires generated reflection metadata");
  },
  Next(_receiver: MapIter | undefined): bool {
    return providerPlaceholder("reflect.MapIter.Next requires generated reflection metadata");
  },
  Value(_receiver: MapIter | undefined): Value {
    return providerPlaceholder("reflect.MapIter.Value requires generated reflection metadata");
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

export function Append(_slice: Value, _values: RuntimeSlice<Value>): Value {
  return providerPlaceholder("reflect.Append requires generated reflection metadata");
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

export function MakeMap(_type: Type | undefined): Value {
  return providerPlaceholder("reflect.MakeMap requires generated reflection metadata");
}

export function MakeSlice(
  _type: Type | undefined,
  _length: int,
  _capacity: int,
): Value {
  return providerPlaceholder("reflect.MakeSlice requires generated reflection metadata");
}

export function MapOf(
  _key: Type | undefined,
  _element: Type | undefined,
): Type | undefined {
  return providerPlaceholder("reflect.MapOf requires generated reflection metadata");
}

export function New(_type: Type | undefined): Value {
  return providerPlaceholder("reflect.New requires generated reflection metadata");
}

export function PointerTo(_type: Type | undefined): Type | undefined {
  return providerPlaceholder("reflect.PointerTo requires generated reflection metadata");
}

export function SliceOf(_type: Type | undefined): Type | undefined {
  return providerPlaceholder("reflect.SliceOf requires generated reflection metadata");
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

export function Zero(_type: Type | undefined): Value {
  return providerPlaceholder("reflect.Zero requires generated reflection metadata");
}

const kindNames: readonly string[] = [
  "invalid", "bool", "int", "int8", "int16", "int32", "int64", "uint",
  "uint8", "uint16", "uint32", "uint64", "uintptr", "float32", "float64",
  "complex64", "complex128", "array", "chan", "func", "interface", "map",
  "ptr", "slice", "string", "struct", "unsafe.Pointer",
];

