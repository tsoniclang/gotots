import { getHeapStatistics } from "node:v8";
import type {
  gostring,
  int64,
  uint64,
} from "@gotots/gostdlib/internal/scalars.js";
import {
  hostInteger,
  integerFromHost,
  unsignedIntegerFromHost,
} from "../../host-integer.js";
import type { MemorySnapshot } from "../../portable/runtime/mem-stats.js";

export function goOperatingSystem(): gostring {
  switch (process.platform) {
    case "win32":
      return "windows";
    case "aix":
    case "android":
    case "darwin":
    case "freebsd":
    case "linux":
    case "netbsd":
    case "openbsd":
      return process.platform;
    case "sunos":
      return "solaris";
    case "cygwin":
      return "windows";
    case "haiku":
      throw new RangeError("the selected Go toolchain has no haiku GOOS");
  }
}

export function goArchitecture(): gostring {
  switch (process.arch) {
    case "x64":
      return "amd64";
    case "ia32":
      return "386";
    case "arm":
    case "arm64":
    case "loong64":
    case "mips":
    case "ppc64":
    case "riscv64":
    case "s390x":
      return process.arch;
    case "mipsel":
      return "mipsle";
  }
}

export function memorySnapshot(): MemorySnapshot {
  const memory = process.memoryUsage();
  const heap = getHeapStatistics();
  return {
    rss: unsignedIntegerFromHost(memory.rss),
    heapTotal: unsignedIntegerFromHost(memory.heapTotal),
    heapUsed: unsignedIntegerFromHost(memory.heapUsed),
    external: unsignedIntegerFromHost(memory.external),
    arrayBuffers: unsignedIntegerFromHost(memory.arrayBuffers),
    heapLimit: unsignedIntegerFromHost(heap.heap_size_limit),
    malloced: unsignedIntegerFromHost(heap.malloced_memory),
  };
}

export function caller(skip: int64): [uint64, gostring, int64, boolean] {
  const stack = new Error().stack;
  if (stack === undefined) {
    return [0n, "", 0n, false];
  }
  const frames = stack.split("\n").slice(2);
  const frame = frames[Math.max(0, hostInteger(skip)) + 1];
  if (frame === undefined) {
    return [0n, "", 0n, false];
  }
  const match = /\(?(.+):(\d+):(\d+)\)?$/u.exec(frame.trim());
  if (match === null) {
    return [0n, "", 0n, false];
  }
  const file = match[1]?.replaceAll("\\", "/") ?? "";
  const line = Number(match[2] ?? 0);
  return [0n, file, integerFromHost(line), file !== "" && line > 0];
}

export function stackBytes(): Uint8Array {
  return new TextEncoder().encode(new Error().stack ?? "");
}

export function cpuSeconds(): number {
  const usage = process.cpuUsage();
  return (usage.user + usage.system) / 1_000_000;
}

export function isNodeTest(): boolean {
  return process.env.NODE_TEST_CONTEXT !== undefined
    || process.execArgv.includes("--test");
}
