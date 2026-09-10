import type { gostring, int64 } from "@gotots/gostdlib/internal/scalars.js";

import {
  Description,
  Sample,
  Value,
  ValueKind,
} from "../../runtime/metrics.js";

export class RuntimeMetricsValueKindValueOperations {
  static $project(source: ValueKind): int64 {
    return source.value;
  }

  static $wrap(source: int64): ValueKind {
    return new ValueKind(source);
  }
}

export class RuntimeMetricsDescriptionOperations {
  static $copy(source: Description): Description {
    return new Description(
      source.Name,
      source.Description,
      source.Kind,
      source.Cumulative,
    );
  }

  static $assign(target: Description, source: Description): void {
    target.Name = source.Name;
    target.Description = source.Description;
    target.Kind = source.Kind;
    target.Cumulative = source.Cumulative;
  }
}

export class RuntimeMetricsSampleOperations {
  static $make(name: gostring, value: Value): Sample {
    return new Sample(name, value);
  }

  static $zero(): Sample {
    return new Sample();
  }

  static $copy(source: Sample): Sample {
    return new Sample(source.Name, source.Value);
  }

  static $assign(target: Sample, source: Sample): void {
    target.Name = source.Name;
    Value.$assign(target.Value, source.Value);
  }
}

export class RuntimeMetricsValueOperations {
  static $zero(): Value {
    return new Value();
  }

  static $copy(source: Value): Value {
    return Value.$copy(source);
  }

  static $assign(target: Value, source: Value): void {
    Value.$assign(target, source);
  }
}
