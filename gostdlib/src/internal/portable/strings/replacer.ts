import { GoPanic } from "@gotots/runtime/panic.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import { GoString } from "@gotots/runtime/string-value.js";
import type { gostring } from "@gotots/gostdlib/internal/scalars.js";

import { runeBoundaries } from "../utf8/codec.js";

type Replacement = readonly [gostring, gostring];

let createReplacer: (replacements: readonly Replacement[]) => Replacer;
let copyReplacer: (source: Replacer) => Replacer;
let assignReplacer: (target: Replacer, source: Replacer) => void;

export class Replacer {
  #replacements: readonly Replacement[];

  private constructor(replacements: readonly Replacement[]) {
    this.#replacements = replacements;
  }

  static {
    createReplacer = (replacements: readonly Replacement[]): Replacer =>
      new Replacer(replacements);
    copyReplacer = (source: Replacer): Replacer =>
      new Replacer(source.#replacements);
    assignReplacer = (target: Replacer, source: Replacer): void => {
      target.#replacements = source.#replacements;
    };
  }

  static Replace(receiver: Replacer | undefined, text: gostring): gostring {
    if (receiver === undefined) {
      GoPanic.raiseRuntime("nil *strings.Replacer");
    }
    return replacePairs(text, receiver.#replacements);
  }
}

export function NewReplacer(values: RuntimeSlice<gostring>): Replacer {
  if (values.length % 2 !== 0) {
    GoPanic.raiseRuntime("strings.NewReplacer: odd argument count");
  }
  const replacements: Replacement[] = [];
  for (let index = 0; index < values.length; index += 2) {
    replacements.push([values.get(index), values.get(index + 1)]);
  }
  return createReplacer(replacements);
}

export function replacerRepresentationCopy(source: Replacer): Replacer {
  return copyReplacer(source);
}

export function replacerRepresentationAssign(
  target: Replacer,
  source: Replacer,
): void {
  assignReplacer(target, source);
}

function replacePairs(text: gostring, replacements: readonly Replacement[]): gostring {
  const boundaries = new Set(runeBoundaries(text));
  const bytes = text.text();
  let result = "";
  let index = 0;
  let emptyMatched = false;
  let replaced = false;
  while (index <= bytes.length) {
    let selected: Replacement | undefined;
    for (const replacement of replacements) {
      const [oldText] = replacement;
      const oldBytes = oldText.text();
      if (
        (oldBytes.length === 0 && boundaries.has(index) && !emptyMatched) ||
        (oldBytes.length > 0 && bytes.startsWith(oldBytes, index))
      ) {
        selected = replacement;
        break;
      }
    }
    if (selected !== undefined) {
      const [oldText, newText] = selected;
      replaced = true;
      result += newText.text();
      const length = Number(oldText.sourceLength());
      if (length > 0) {
        index += length;
        emptyMatched = false;
      } else {
        emptyMatched = true;
      }
      continue;
    }
    if (index === bytes.length) {
      break;
    }
    result += bytes[index];
    index += 1;
    emptyMatched = false;
  }
  return replaced ? GoString.fromText(result) : text;
}
