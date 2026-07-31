import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { gostring } from "@gotots/runtime/scalars.js";

import { joinSlashPath } from "./clean.js";
import { sliceValues } from "../../runtime/slice.js";

export function Join(elements: RuntimeSlice<gostring>): gostring {
  return joinSlashPath(sliceValues(elements));
}
