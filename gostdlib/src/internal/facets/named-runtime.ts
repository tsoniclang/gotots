import { MemStats } from "../../runtime.js";

export type RuntimeMemStatsStorage = MemStats;

export class RuntimeMemStatsOperations {
  static $zero(): MemStats {
    return new MemStats();
  }

  static $copy(source: MemStats): MemStats {
    const target = new MemStats();
    RuntimeMemStatsOperations.$assign(target, source);
    return target;
  }

  static $assign(target: MemStats, source: MemStats): void {
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
    for (let index = 0; index < 256; index++) {
      target.PauseNs.set(index, source.PauseNs.get(index));
      target.PauseEnd.set(index, source.PauseEnd.get(index));
    }
    target.NumGC = source.NumGC;
    target.NumForcedGC = source.NumForcedGC;
    target.GCCPUFraction = source.GCCPUFraction;
    target.EnableGC = source.EnableGC;
    target.DebugGC = source.DebugGC;
    for (let index = 0; index < 61; index += 1) {
      const entry = source.BySize.get(index);
      const destination = target.BySize.get(index);
      destination.Size = entry.Size;
      destination.Mallocs = entry.Mallocs;
      destination.Frees = entry.Frees;
    }
  }

  static $storageOf(source: MemStats): RuntimeMemStatsStorage {
    return source;
  }

  static $fromStorage(source: RuntimeMemStatsStorage): MemStats {
    return source;
  }
}
