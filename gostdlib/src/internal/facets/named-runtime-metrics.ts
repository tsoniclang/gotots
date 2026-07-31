import type { gostring, int64 } from "@gotots/runtime/scalars.js";

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
}

export class RuntimeMetricsValueOperations {
  static $zero(): Value {
    return new Value();
  }
}
