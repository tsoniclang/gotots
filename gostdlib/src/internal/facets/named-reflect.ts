import type { gostring, int64, uint64 } from "@gotots/runtime/scalars.js";

import { ChanDir, Kind, StructTag, Value } from "../../reflect.js";

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
