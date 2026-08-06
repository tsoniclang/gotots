import { GoArray } from "@gotots/runtime/array.js";

import { MemStats } from "../../runtime.js";
import type { MemStatsBySize } from "../portable/runtime/mem-stats.js";

export type RuntimeMemStatsStorage = MemStats;

export class RuntimeMemStatsOperations {
  static $zero(): MemStats {
    return new MemStats();
  }

  static $copy(source: MemStats): MemStats {
    const target = new MemStats();
    target.Alloc = source.Alloc;
    target.TotalAlloc = source.TotalAlloc;
    target.Sys = source.Sys;
    target.Lookups = source.Lookups;
    target.Mallocs = source.Mallocs;
    target.Frees = source.Frees;
    target.HeapAlloc = source.HeapAlloc;
    target.HeapSys = source.HeapSys;
    target.HeapIdle = source.HeapIdle;
    target.HeapInuse = source.HeapInuse;
    target.HeapReleased = source.HeapReleased;
    target.HeapObjects = source.HeapObjects;
    target.StackInuse = source.StackInuse;
    target.StackSys = source.StackSys;
    target.MSpanInuse = source.MSpanInuse;
    target.MSpanSys = source.MSpanSys;
    target.MCacheInuse = source.MCacheInuse;
    target.MCacheSys = source.MCacheSys;
    target.BuckHashSys = source.BuckHashSys;
    target.GCSys = source.GCSys;
    target.OtherSys = source.OtherSys;
    target.NextGC = source.NextGC;
    target.LastGC = source.LastGC;
    target.PauseTotalNs = source.PauseTotalNs;
    target.PauseNs = source.PauseNs.copy();
    target.PauseEnd = source.PauseEnd.copy();
    target.NumGC = source.NumGC;
    target.NumForcedGC = source.NumForcedGC;
    target.GCCPUFraction = source.GCCPUFraction;
    target.EnableGC = source.EnableGC;
    target.DebugGC = source.DebugGC;
    const bySize = GoArray.zero<MemStatsBySize, 61>(
      61,
      { Size: 0, Mallocs: 0n, Frees: 0n },
    );
    for (let index = 0; index < 61; index += 1) {
      const entry = source.BySize.get(index);
      bySize.set(index, {
        Size: entry.Size,
        Mallocs: entry.Mallocs,
        Frees: entry.Frees,
      });
    }
    target.BySize = bySize;
    return target;
  }

  static $storageOf(source: MemStats): RuntimeMemStatsStorage {
    return source;
  }

  static $fromStorage(source: RuntimeMemStatsStorage): MemStats {
    return source;
  }
}
