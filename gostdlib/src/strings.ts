import { GoPanic } from "@gotots/runtime/panic.js";
import type { gostring } from "@gotots/runtime/scalars.js";

import { Seq } from "./iter.js";

export { Builder } from "./internal/portable/strings/builder.js";
export { NewReader, Reader } from "./internal/portable/strings/reader.js";
export { NewReplacer, Replacer } from "./internal/portable/strings/replacer.js";
export {
  Clone,
  Compare,
  Contains,
  ContainsFunc,
  ContainsAny,
  ContainsRune,
  Count,
  Cut,
  CutPrefix,
  CutSuffix,
  EqualFold,
  HasPrefix,
  HasSuffix,
  Index,
  IndexAny,
  IndexByte,
  IndexFunc,
  IndexRune,
  LastIndex,
  LastIndexByte,
  LastIndexFunc,
} from "./internal/portable/strings/search.js";
export { Join, Split } from "./internal/portable/strings/split.js";
export {
  Map,
  Repeat,
  Replace,
  ReplaceAll,
  ToLower,
  ToUpper,
  ToValidUTF8,
  Trim,
  TrimFunc,
  TrimLeft,
  TrimLeftFunc,
  TrimPrefix,
  TrimRight,
  TrimRightFunc,
  TrimSpace,
  TrimSuffix,
} from "./internal/portable/strings/transform.js";

type AsyncYield = ((value: gostring) => Promise<boolean>) | undefined;
type AsyncSequence = (yieldValue: AsyncYield) => Promise<void>;

export function Lines(text: gostring): Seq<gostring, AsyncSequence | undefined> {
  return new Seq<gostring, AsyncSequence | undefined>(
    async (yieldValue: AsyncYield): Promise<void> => {
      if (yieldValue === undefined) {
        GoPanic.raiseRuntime("call of nil yield function");
      }
      let remaining = text;
      while (remaining.length > 0) {
        const newline = remaining.indexOf("\n");
        const line = newline < 0 ? remaining : remaining.slice(0, newline + 1);
        remaining = newline < 0 ? "" : remaining.slice(newline + 1);
        if (!(await yieldValue(line))) {
          return;
        }
      }
    },
  );
}
