import type { int8 } from "@gotots/gostdlib/internal/scalars.js";

import { Accuracy, Float, Int } from "../../math/big.js";

export class MathBigAccuracyValueOperations {
  static $project(source: Accuracy): int8 {
    return source.value;
  }

  static $wrap(source: int8): Accuracy {
    return new Accuracy(source);
  }
}

export type MathBigFloatStorage = Float;

export class MathBigFloatOperations {
  static $zero(): Float {
    return new Float();
  }

  static $storageOf(source: Float): MathBigFloatStorage {
    return source;
  }

  static $fromStorage(source: MathBigFloatStorage): Float {
    return source;
  }
}

export type MathBigIntStorage = Int;

export class MathBigIntOperations {
  static $zero(): Int {
    return new Int();
  }

  static $storageOf(source: Int): MathBigIntStorage {
    return source;
  }

  static $fromStorage(source: MathBigIntStorage): Int {
    return source;
  }
}
