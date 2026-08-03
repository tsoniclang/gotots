import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { bool, gostring } from "@gotots/runtime/scalars.js";

import {
  cleanSlashPath,
  joinSlashPath,
  slashDir,
} from "../path/clean.js";
import { sliceValues } from "../../runtime/slice.js";

export function Clean(path: gostring): gostring {
  return cleanSlashPath(path);
}

export function Dir(path: gostring): gostring {
  return slashDir(path);
}

export function Ext(path: gostring): gostring {
  for (let index = path.length - 1; index >= 0 && path[index] !== "/"; index -= 1) {
    if (path[index] === ".") {
      return path.slice(index);
    }
  }
  return "";
}

export function FromSlash(path: gostring): gostring {
  return path;
}

export function IsAbs(path: gostring): bool {
  return path.startsWith("/");
}

export function Join(elements: RuntimeSlice<gostring>): gostring {
  const values = sliceValues(elements);
  const firstNonEmpty = values.findIndex((value) => value.length > 0);
  return firstNonEmpty < 0 ? "" : joinSlashPath(values.slice(firstNonEmpty));
}

export function joinValues(elements: readonly gostring[]): gostring {
  const firstNonEmpty = elements.findIndex((value) => value.length > 0);
  return firstNonEmpty < 0 ? "" : joinSlashPath(elements.slice(firstNonEmpty));
}
