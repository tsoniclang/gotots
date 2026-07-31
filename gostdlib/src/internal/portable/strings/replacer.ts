import { GoPanic } from "@gotots/runtime/panic.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { gostring } from "@gotots/runtime/scalars.js";

import { runeBoundaries } from "../utf8/codec.js";

type Replacement = readonly [gostring, gostring];

let createReplacer: (replacements: readonly Replacement[]) => Replacer;

export class Replacer {
  readonly #replacements: readonly Replacement[];

  private constructor(replacements: readonly Replacement[]) {
    this.#replacements = replacements;
  }

  static {
    createReplacer = (replacements: readonly Replacement[]): Replacer =>
      new Replacer(replacements);
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

function replacePairs(text: gostring, replacements: readonly Replacement[]): gostring {
  const boundaries = new Set(runeBoundaries(text));
  let result = "";
  let index = 0;
  let emptyMatched = false;
  while (index <= text.length) {
    let selected: Replacement | undefined;
    for (const replacement of replacements) {
      const [oldText] = replacement;
      if (
        (oldText.length === 0 && boundaries.has(index) && !emptyMatched) ||
        (oldText.length > 0 && text.startsWith(oldText, index))
      ) {
        selected = replacement;
        break;
      }
    }
    if (selected !== undefined) {
      const [oldText, newText] = selected;
      result += newText;
      if (oldText.length > 0) {
        index += oldText.length;
        emptyMatched = false;
      } else {
        emptyMatched = true;
      }
      continue;
    }
    if (index === text.length) {
      break;
    }
    result += text[index];
    index += 1;
    emptyMatched = false;
  }
  return result;
}
