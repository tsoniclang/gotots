import { gzipSync } from "node:zlib";
import type { MemorySnapshot } from "../../../portable/runtime/mem-stats.js";

const nanosecondsPerMillisecond = 1_000_000n;
const nanosecondsPerSecond = 1_000_000_000;

export function cpuProfile(
  startedAt: number,
  stoppedAt: number,
  cpuSeconds: number,
): Uint8Array {
  const strings = ["", "cpu", "nanoseconds"];
  const profile = new Proto();
  profile.message(1, valueType(1, 2));
  profile.message(2, sample([Math.round(cpuSeconds * nanosecondsPerSecond)]));
  writeStrings(profile, strings);
  profile.integer(9, BigInt(startedAt) * nanosecondsPerMillisecond);
  profile.integer(10, BigInt(stoppedAt - startedAt) * nanosecondsPerMillisecond);
  profile.message(11, valueType(1, 2));
  profile.integer(12, 1);
  profile.integer(14, 1);
  return gzipSync(profile.finish());
}

export function memoryProfile(
  name: string,
  memory: MemorySnapshot,
): Uint8Array {
  const strings = [
    "",
    name === "allocs" ? "alloc_space" : "inuse_space",
    "bytes",
    name === "allocs" ? "alloc_objects" : "inuse_objects",
    "count",
  ];
  const profile = new Proto();
  profile.message(1, valueType(1, 2));
  profile.message(1, valueType(3, 4));
  profile.message(2, sample([memory.heapUsed, 0]));
  writeStrings(profile, strings);
  profile.integer(9, BigInt(Date.now()) * nanosecondsPerMillisecond);
  profile.integer(14, 1);
  return gzipSync(profile.finish());
}

export function stackProfile(stack: Uint8Array): Uint8Array {
  const profile = new Proto();
  profile.message(1, valueType(1, 2));
  profile.message(2, sample([1]));
  const strings = ["", "goroutines", "count", new TextDecoder().decode(stack)];
  writeStrings(profile, strings);
  profile.integer(9, BigInt(Date.now()) * nanosecondsPerMillisecond);
  profile.integer(13, 3);
  profile.integer(14, 1);
  return gzipSync(profile.finish());
}

export function emptyProfile(name: string): Uint8Array {
  const profile = new Proto();
  profile.message(1, valueType(1, 2));
  writeStrings(profile, ["", name, "count"]);
  profile.integer(9, BigInt(Date.now()) * nanosecondsPerMillisecond);
  profile.integer(14, 1);
  return gzipSync(profile.finish());
}

function valueType(type: number, unit: number): Uint8Array {
  const value = new Proto();
  value.integer(1, type);
  value.integer(2, unit);
  return value.finish();
}

function sample(values: readonly number[]): Uint8Array {
  const result = new Proto();
  result.packedIntegers(2, values);
  return result.finish();
}

function writeStrings(profile: Proto, values: readonly string[]): void {
  for (const value of values) {
    profile.bytes(6, new TextEncoder().encode(value));
  }
}

class Proto {
  readonly #bytes: number[] = [];

  integer(field: number, value: number | bigint): void {
    this.#tag(field, 0);
    this.#varint(typeof value === "bigint" ? value : BigInt(Math.trunc(value)));
  }

  bytes(field: number, value: Uint8Array): void {
    this.#tag(field, 2);
    this.#varint(BigInt(value.length));
    this.#bytes.push(...value);
  }

  message(field: number, value: Uint8Array): void {
    this.bytes(field, value);
  }

  packedIntegers(field: number, values: readonly number[]): void {
    const packed = new Proto();
    for (const value of values) {
      packed.#varint(BigInt(Math.trunc(value)));
    }
    this.bytes(field, packed.finish());
  }

  finish(): Uint8Array {
    return Uint8Array.from(this.#bytes);
  }

  #tag(field: number, wire: number): void {
    this.#varint(BigInt(field * 8 + wire));
  }

  #varint(value: bigint): void {
    let remaining = value < 0 ? BigInt.asUintN(64, value) : value;
    while (remaining >= 0x80n) {
      this.#bytes.push(Number(remaining & 0x7fn) | 0x80);
      remaining >>= 7n;
    }
    this.#bytes.push(Number(remaining));
  }
}
