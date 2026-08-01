import type { GoComplex128 } from "@gotots/runtime/complex.js";
import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type {
  bool,
  float64,
  gostring,
  int64,
  uint8,
  uint64,
} from "@gotots/runtime/scalars.js";
import type { GoUnsafePointer } from "@gotots/runtime/unsafe-pointer.js";

import {
  getStructTag,
  lookupStructTag,
} from "./internal/portable/reflect/struct-tag.js";
import { ProviderError } from "./internal/runtime/error.js";
import type { Seq } from "./iter.js";

export class ChanDir {
  constructor(readonly value: int64) {}
}

export class Kind {
  constructor(readonly value: uint64) {}

  String(): gostring {
    return kindNames[this.value] ?? `Kind(${this.value})`;
  }
}

export const Invalid = new Kind(0);
export const Bool = new Kind(1);
export const Int = new Kind(2);
export const Int8 = new Kind(3);
export const Int16 = new Kind(4);
export const Int32 = new Kind(5);
export const Int64 = new Kind(6);
export const Uint = new Kind(7);
export const Uint8 = new Kind(8);
export const Uint16 = new Kind(9);
export const Uint32 = new Kind(10);
export const Uint64 = new Kind(11);
export const Uintptr = new Kind(12);
export const Float32 = new Kind(13);
export const Float64 = new Kind(14);
export const Complex64 = new Kind(15);
export const Complex128 = new Kind(16);
export const Array = new Kind(17);
export const Chan = new Kind(18);
export const Func = new Kind(19);
export const Interface = new Kind(20);
export const Map = new Kind(21);
export const Pointer = new Kind(22);
export const Ptr = Pointer;
export const Slice = new Kind(23);
export const String = new Kind(24);
export const Struct = new Kind(25);
export const UnsafePointer = new Kind(26);

export abstract class Value {
  protected constructor(protected readonly source?: GoInterfaceValue) {}

  Addr(): Value { return unsupportedReflect("Value.Addr"); }
  Bool(): bool { return unsupportedReflect("Value.Bool"); }
  Bytes(): RuntimeSlice<uint8> { return unsupportedReflect("Value.Bytes"); }
  CanInt(): bool { return unsupportedReflect("Value.CanInt"); }
  CanSet(): bool { return unsupportedReflect("Value.CanSet"); }
  Cap(): int64 { return unsupportedReflect("Value.Cap"); }
  Convert(_target: Type | undefined): Value { return unsupportedReflect("Value.Convert"); }
  Elem(): Value { return unsupportedReflect("Value.Elem"); }
  Field(_index: int64): Value { return unsupportedReflect("Value.Field"); }
  Float(): float64 { return unsupportedReflect("Value.Float"); }
  Grow(_capacity: int64): void { return unsupportedReflect("Value.Grow"); }
  Index(_index: int64): Value { return unsupportedReflect("Value.Index"); }
  Int(): int64 { return unsupportedReflect("Value.Int"); }

  Interface(): GoInterfaceValue | undefined {
    if (this.source === undefined) {
      return unsupportedReflect("Value.Interface on an invalid value");
    }
    return this.source;
  }

  IsNil(): bool { return unsupportedReflect("Value.IsNil"); }
  IsValid(): bool { return this.source !== undefined; }
  IsZero(): bool { return unsupportedReflect("Value.IsZero"); }

  Kind(): Kind {
    return this.source === undefined ? Invalid : unsupportedReflect("Value.Kind");
  }

  Len(): int64 { return unsupportedReflect("Value.Len"); }
  MapIndex(_key: Value): Value { return unsupportedReflect("Value.MapIndex"); }
  MapRange(): MapIter | undefined { return unsupportedReflect("Value.MapRange"); }
  NumField(): int64 { return unsupportedReflect("Value.NumField"); }
  Set(_value: Value): void { return unsupportedReflect("Value.Set"); }
  SetBool(_value: bool): void { return unsupportedReflect("Value.SetBool"); }
  SetBytes(_value: RuntimeSlice<uint8>): void { return unsupportedReflect("Value.SetBytes"); }
  SetFloat(_value: float64): void { return unsupportedReflect("Value.SetFloat"); }
  SetInt(_value: int64): void { return unsupportedReflect("Value.SetInt"); }
  SetIterKey(_iterator: MapIter | undefined): void { return unsupportedReflect("Value.SetIterKey"); }
  SetIterValue(_iterator: MapIter | undefined): void { return unsupportedReflect("Value.SetIterValue"); }
  SetLen(_length: int64): void { return unsupportedReflect("Value.SetLen"); }
  SetMapIndex(_key: Value, _element: Value): void { return unsupportedReflect("Value.SetMapIndex"); }
  SetString(_value: gostring): void { return unsupportedReflect("Value.SetString"); }
  SetUint(_value: uint64): void { return unsupportedReflect("Value.SetUint"); }
  SetZero(): void { return unsupportedReflect("Value.SetZero"); }

