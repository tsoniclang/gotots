import { GoArray } from "@gotots/runtime/array.js";
import type { GoError } from "@gotots/runtime/interface-value.js";
import { GoPointer } from "@gotots/runtime/pointer.js";
import type {
  bool,
  gostring,
  int64,
  uint8,
  uint64,
} from "@gotots/runtime/scalars.js";
import type { RuntimeSlice } from "@gotots/runtime/slice.js";

const laneBase = 67_108_864;
const utf8Encoder = new TextEncoder();

type HashState = {
  h1: number;
  h2: number;
  h3: number;
  h4: number;
  length: number;
  seed: number;
};

function makeState(seed: number): HashState {
  const low = seed >>> 0;
  const high = Math.floor(seed / 4_294_967_296) >>> 0;
  return {
    h1: (0x9e3779b1 ^ low) >>> 0,
    h2: (0x85ebca77 ^ high) >>> 0,
    h3: (0xc2b2ae3d ^ low ^ high) >>> 0,
    h4: (0x27d4eb2f ^ Math.imul(low, 0x165667b1)) >>> 0,
    length: 0,
    seed,
  };
}

function resetState(target: HashState): void {
  const initial = makeState(target.seed);
  target.h1 = initial.h1;
  target.h2 = initial.h2;
  target.h3 = initial.h3;
  target.h4 = initial.h4;
  target.length = 0;
}

function mixByte(target: HashState, input: number): void {
  const value = input & 0xff;
  const index = target.length;
  target.h1 = Math.imul(target.h1 ^ value, 0x01000193) >>> 0;
  target.h2 = Math.imul(target.h2 ^ ((value + index) & 0xff), 0x85ebca6b) >>> 0;
  target.h3 = Math.imul(target.h3 ^ ((value + (index >>> 8)) & 0xff), 0xc2b2ae35) >>> 0;
  target.h4 = Math.imul(target.h4 ^ (value + ((index * 17) & 0xff)), 0x27d4eb2d) >>> 0;
  target.length = index + 1;
}

function avalanche(input: number): number {
  let value = input >>> 0;
  value ^= value >>> 16;
  value = Math.imul(value, 0x7feb352d) >>> 0;
  value ^= value >>> 15;
  value = Math.imul(value, 0x846ca68b) >>> 0;
  return (value ^ (value >>> 16)) >>> 0;
}

function digest(target: HashState): Uint128 {
  const h1 = avalanche(target.h1 ^ target.length);
  const h2 = avalanche(target.h2 ^ Math.imul(target.length, 0x9e3779b1));
  const h3 = avalanche(target.h3 ^ Math.imul(target.length, 0x85ebca77));
  const h4 = avalanche(target.h4 ^ Math.imul(target.length, 0xc2b2ae3d));
  return Uint128.$make(
    (h1 & 0x03ffffff) * laneBase + (h2 & 0x03ffffff),
    (h3 & 0x03ffffff) * laneBase + (h4 & 0x03ffffff),
  );
}

function hashSlice(value: RuntimeSlice<uint8>, seed: uint64): Uint128 {
  const state = makeState(seed);
  for (let index = 0; index < value.length; index++) {
    mixByte(state, value.get(index));
  }
  return digest(state);
}

function hashString(value: gostring, seed: uint64): Uint128 {
  const state = makeState(seed);
  mixString(state, value);
  return digest(state);
}

function mixString(target: HashState, value: gostring): number {
  const bytes = utf8Encoder.encode(value);
  for (let index = 0; index < bytes.length; index++) {
    mixByte(target, bytes[index]!);
  }
  return bytes.length;
}

function encodeWord(value: number): number[] {
  const result = new Array<number>(8);
  let remaining = value;
  for (let index = 7; index >= 0; index--) {
    result[index] = remaining % 256;
    remaining = Math.floor(remaining / 256);
  }
  return result;
}

export type Uint128$Storage = {
  Hi: uint64;
  Lo: uint64;
};

export class Uint128 {
  declare private readonly $goType: void;

  private constructor(private readonly $storage: Uint128$Storage) {}

  public static $make(hi: uint64, lo: uint64): Uint128 {
    return new Uint128({ Hi: hi, Lo: lo });
  }

  public static $storageOf(source: Uint128): Uint128$Storage {
    return source.$storage;
  }

