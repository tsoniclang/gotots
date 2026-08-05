import {
  GoInterfaceValue,
} from "@gotots/runtime/interface-value.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { GoComplex128 } from "@gotots/runtime/complex.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import type {
  bool,
  float64,
  gostring,
  int64,
  uint64,
} from "@gotots/gostdlib/internal/scalars.js";

import { hostInteger, integerFromHost } from "../../host-integer.js";

import { Seq } from "../../../iter.js";
import {
  ChanDir,
  Invalid,
  Kind,
  Method,
  StructField,
  StructTag,
  type Type,
  Value,
} from "../../../reflect.js";
import { ProviderError } from "../../runtime/error.js";

export interface RuntimeStructFieldMetadata {
  readonly name: gostring;
  readonly type: () => Type;
  readonly pkgPath?: gostring;
  readonly tag?: gostring;
  readonly offset?: uint64;
  readonly index?: readonly int64[];
  readonly anonymous?: bool;
}

export interface RuntimeMethodMetadata {
  readonly name: gostring;
  readonly pkgPath: gostring;
  readonly type: () => Type;
  readonly index: int64;
}

export interface RuntimeTypeMetadata {
  readonly identity: gostring;
  readonly kind: uint64;
  readonly text: gostring;
  readonly size: uint64;
  readonly align: int64;
  readonly name?: gostring;
  readonly pkgPath?: gostring;
  readonly bits?: int64;
  readonly comparable?: bool;
  readonly length?: int64;
  readonly chanDir?: int64;
  readonly variadic?: bool;
  readonly dynamicType?: GoInterfaceValue["$go$type"];
  readonly elem?: () => Type;
  readonly key?: () => Type;
  readonly fields?: readonly RuntimeStructFieldMetadata[];
  readonly methods?: () => readonly RuntimeMethodMetadata[];
  readonly inputs?: () => readonly Type[];
  readonly outputs?: () => readonly Type[];
}

const runtimeTypeDynamicType = Object.freeze({ comparable: true });
const runtimeTypesByDynamicType = new Map<
  GoInterfaceValue["$go$type"],
  RuntimeType
>();

export class RuntimeType extends GoInterfaceValue implements Type {
  readonly $go$type = runtimeTypeDynamicType;
  readonly $go$methods: ReadonlySet<object>;
  readonly $go$formatString = true;

  constructor(
    private readonly metadata: RuntimeTypeMetadata,
    methodTokens: readonly object[],
  ) {
    super();
    this.$go$methods = new Set(methodTokens);
  }

  $go$implements(contract: readonly object[]): boolean {
    return contract.every((token: object): boolean => this.$go$methods.has(token));
  }

  $go$equal(other: GoInterfaceValue): boolean {
    return this === other;
  }

  $go$hash(): number {
    let hash = 2166136261;
    for (let index = 0; index < this.metadata.identity.length; index++) {
      hash ^= this.metadata.identity.charCodeAt(index);
      hash = Math.imul(hash, 16777619);
    }
    return hash >>> 0;
  }

  $go$format(verb: string): string {
    return verb === "T" ? "reflect.rtype" : this.metadata.text;
  }

  Align(): int64 { return this.metadata.align; }

  AssignableTo(target: Type | undefined): bool {
    return this.identical(target);
  }

  Bits(): int64 {
    const bits = this.metadata.bits ?? 0n;
    if (bits === 0n) {
      return invalidTypeOperation(this.metadata.text, "Bits");
    }
    return bits;
  }

  CanSeq(): bool {
    return [17n, 18n, 21n, 22n, 23n, 24n].includes(this.metadata.kind);
  }

  CanSeq2(): bool {
    return [17n, 21n, 23n, 24n].includes(this.metadata.kind);
  }

  ChanDir(): ChanDir {
    if (this.metadata.kind !== 18n) {
      return invalidTypeOperation(this.metadata.text, "ChanDir");
    }
    return new ChanDir(this.metadata.chanDir ?? 0n);
  }

  Comparable(): bool { return this.metadata.comparable ?? true; }

  ConvertibleTo(target: Type | undefined): bool {
    return this.identical(target);
  }

  Elem(): Type | undefined {
    return this.metadata.elem?.() ?? invalidTypeOperation(this.metadata.text, "Elem");
  }

