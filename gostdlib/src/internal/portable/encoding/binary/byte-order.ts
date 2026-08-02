import { RuntimeSlice } from "@gotots/runtime/slice.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import type {
  gostring,
  uint16,
  uint32,
  uint64,
  uint8,
} from "@gotots/runtime/scalars.js";

import { ProviderInterfaceValue } from "../../io/value.js";
import { sliceValues } from "../../../runtime/slice.js";

const bigEndianType = Object.freeze({ comparable: true });
const littleEndianType = Object.freeze({ comparable: true });

export abstract class EndianOrder extends ProviderInterfaceValue {
	protected constructor(typeIdentity: { readonly comparable: boolean }) {
    super(typeIdentity);
  }

  abstract PutUint16(buffer: RuntimeSlice<uint8>, value: uint16): void;
  abstract PutUint32(buffer: RuntimeSlice<uint8>, value: uint32): void;
  abstract PutUint64(buffer: RuntimeSlice<uint8>, value: uint64): void;
  abstract String(): gostring;
  abstract Uint16(buffer: RuntimeSlice<uint8>): uint16;
  abstract Uint32(buffer: RuntimeSlice<uint8>): uint32;
  abstract Uint64(buffer: RuntimeSlice<uint8>): uint64;

  GoString(): gostring {
    return this.String();
  }

  AppendUint16(
    buffer: RuntimeSlice<uint8>,
    value: uint16,
  ): RuntimeSlice<uint8> {
    const encoded = RuntimeSlice.make<uint8>(2, 2, 0);
    this.PutUint16(encoded, value);
    return buffer.append(0, sliceValues(encoded));
  }

  AppendUint32(
    buffer: RuntimeSlice<uint8>,
    value: uint32,
  ): RuntimeSlice<uint8> {
    const encoded = RuntimeSlice.make<uint8>(4, 4, 0);
    this.PutUint32(encoded, value);
    return buffer.append(0, sliceValues(encoded));
  }

  AppendUint64(
    buffer: RuntimeSlice<uint8>,
    value: uint64,
  ): RuntimeSlice<uint8> {
    const encoded = RuntimeSlice.make<uint8>(8, 8, 0);
    this.PutUint64(encoded, value);
    return buffer.append(0, sliceValues(encoded));
  }

  protected require(buffer: RuntimeSlice<uint8>, length: number): void {
    if (buffer.length < length) {
      GoPanic.raiseRuntime("index out of range");
    }
  }
}

export class BigEndianOrder extends EndianOrder {
  constructor() {
    super(bigEndianType);
  }

  PutUint16(buffer: RuntimeSlice<uint8>, value: uint16): void {
    this.require(buffer, 2);
    buffer.set(0, (value >>> 8) & 0xff);
    buffer.set(1, value & 0xff);
  }

  PutUint32(buffer: RuntimeSlice<uint8>, value: uint32): void {
    this.require(buffer, 4);
    buffer.set(0, (value >>> 24) & 0xff);
    buffer.set(1, (value >>> 16) & 0xff);
    buffer.set(2, (value >>> 8) & 0xff);
    buffer.set(3, value & 0xff);
  }

  PutUint64(buffer: RuntimeSlice<uint8>, value: uint64): void {
    this.require(buffer, 8);
    const high = Math.floor(value / 0x1_0000_0000);
    const low = value >>> 0;
    buffer.set(0, (high >>> 24) & 0xff);
    buffer.set(1, (high >>> 16) & 0xff);
    buffer.set(2, (high >>> 8) & 0xff);
    buffer.set(3, high & 0xff);
    buffer.set(4, (low >>> 24) & 0xff);
    buffer.set(5, (low >>> 16) & 0xff);
    buffer.set(6, (low >>> 8) & 0xff);
    buffer.set(7, low & 0xff);
  }

  String(): gostring {
    return "BigEndian";
  }

  Uint16(buffer: RuntimeSlice<uint8>): uint16 {
    this.require(buffer, 2);
    return ((buffer.get(0) << 8) | buffer.get(1)) >>> 0;
  }

  Uint32(buffer: RuntimeSlice<uint8>): uint32 {
    this.require(buffer, 4);
    return (
      buffer.get(0) * 0x1_000_000
      + buffer.get(1) * 0x1_0000
      + buffer.get(2) * 0x100
      + buffer.get(3)
    ) >>> 0;
  }

  Uint64(buffer: RuntimeSlice<uint8>): uint64 {
    this.require(buffer, 8);
    const high = (
      buffer.get(0) * 0x1_000_000
      + buffer.get(1) * 0x1_0000
      + buffer.get(2) * 0x100
      + buffer.get(3)
    ) >>> 0;
    const low = (
      buffer.get(4) * 0x1_000_000
      + buffer.get(5) * 0x1_0000
      + buffer.get(6) * 0x100
      + buffer.get(7)
    ) >>> 0;
    return high * 0x1_0000_0000 + low;
  }
}

export class LittleEndianOrder extends EndianOrder {
  constructor() {
    super(littleEndianType);
  }

  PutUint16(buffer: RuntimeSlice<uint8>, value: uint16): void {
    this.require(buffer, 2);
    buffer.set(0, value & 0xff);
    buffer.set(1, (value >>> 8) & 0xff);
  }

  PutUint32(buffer: RuntimeSlice<uint8>, value: uint32): void {
    this.require(buffer, 4);
    buffer.set(0, value & 0xff);
    buffer.set(1, (value >>> 8) & 0xff);
    buffer.set(2, (value >>> 16) & 0xff);
    buffer.set(3, (value >>> 24) & 0xff);
  }

  PutUint64(buffer: RuntimeSlice<uint8>, value: uint64): void {
    this.require(buffer, 8);
    const low = value >>> 0;
    const high = Math.floor(value / 0x1_0000_0000);
    buffer.set(0, low & 0xff);
    buffer.set(1, (low >>> 8) & 0xff);
    buffer.set(2, (low >>> 16) & 0xff);
    buffer.set(3, (low >>> 24) & 0xff);
    buffer.set(4, high & 0xff);
    buffer.set(5, (high >>> 8) & 0xff);
    buffer.set(6, (high >>> 16) & 0xff);
    buffer.set(7, (high >>> 24) & 0xff);
  }

  String(): gostring {
    return "LittleEndian";
  }

  Uint16(buffer: RuntimeSlice<uint8>): uint16 {
    this.require(buffer, 2);
    return (buffer.get(0) | (buffer.get(1) << 8)) >>> 0;
  }

  Uint32(buffer: RuntimeSlice<uint8>): uint32 {
    this.require(buffer, 4);
    return (
      buffer.get(0)
      + buffer.get(1) * 0x100
      + buffer.get(2) * 0x1_0000
      + buffer.get(3) * 0x1_000_000
    ) >>> 0;
  }

  Uint64(buffer: RuntimeSlice<uint8>): uint64 {
    this.require(buffer, 8);
    const low = (
      buffer.get(0)
      + buffer.get(1) * 0x100
      + buffer.get(2) * 0x1_0000
      + buffer.get(3) * 0x1_000_000
    ) >>> 0;
    const high = (
      buffer.get(4)
      + buffer.get(5) * 0x100
      + buffer.get(6) * 0x1_0000
      + buffer.get(7) * 0x1_000_000
    ) >>> 0;
    return low + high * 0x1_0000_0000;
  }
}