  public static $fromStorage(source: Uint128$Storage): Uint128 {
    return new Uint128(source);
  }

  public get Hi(): uint64 {
    return this.$storage.Hi;
  }

  public set Hi(value: uint64) {
    this.$storage.Hi = value;
  }

  public get Lo(): uint64 {
    return this.$storage.Lo;
  }

  public set Lo(value: uint64) {
    this.$storage.Lo = value;
  }

  public static $zero(): Uint128 {
    return Uint128.$make(0, 0);
  }

  public static $copy(source: Uint128): Uint128 {
    return Uint128.$make(source.Hi, source.Lo);
  }

  public static $equal(left: Uint128, right: Uint128): bool {
    return left.Hi === right.Hi && left.Lo === right.Lo;
  }

  public static $hash(source: Uint128): number {
    return avalanche(
      (source.Hi >>> 0) ^
        Math.floor(source.Hi / 4_294_967_296) ^
        (source.Lo >>> 0) ^
        Math.floor(source.Lo / 4_294_967_296),
    );
  }

  public Bytes(): GoArray<uint8, 16> {
    const values = [...encodeWord(this.Hi), ...encodeWord(this.Lo)];
    return GoArray.literal<uint8, 16>(
      16,
      0,
      [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15],
      values,
    );
  }
}

export type Hasher$Storage = HashState;

export class Hasher {
  declare private readonly $goType: void;

  private constructor(private readonly $storage: Hasher$Storage) {}

  public static $make(
    h1: number,
    h2: number,
    h3: number,
    h4: number,
    length: number,
    seed: number,
  ): Hasher {
    return new Hasher({ h1, h2, h3, h4, length, seed });
  }

  public static $storageOf(source: Hasher): Hasher$Storage {
    return source.$storage;
  }

  public static $fromStorage(source: Hasher$Storage): Hasher {
    return new Hasher(source);
  }

  public static $zero(): Hasher {
    return new Hasher(makeState(0));
  }

  public static $copy(source: Hasher): Hasher {
    const state = source.$storage;
    return Hasher.$make(
      state.h1,
      state.h2,
      state.h3,
      state.h4,
      state.length,
      state.seed,
    );
  }

  public static $assign(target: Hasher, source: Hasher): void {
    const targetState = target.$storage;
    const sourceState = source.$storage;
    targetState.h1 = sourceState.h1;
    targetState.h2 = sourceState.h2;
    targetState.h3 = sourceState.h3;
    targetState.h4 = sourceState.h4;
    targetState.length = sourceState.length;
    targetState.seed = sourceState.seed;
  }

  public static Reset(
    hasher: Hasher | undefined,
  ): void {
    resetState(GoPointer.direct(hasher).$storage);
  }

  public static Sum128(
    hasher: Hasher | undefined,
  ): Uint128 {
    return digest(GoPointer.direct(hasher).$storage);
  }

  public static Sum64(
    hasher: Hasher | undefined,
  ): uint64 {
    return Hasher.Sum128(hasher).Lo;
  }

  public static Write(
    hasher: Hasher | undefined,
    value: RuntimeSlice<uint8>,
  ): [int64, GoError | undefined] {
    const state = GoPointer.direct(hasher).$storage;
    for (let index = 0; index < value.length; index++) {
      mixByte(state, value.get(index));
    }
    return [value.length, undefined];
  }

  public static WriteString(
    hasher: Hasher | undefined,
    value: gostring,
  ): [int64, GoError | undefined] {
    const state = GoPointer.direct(hasher).$storage;
    const written = mixString(state, value);
    return [written, undefined];
  }
}

export function $initialize(): void {}

export function Hash128(value: RuntimeSlice<uint8>): Uint128 {
  return hashSlice(value, 0);
}

export function HashString128(value: gostring): Uint128 {
  return hashString(value, 0);
}

export function Hash128Seed(
  value: RuntimeSlice<uint8>,
  seed: uint64,
): Uint128 {
  return hashSlice(value, seed);
}

export function Hash(value: RuntimeSlice<uint8>): uint64 {
  return Hash128(value).Lo;
}

export function HashSeed(
  value: RuntimeSlice<uint8>,
  seed: uint64,
): uint64 {
  return Hash128Seed(value, seed).Lo;
}

export function New(): Hasher | undefined {
  return Hasher.$zero();
}
