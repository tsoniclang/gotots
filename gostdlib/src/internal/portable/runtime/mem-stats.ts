import { GoArray } from "@gotots/runtime/array.js";
import type {
  bool,
  float64,
  uint32,
  uint64,
} from "@gotots/gostdlib/internal/scalars.js";

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
  Alloc: uint64 = 0n;
  TotalAlloc: uint64 = 0n;
  Sys: uint64 = 0n;
  Lookups: uint64 = 0n;
  Mallocs: uint64 = 0n;
  Frees: uint64 = 0n;
  HeapAlloc: uint64 = 0n;
  HeapSys: uint64 = 0n;
  HeapIdle: uint64 = 0n;
  HeapInuse: uint64 = 0n;
  HeapReleased: uint64 = 0n;
  HeapObjects: uint64 = 0n;
  StackInuse: uint64 = 0n;
  StackSys: uint64 = 0n;
  MSpanInuse: uint64 = 0n;
  MSpanSys: uint64 = 0n;
  MCacheInuse: uint64 = 0n;
  MCacheSys: uint64 = 0n;
  BuckHashSys: uint64 = 0n;
  GCSys: uint64 = 0n;
  OtherSys: uint64 = 0n;
  NextGC: uint64 = 0n;
  LastGC: uint64 = 0n;
  PauseTotalNs: uint64 = 0n;
  PauseNs: GoArray<uint64, 256> = GoArray.zero<uint64, 256>(256, 0n);
  PauseEnd: GoArray<uint64, 256> = GoArray.zero<uint64, 256>(256, 0n);
  NumGC: uint32 = 0;
  NumForcedGC: uint32 = 0;
  GCCPUFraction: float64 = 0;
  EnableGC: bool = false;
  DebugGC: bool = false;
  BySize: GoArray<MemStatsBySize, 61> = zeroBySize();
}

export function populateMemStats(target: MemStats, source: MemorySnapshot): void {
  target.Alloc = source.heapUsed;
  target.TotalAlloc = source.heapTotal;
  target.Sys = source.rss + source.external + source.arrayBuffers;
  target.HeapAlloc = source.heapUsed;
  target.HeapSys = source.heapTotal;
  target.HeapIdle = source.heapTotal > source.heapUsed
    ? source.heapTotal - source.heapUsed
    : 0n;
  target.HeapInuse = source.heapUsed;
  target.MSpanInuse = source.malloced;
  target.MSpanSys = source.malloced;
  target.OtherSys = source.external + source.arrayBuffers;
  target.NextGC = source.heapLimit;
  target.EnableGC = true;
}

function zeroBySize(): GoArray<MemStatsBySize, 61> {
  const indexes: number[] = [];
  const values: MemStatsBySize[] = [];
  for (let index = 0; index < 61; index += 1) {
    indexes.push(index);
    values.push({ Size: 0, Mallocs: 0n, Frees: 0n });
  }
  return GoArray.literal<MemStatsBySize, 61>(
    61,
    { Size: 0, Mallocs: 0n, Frees: 0n },
    indexes,
    values,
  );
}
