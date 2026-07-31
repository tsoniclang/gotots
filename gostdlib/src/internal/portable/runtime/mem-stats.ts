import { GoArray } from "@gotots/runtime/array.js";
import type {
  bool,
  float64,
  uint32,
  uint64,
} from "@gotots/runtime/scalars.js";

export interface MemStatsBySize {
  Size: uint32;
  Mallocs: uint64;
  Frees: uint64;
}

export interface MemorySnapshot {
  readonly rss: uint64;
  readonly heapTotal: uint64;
  readonly heapUsed: uint64;
  readonly external: uint64;
  readonly arrayBuffers: uint64;
  readonly heapLimit: uint64;
  readonly malloced: uint64;
}

export class MemStats {
  Alloc: uint64 = 0;
  TotalAlloc: uint64 = 0;
  Sys: uint64 = 0;
  Lookups: uint64 = 0;
  Mallocs: uint64 = 0;
  Frees: uint64 = 0;
  HeapAlloc: uint64 = 0;
  HeapSys: uint64 = 0;
  HeapIdle: uint64 = 0;
  HeapInuse: uint64 = 0;
  HeapReleased: uint64 = 0;
  HeapObjects: uint64 = 0;
  StackInuse: uint64 = 0;
  StackSys: uint64 = 0;
  MSpanInuse: uint64 = 0;
  MSpanSys: uint64 = 0;
  MCacheInuse: uint64 = 0;
  MCacheSys: uint64 = 0;
  BuckHashSys: uint64 = 0;
  GCSys: uint64 = 0;
  OtherSys: uint64 = 0;
  NextGC: uint64 = 0;
  LastGC: uint64 = 0;
  PauseTotalNs: uint64 = 0;
  PauseNs = GoArray.zero<uint64, 256>(256, 0);
  PauseEnd = GoArray.zero<uint64, 256>(256, 0);
  NumGC: uint32 = 0;
  NumForcedGC: uint32 = 0;
  GCCPUFraction: float64 = 0;
  EnableGC: bool = true;
  DebugGC: bool = false;
  BySize = zeroBySize();
}

export function populateMemStats(target: MemStats, source: MemorySnapshot): void {
  target.Alloc = source.heapUsed;
  target.TotalAlloc = source.heapTotal;
  target.Sys = source.rss + source.external + source.arrayBuffers;
  target.HeapAlloc = source.heapUsed;
  target.HeapSys = source.heapTotal;
  target.HeapIdle = Math.max(0, source.heapTotal - source.heapUsed);
  target.HeapInuse = source.heapUsed;
  target.MSpanInuse = source.malloced;
  target.MSpanSys = source.malloced;
  target.OtherSys = source.external + source.arrayBuffers;
  target.NextGC = source.heapLimit;
}

function zeroBySize(): GoArray<MemStatsBySize, 61> {
  const indexes: number[] = [];
  const values: MemStatsBySize[] = [];
  for (let index = 0; index < 61; index += 1) {
    indexes.push(index);
    values.push({ Size: 0, Mallocs: 0, Frees: 0 });
  }
  return GoArray.literal<MemStatsBySize, 61>(
    61,
    { Size: 0, Mallocs: 0, Frees: 0 },
    indexes,
    values,
  );
}
