import type { uint64 } from "@gotots/gostdlib/internal/scalars.js";

import { cpuSeconds, memorySnapshot } from "./process.js";

export type MetricReading =
  | { readonly kind: "uint64"; readonly value: uint64 }
  | { readonly kind: "float64"; readonly value: number }
  | { readonly kind: "missing" };

export function readMetric(name: string): MetricReading {
  const memory = memorySnapshot();
  switch (name) {
    case "/memory/classes/total:bytes":
      return { kind: "uint64", value: memory.rss + memory.external + memory.arrayBuffers };
    case "/memory/classes/heap/objects:bytes":
    case "/gc/heap/live:bytes":
    case "/gc/scan/heap:bytes":
      return { kind: "uint64", value: memory.heapUsed };
    case "/memory/classes/heap/free:bytes":
      return {
        kind: "uint64",
        value: memory.heapTotal > memory.heapUsed
          ? memory.heapTotal - memory.heapUsed
          : 0n,
      };
    case "/memory/classes/heap/released:bytes":
    case "/memory/classes/heap/stacks:bytes":
    case "/gc/heap/objects:objects":
    case "/gc/cycles/total:gc-cycles":
      return { kind: "uint64", value: 0n };
    case "/gc/gomemlimit:bytes":
    case "/gc/heap/goal:bytes":
      return { kind: "uint64", value: memory.heapLimit };
    case "/gc/gogc:percent":
      return { kind: "uint64", value: 100n };
    case "/sched/gomaxprocs:threads":
    case "/sched/goroutines:goroutines":
      return { kind: "uint64", value: 1n };
    case "/cpu/classes/gc/total:cpu-seconds":
      return { kind: "float64", value: 0 };
    case "/cpu/classes/user:cpu-seconds":
    case "/cpu/classes/total:cpu-seconds":
      return { kind: "float64", value: cpuSeconds() };
    default:
      return { kind: "missing" };
  }
}
