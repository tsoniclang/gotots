import { RuntimeSlice } from "@gotots/runtime/slice.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import type {
  bool,
  float64,
  gostring,
  int64,
  uint64,
} from "@gotots/gostdlib/internal/scalars.js";
import { readMetric } from "../../node/runtime/metrics.js";

export class ValueKind {
  constructor(readonly value: int64) {}
}

export const KindUint64 = new ValueKind(1n);
export const KindFloat64 = new ValueKind(2n);
export const KindFloat64Histogram = new ValueKind(3n);
const kindBad = new ValueKind(0n);

export class Value {
  constructor(
    private readonly kind: ValueKind = kindBad,
    private readonly floatValue: float64 = 0,
    private readonly uintValue: uint64 = 0n,
  ) {}

  static FromFloat64(value: float64): Value {
    return new Value(KindFloat64, value, 0n);
  }

  static FromUint64(value: uint64): Value {
    return new Value(KindUint64, 0, value);
  }

  Float64(): float64 {
    if (this.kind.value !== KindFloat64.value) {
      GoPanic.raiseRuntime("called Float64 on non-float64 metric value");
    }
    return this.floatValue;
  }

  Kind(): ValueKind {
    return this.kind;
  }

  Uint64(): uint64 {
    if (this.kind.value !== KindUint64.value) {
      GoPanic.raiseRuntime("called Uint64 on non-uint64 metric value");
    }
    return this.uintValue;
  }
}

export class Description {
  constructor(
    public Name: gostring = "",
    public Description: gostring = "",
    public Kind: ValueKind = kindBad,
    public Cumulative: bool = false,
  ) {}
}

export class Sample {
  Name: gostring;
  Value: Value;

  constructor(
    name: gostring = "",
    value: Value = new Value(),
  ) {
    this.Name = name;
    this.Value = value;
  }
}

const descriptions: readonly Description[] = [
  uintMetric("/memory/classes/total:bytes", "Process memory attributed by the provider.", false),
  uintMetric("/memory/classes/heap/objects:bytes", "Live JavaScript heap bytes.", false),
  uintMetric("/memory/classes/heap/free:bytes", "Reserved but unused JavaScript heap bytes.", false),
  uintMetric("/memory/classes/heap/released:bytes", "Heap bytes released to the host.", false),
  uintMetric("/memory/classes/heap/stacks:bytes", "Host stack bytes visible to the provider.", false),
  uintMetric("/gc/gomemlimit:bytes", "Host JavaScript heap limit.", false),
  uintMetric("/gc/gogc:percent", "Provider garbage-collection target percentage.", false),
  uintMetric("/gc/heap/goal:bytes", "Host JavaScript heap target.", false),
  uintMetric("/gc/heap/live:bytes", "Live JavaScript heap bytes.", false),
  uintMetric("/gc/heap/objects:objects", "Heap object count visible to the provider.", false),
  uintMetric("/gc/scan/heap:bytes", "Heap bytes visible to the host collector.", false),
  uintMetric("/sched/gomaxprocs:threads", "Provider scheduler execution threads.", false),
  uintMetric("/sched/goroutines:goroutines", "Provider scheduler root task count.", false),
  uintMetric("/gc/cycles/total:gc-cycles", "Host collection cycles exposed to the provider.", true),
  floatMetric("/cpu/classes/gc/total:cpu-seconds", "Host collection CPU seconds."),
  floatMetric("/cpu/classes/user:cpu-seconds", "Process CPU seconds."),
  floatMetric("/cpu/classes/total:cpu-seconds", "Total process CPU seconds."),
];

export function All(): RuntimeSlice<Description> {
  return RuntimeSlice.literal(descriptions.map((entry) => new Description(
    entry.Name,
    entry.Description,
    entry.Kind,
    entry.Cumulative,
  )));
}

export function Read(m: RuntimeSlice<Sample>): void {
  for (let index = 0; index < m.length; index += 1) {
    const sample = m.get(index);
    const reading = readMetric(sample.Name);
    switch (reading.kind) {
      case "uint64":
        sample.Value = Value.FromUint64(reading.value);
        break;
      case "float64":
        sample.Value = Value.FromFloat64(reading.value);
        break;
      case "missing":
        sample.Value = new Value();
        break;
    }
  }
}

function uintMetric(
  name: string,
  description: string,
  cumulative: boolean,
): Description {
  return new Description(name, description, KindUint64, cumulative);
}

function floatMetric(name: string, description: string): Description {
  return new Description(name, description, KindFloat64, true);
}
