import type { int32 } from "@gotots/gostdlib/internal/scalars.js";

export { Abs, EvalSymlinks } from "../internal/node/filepath/filesystem.js";
export {
  Clean,
  Dir,
  Ext,
  FromSlash,
  IsAbs,
  Join,
} from "../internal/portable/filepath/lexical.js";

export const Separator: int32 = 0x2f;
