import { MemStats } from "../../runtime.js";

export type RuntimeMemStatsStorage = MemStats;

export class RuntimeMemStatsOperations {
  static $zero(): MemStats {
    return new MemStats();
  }

  static $storageOf(source: MemStats): RuntimeMemStatsStorage {
    return source;
  }

  static $fromStorage(source: RuntimeMemStatsStorage): MemStats {
    return source;
  }
}