  Field(index: int64): StructField {
    const fields = this.structFields();
    const ordinal = hostInteger(index);
    const selected = fields[ordinal];
    if (selected === undefined) {
      return invalidTypeOperation(this.metadata.text, "Field");
    }
    return materializeField(selected, ordinal);
  }

  FieldAlign(): int64 { return this.metadata.align; }

  FieldByIndex(index: RuntimeSlice<int64>): StructField {
    let current: Type = this;
    let selected: StructField | undefined;
    for (let position = 0; position < index.length; position++) {
      selected = current.Field(index.get(position));
      if (position + 1 < index.length) {
        current = selected.Type?.Elem() ?? selected.Type ??
          invalidTypeOperation(this.metadata.text, "FieldByIndex");
      }
    }
    return selected ?? invalidTypeOperation(this.metadata.text, "FieldByIndex");
  }

  FieldByName(name: gostring): [StructField, bool] {
    const fields = this.structFields();
    const index = fields.findIndex(
      (field: RuntimeStructFieldMetadata): boolean => field.name === name,
    );
    const selected = fields[index];
    return selected === undefined
      ? [zeroStructField(), false]
      : [materializeField(selected, index), true];
  }

  FieldByNameFunc(
    match: ((name: gostring) => bool) | undefined,
  ): [StructField, bool] {
    if (match === undefined) {
      return invalidTypeOperation(this.metadata.text, "FieldByNameFunc");
    }
    const fields = this.structFields();
    const index = fields.findIndex(
      (field: RuntimeStructFieldMetadata): boolean => match(field.name),
    );
    const selected = fields[index];
    return selected === undefined
      ? [zeroStructField(), false]
      : [materializeField(selected, index), true];
  }

  Fields(): Seq<StructField> {
    const fields = this.structFields();
    return new Seq<StructField>(
      async (yieldValue): Promise<void> => {
        if (yieldValue === undefined) return;
        for (let index = 0; index < fields.length; index++) {
          const field = fields[index];
          if (field !== undefined &&
            !await yieldValue(materializeField(field, index))) return;
        }
      },
    );
  }

  Implements(target: Type | undefined): bool {
    return this.identical(target);
  }

  In(index: int64): Type | undefined { return sequenceAt(this.inputs(), index, "In"); }
  Ins(): Seq<Type | undefined> { return typeSequence(this.inputs()); }
  IsVariadic(): bool { return this.metadata.variadic ?? false; }

  Key(): Type | undefined {
    return this.metadata.key?.() ?? invalidTypeOperation(this.metadata.text, "Key");
  }

  Kind(): Kind { return this.metadata.kind === 0n ? Invalid : new Kind(this.metadata.kind); }

  Len(): int64 {
    if (this.metadata.kind !== 17n) {
      return invalidTypeOperation(this.metadata.text, "Len");
    }
    return this.metadata.length ?? 0n;
  }

  Method(index: int64): Method {
    const method = this.runtimeMethods()[hostInteger(index)];
    if (method === undefined) {
      return invalidTypeOperation(this.metadata.text, "Method");
    }
    return materializeMethod(method);
  }

  MethodByName(name: gostring): [Method, bool] {
    const method = this.runtimeMethods().find(
      (selected: RuntimeMethodMetadata): boolean => selected.name === name,
    );
    return method === undefined
      ? [zeroMethod(), false]
      : [materializeMethod(method), true];
  }

  Methods(): Seq<Method> {
    const methods = this.runtimeMethods();
    return new Seq<Method>(
      async (yieldValue): Promise<void> => {
        if (yieldValue === undefined) return;
        for (const method of methods) {
          if (!await yieldValue(materializeMethod(method))) return;
        }
      },
    );
  }

  Name(): gostring { return this.metadata.name ?? ""; }
  NumField(): int64 { return integerFromHost(this.structFields().length); }
  NumIn(): int64 { return integerFromHost(this.inputs().length); }
  NumMethod(): int64 { return integerFromHost(this.runtimeMethods().length); }
  NumOut(): int64 { return integerFromHost(this.outputs().length); }
  Out(index: int64): Type | undefined { return sequenceAt(this.outputs(), index, "Out"); }
  Outs(): Seq<Type | undefined> { return typeSequence(this.outputs()); }

