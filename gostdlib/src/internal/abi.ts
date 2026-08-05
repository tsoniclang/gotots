import type { GoPointer } from "@gotots/runtime/pointer.js";
import type { GoUnsafePointer } from "@gotots/runtime/unsafe-pointer.js";
import type {
  bool,
  int32,
  uint16,
  uint32,
  uint8,
  uintptr,
} from "@gotots/gostdlib/internal/scalars.js";

export class Kind {
  constructor(readonly value: uint8) {}
}

export class NameOff {
  constructor(readonly value: int32) {}
}

export class TFlag {
  constructor(readonly value: uint8) {}
}

export class TypeOff {
  constructor(readonly value: int32) {}
}

type EqualFunction =
  | ((
      argument0: GoUnsafePointer | undefined,
      argument1: GoUnsafePointer | undefined,
    ) => bool)
  | undefined;

export class Type {
  Size_: uintptr;
  PtrBytes: uintptr;
  Hash: uint32;
  TFlag: TFlag;
  Align_: uint8;
  FieldAlign_: uint8;
  Kind_: Kind;
  Equal: EqualFunction;
  GCData: GoPointer<uint8, uint8> | undefined;
  Str: NameOff;
  PtrToThis: TypeOff;

  constructor(fields: {
    Size_: uintptr;
    PtrBytes: uintptr;
    Hash: uint32;
    TFlag: TFlag;
    Align_: uint8;
    FieldAlign_: uint8;
    Kind_: Kind;
    Equal: EqualFunction;
    GCData: GoPointer<uint8, uint8> | undefined;
    Str: NameOff;
    PtrToThis: TypeOff;
  }) {
    this.Size_ = fields.Size_;
    this.PtrBytes = fields.PtrBytes;
    this.Hash = fields.Hash;
    this.TFlag = fields.TFlag;
    this.Align_ = fields.Align_;
    this.FieldAlign_ = fields.FieldAlign_;
    this.Kind_ = fields.Kind_;
    this.Equal = fields.Equal;
    this.GCData = fields.GCData;
    this.Str = fields.Str;
    this.PtrToThis = fields.PtrToThis;
  }
}

export class UncommonType {
  PkgPath: NameOff;
  Mcount: uint16;
  Xcount: uint16;
  Moff: uint32;

  constructor(fields: {
    PkgPath: NameOff;
    Mcount: uint16;
    Xcount: uint16;
    Moff: uint32;
  }) {
    this.PkgPath = fields.PkgPath;
    this.Mcount = fields.Mcount;
    this.Xcount = fields.Xcount;
    this.Moff = fields.Moff;
  }
}
