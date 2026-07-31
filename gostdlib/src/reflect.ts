import type { GoComplex128 } from "@gotots/runtime/complex.js";
import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type {
  bool,
  float64,
  gostring,
  int64,
  uint64,
} from "@gotots/runtime/scalars.js";

import { getStructTag } from "./internal/portable/reflect/struct-tag.js";
import type { Seq } from "./iter.js";

export class ChanDir {
  constructor(readonly value: int64) {}
}

export class Kind {
  constructor(readonly value: uint64) {}
}

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
export const Float64 = new Kind(14);
export const Array = new Kind(17);
export const Map = new Kind(21);
export const Slice = new Kind(23);
export const String = new Kind(24);
export const Struct = new Kind(25);

export abstract class Value {
  protected constructor() {}
}

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

  static IsExported(receiver: StructField): bool {
    return receiver.PkgPath === "";
  }
}

export class StructTag {
  constructor(readonly value: gostring) {}

  static Get(receiver: StructTag, key: gostring): gostring {
    return getStructTag(receiver.value, key);
  }
}

export interface Type {
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
