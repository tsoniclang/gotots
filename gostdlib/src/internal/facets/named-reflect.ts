import type { gostring, int64, uint64 } from "@gotots/runtime/scalars.js";

import {
  ChanDir,
  Kind,
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
} from "../portable/reflect/runtime-type.js";

export class ReflectTypeMetadataOperations {
  static $create(
    metadata: RuntimeTypeMetadata,
    methodTokens: readonly object[],
  ): Type {
    return createRuntimeType(metadata, methodTokens);
  }

  static $typeOf(value: GoInterfaceValue | undefined): Type | undefined {
    return runtimeTypeOf(value);
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
