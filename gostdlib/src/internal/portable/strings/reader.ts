import { GoPanic } from "@gotots/runtime/panic.js";
import { GoMapHash } from "@gotots/runtime/map.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { GoError } from "@gotots/runtime/interface-value.js";
import type { gostring, int, uint8 } from "@gotots/gostdlib/internal/scalars.js";
import { integerFromHost } from "../../host-integer.js";

import { state as ioState } from "../../../io.js";

let createReader: (source: gostring) => Reader;
let assignReaderRepresentation: (target: Reader, source: Reader) => void;
let copyReaderRepresentation: (source: Reader) => Reader;
let equalReaderRepresentation: (left: Reader, right: Reader) => boolean;
let hashReaderRepresentation: (source: Reader) => number;

export class Reader {
  #source: gostring;
  #offset = 0;
  #previousRune = -1;

  private constructor(source: gostring) {
    this.#source = source;
  }

  static {
    createReader = (source: gostring): Reader => new Reader(source);
    assignReaderRepresentation = (target: Reader, source: Reader): void => {
      const text = source.#source;
      const offset = source.#offset;
      const previousRune = source.#previousRune;
      target.#source = text;
      target.#offset = offset;
      target.#previousRune = previousRune;
    };
    copyReaderRepresentation = (source: Reader): Reader => {
      const target = new Reader(source.#source);
      assignReaderRepresentation(target, source);
      return target;
    };
    equalReaderRepresentation = (left: Reader, right: Reader): boolean => (
      left.#source === right.#source &&
      left.#offset === right.#offset &&
      left.#previousRune === right.#previousRune
    );
    hashReaderRepresentation = (source: Reader): number => {
      let hash = GoMapHash.string(source.#source);
      hash = GoMapHash.mix(hash, GoMapHash.number(source.#offset));
      return GoMapHash.mix(hash, GoMapHash.number(source.#previousRune));
    };
  }

  static Read(
    receiver: Reader | undefined,
    target: RuntimeSlice<uint8>,
  ): [int, GoError | undefined] {
    if (receiver === undefined) {
      GoPanic.raiseRuntime("nil *strings.Reader");
    }
    receiver.#previousRune = -1;
    if (receiver.#offset >= receiver.#source.length) {
      return [0n, ioState.EOF];
    }
    const count = Math.min(target.length, receiver.#source.length - receiver.#offset);
    for (let index = 0; index < count; index += 1) {
      target.set(index, receiver.#source.charCodeAt(receiver.#offset + index));
    }
    receiver.#offset += count;
    return [integerFromHost(count), undefined];
  }
}

export function NewReader(text: gostring): Reader {
  return createReader(text);
}

export function readerRepresentationAssign(
  target: Reader,
  source: Reader,
): void {
  assignReaderRepresentation(target, source);
}

export function readerRepresentationCopy(source: Reader): Reader {
  return copyReaderRepresentation(source);
}

export function readerRepresentationEqual(
  left: Reader,
  right: Reader,
): boolean {
  return equalReaderRepresentation(left, right);
}

export function readerRepresentationHash(source: Reader): number {
  return hashReaderRepresentation(source);
}
