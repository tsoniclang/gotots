import type { gostring, int64, uint64 } from "@gotots/gostdlib/internal/scalars.js";

import {
  ChanDir,
  Kind,
  MapIter,
  StructField,
  StructTag,
  Value,
} from "../../reflect.js";
import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import type { Type } from "../../reflect.js";
import {
  createRuntimeType,
  runtimeTypeOf,
  type RuntimeTypeMetadata,
  type RuntimeTypeRegistration,
} from "../portable/reflect/runtime-type.js";
import {
  registerRuntimeOpaqueStructValueOperations,
  registerRuntimePointerValueOperations,
  registerRuntimeStructValueOperations,
  registerRuntimeValueOperations,
  type RuntimePointerValueOperations,
  type RuntimeStructFieldOperations,
  type RuntimeValueAdapter,
  type RuntimeValueOperations,
} from "../portable/reflect/runtime-value.js";

export class ReflectTypeMetadataOperations {
  static $create(
    metadata: () => RuntimeTypeMetadata,
    methodTokens: () => readonly object[],
    registration?: RuntimeTypeRegistration,
  ): Type {
    return createRuntimeType(metadata, methodTokens, registration);
  }

  static $typeOf(value: GoInterfaceValue | undefined): Type | undefined {
    return runtimeTypeOf(value);
  }

  static $registerValue(
    type: Type,
    operations: () => RuntimeValueOperations,
  ): void {
    registerRuntimeValueOperations(type, operations);
  }

  static $registerStruct<T>(
    type: Type,
    adapter: RuntimeValueAdapter<T>,
    fields: readonly RuntimeStructFieldOperations<T>[],
    clone?: (value: T) => T,
  ): void {
    registerRuntimeStructValueOperations(type, adapter, fields, clone);
  }

  static $registerOpaqueStruct<T>(
    type: Type,
    adapter: RuntimeValueAdapter<T>,
    unavailableFields: readonly gostring[],
  ): void {
    registerRuntimeOpaqueStructValueOperations(
      type,
      adapter,
      unavailableFields,
    );
  }

  static $registerPointer<P>(
    type: Type,
    adapter: RuntimeValueAdapter<P | undefined>,
    descriptor: RuntimePointerValueOperations<P>,
  ): void {
    registerRuntimePointerValueOperations(type, adapter, descriptor);
  }
}

export class ReflectChanDirValueOperations {
  static $project(source: ChanDir): int64 {
    return source.value;
  }

  static $wrap(source: int64): ChanDir {
    return new ChanDir(source);
  }
}

export class ReflectKindValueOperations {
  static $project(source: Kind): uint64 {
    return source.value;
  }

  static $wrap(source: uint64): Kind {
    return new Kind(source);
  }
}

export class ReflectStructTagValueOperations {
  static $project(source: StructTag): gostring {
    return source.value;
  }

  static $wrap(source: gostring): StructTag {
    return new StructTag(source);
  }
}

export class ReflectStructFieldOperations {
  static $copy(source: StructField): StructField {
    return new StructField({
      Name: source.Name,
      PkgPath: source.PkgPath,
      Type: source.Type,
      Tag: source.Tag,
      Offset: source.Offset,
      Index: source.Index,
      Anonymous: source.Anonymous,
    });
  }
}

export class ReflectMapIterOperations {
  static $copy(source: MapIter): MapIter {
    return source.$copy();
  }

  static $assign(target: MapIter, source: MapIter): void {
    if (target === source) {
      return;
    }
    target.position = source.position;
    target.keys = source.keys;
    target.valueAt = source.valueAt;
  }
}

class InvalidValue extends Value {
  constructor() {
    super();
  }
}

export class ReflectValueOperations {
  static $zero(): Value {
    return new InvalidValue();
  }

  static $copy(source: Value): Value {
    return source;
  }
}