  String(): gostring {
    return this.source === undefined
      ? "<invalid Value>"
      : unsupportedReflect("Value.String");
  }

  Type(): Type | undefined { return unsupportedReflect("Value.Type"); }
  Uint(): uint64 { return unsupportedReflect("Value.Uint"); }
  UnsafePointer(): GoUnsafePointer | undefined {
    return unsupportedReflect("Value.UnsafePointer");
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
    return unsupportedReflect("MapIter.Key");
  },
  Next(_receiver: MapIter | undefined): bool {
    return unsupportedReflect("MapIter.Next");
  },
  Value(_receiver: MapIter | undefined): Value {
    return unsupportedReflect("MapIter.Value");
  },
});

export class Method {
  Name: gostring;
  PkgPath: gostring;
  Type: Type | undefined;
  Func: Value;
  Index: int64;

  constructor(fields: {
    Name: gostring;
    PkgPath: gostring;
    Type: Type | undefined;
    Func: Value;
    Index: int64;
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
  Offset: uint64;
  Index: RuntimeSlice<int64>;
  Anonymous: bool;

  constructor(fields: {
    Name: gostring;
    PkgPath: gostring;
    Type: Type | undefined;
    Tag: StructTag;
    Offset: uint64;
    Index: RuntimeSlice<int64>;
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
  Align(): int64;
  AssignableTo(u: Type | undefined): bool;
  Bits(): int64;
  CanSeq(): bool;
  CanSeq2(): bool;
  ChanDir(): ChanDir;
  Comparable(): bool;
  ConvertibleTo(u: Type | undefined): bool;
  Elem(): Type | undefined;
  Field(i: int64): StructField;
  FieldAlign(): int64;
  FieldByIndex(index: RuntimeSlice<int64>): StructField;
  FieldByName(name: gostring): [StructField, bool];
  FieldByNameFunc(
    match: ((name: gostring) => bool) | undefined,
  ): [StructField, bool];
  Fields(): Seq<StructField>;
  Implements(u: Type | undefined): bool;
  In(i: int64): Type | undefined;
  Ins(): Seq<Type | undefined>;
  IsVariadic(): bool;
  Key(): Type | undefined;
  Kind(): Kind;
  Len(): int64;
  Method(argument0: int64): Method;
  MethodByName(argument0: gostring): [Method, bool];
  Methods(): Seq<Method>;
  Name(): gostring;
  NumField(): int64;
  NumIn(): int64;
  NumMethod(): int64;
  NumOut(): int64;
  Out(i: int64): Type | undefined;
  Outs(): Seq<Type | undefined>;
  OverflowComplex(x: GoComplex128): bool;
  OverflowFloat(x: float64): bool;
  OverflowInt(x: int64): bool;
  OverflowUint(x: uint64): bool;
  PkgPath(): gostring;
  Size(): uint64;
  String(): gostring;
}

export function Append(_slice: Value, _values: RuntimeSlice<Value>): Value {
  return unsupportedReflect("Append");
}

export function DeepEqual(
  _left: GoInterfaceValue | undefined,
  _right: GoInterfaceValue | undefined,
): bool {
  return unsupportedReflect("DeepEqual");
}

export function Indirect(_value: Value): Value {
  return unsupportedReflect("Indirect");
}

export function MakeMap(_type: Type | undefined): Value {
  return unsupportedReflect("MakeMap");
}

export function MakeSlice(
  _type: Type | undefined,
  _length: int64,
  _capacity: int64,
): Value {
  return unsupportedReflect("MakeSlice");
}

export function MapOf(
  _key: Type | undefined,
  _element: Type | undefined,
): Type | undefined {
  return unsupportedReflect("MapOf");
}

export function New(_type: Type | undefined): Value {
  return unsupportedReflect("New");
}

export function PointerTo(_type: Type | undefined): Type | undefined {
  return unsupportedReflect("PointerTo");
}

export function SliceOf(_type: Type | undefined): Type | undefined {
  return unsupportedReflect("SliceOf");
}

export function TypeAssert<T>(_value: Value): [T, bool] {
  return unsupportedReflect("TypeAssert");
}

export function TypeFor<T>(): Type | undefined {
  return unsupportedReflect("TypeFor");
}

export function TypeOf(_value: GoInterfaceValue | undefined): Type | undefined {
  return unsupportedReflect("TypeOf");
}

export function Zero(_type: Type | undefined): Value {
  return unsupportedReflect("Zero");
}

const kindNames: readonly string[] = [
  "invalid", "bool", "int", "int8", "int16", "int32", "int64", "uint",
  "uint8", "uint16", "uint32", "uint64", "uintptr", "float32", "float64",
  "complex64", "complex128", "array", "chan", "func", "interface", "map",
  "ptr", "slice", "string", "struct", "unsafe.Pointer",
];

function unsupportedReflect(operation: string): never {
  return GoPanic.raise(
    new ProviderError(
      `reflect.${operation} requires generated reflection metadata`,
    ),
  );
}
