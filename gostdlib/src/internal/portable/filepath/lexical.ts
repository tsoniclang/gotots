import { RuntimeSlice } from "@gotots/runtime/slice.js";
import { GoString } from "@gotots/runtime/string-value.js";
import type { bool, gostring } from "@gotots/gostdlib/internal/scalars.js";

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
  const text = path.text();
  for (let index = text.length - 1; index >= 0 && text[index] !== "/"; index -= 1) {
    if (text[index] === ".") {
      return path.slice(index);
    }
  }
  return GoString.empty;
}

export function FromSlash(path: gostring): gostring {
  return path;
}

export function IsAbs(path: gostring): bool {
  return path.text().startsWith("/");
}

export function Join(elements: RuntimeSlice<gostring>): gostring {
  const values = sliceValues(elements);
  const firstNonEmpty = values.findIndex((value) => value.text().length > 0);
  return firstNonEmpty < 0 ? GoString.empty : joinSlashPath(values.slice(firstNonEmpty));
}

export function joinValues(elements: readonly gostring[]): gostring {
  const firstNonEmpty = elements.findIndex((value) => value.text().length > 0);
  return firstNonEmpty < 0 ? GoString.empty : joinSlashPath(elements.slice(firstNonEmpty));
}