  OverflowComplex(value: GoComplex128): bool {
    const limit = this.metadata.kind === 15n ? 3.4028234663852886e38 : Number.MAX_VALUE;
    return Math.abs(value.real) > limit || Math.abs(value.imag) > limit;
  }

  OverflowFloat(value: float64): bool {
    return this.metadata.kind === 13n && Math.abs(value) > 3.4028234663852886e38;
  }

  OverflowInt(value: int64): bool {
    const bits = hostInteger(this.Bits());
    return value < -(1n << BigInt(bits - 1)) ||
      value >= (1n << BigInt(bits - 1));
  }

  OverflowUint(value: uint64): bool {
    const bits = hostInteger(this.Bits());
    return value < 0n || value >= (1n << BigInt(bits));
  }

  PkgPath(): gostring { return this.metadata.pkgPath ?? ""; }
  Size(): uint64 { return this.metadata.size; }
  String(): gostring { return this.metadata.text; }

  private inputs(): readonly Type[] { return this.metadata.inputs?.() ?? []; }
  private outputs(): readonly Type[] { return this.metadata.outputs?.() ?? []; }
  private runtimeMethods(): readonly RuntimeMethodMetadata[] {
    return this.metadata.methods?.() ?? [];
  }
  private structFields(): readonly RuntimeStructFieldMetadata[] {
    if (this.metadata.kind !== 25n) {
      return invalidTypeOperation(this.metadata.text, "Field");
    }
    return this.metadata.fields ?? [];
  }

  private identical(target: Type | undefined): target is RuntimeType {
    return target instanceof RuntimeType &&
      this.metadata.identity === target.metadata.identity;
  }
}

export function createRuntimeType(
  metadata: RuntimeTypeMetadata,
  methodTokens: readonly object[],
): Type {
  const result = new RuntimeType(metadata, methodTokens);
  if (metadata.dynamicType !== undefined) {
    runtimeTypesByDynamicType.set(metadata.dynamicType, result);
  }
  return result;
}

export function runtimeTypeOf(
  value: GoInterfaceValue | undefined,
): Type | undefined {
  return value === undefined
    ? undefined
    : runtimeTypesByDynamicType.get(value.$go$type);
}

function materializeField(
  field: RuntimeStructFieldMetadata,
  ordinal: number,
): StructField {
  return new StructField({
    Name: field.name,
    PkgPath: field.pkgPath ?? "",
    Type: field.type(),
    Tag: new StructTag(field.tag ?? ""),
    Offset: field.offset ?? 0n,
    Index: RuntimeSlice.literal([...(field.index ?? [integerFromHost(ordinal)])]),
    Anonymous: field.anonymous ?? false,
  });
}

function materializeMethod(method: RuntimeMethodMetadata): Method {
  return new Method({
    Name: method.name,
    PkgPath: method.pkgPath,
    Type: method.type(),
    Func: invalidRuntimeValue,
    Index: method.index,
  });
}

class InvalidRuntimeValue extends Value {
  constructor() { super(); }
}

const invalidRuntimeValue = new InvalidRuntimeValue();

function zeroStructField(): StructField {
  return new StructField({
    Name: "",
    PkgPath: "",
    Type: undefined,
    Tag: new StructTag(""),
    Offset: 0n,
    Index: RuntimeSlice.nil<int64>(),
    Anonymous: false,
  });
}

function zeroMethod(): Method {
  return new Method({ Name: "", PkgPath: "", Type: undefined, Func: invalidRuntimeValue, Index: 0n });
}

function sequenceAt(
  values: readonly Type[],
  index: int64,
  operation: string,
): Type {
  const selected = values[hostInteger(index)];
  return selected ?? invalidTypeOperation("func", operation);
}

function typeSequence(values: readonly Type[]): Seq<Type | undefined> {
  return new Seq<Type | undefined>(
    async (yieldValue): Promise<void> => {
      if (yieldValue === undefined) return;
      for (const value of values) {
        if (!await yieldValue(value)) return;
      }
    },
  );
}

function invalidTypeOperation(type: gostring, operation: string): never {
  return GoPanic.raise(
    new ProviderError(`reflect: ${operation} of non-${type} type`),
  );
}
