import type { gostring } from "@gotots/gostdlib/internal/scalars.js";
import { GoString } from "@gotots/runtime/string-value.js";

export function cleanSlashPath(path: gostring): gostring {
  const text = path.text();
  if (text.length === 0) {
    return GoString.fromText(".");
  }

  const rooted = text.startsWith("/");
  const output: string[] = [];
  for (const element of text.split("/")) {
    if (element.length === 0 || element === ".") {
      continue;
    }
    if (element === "..") {
      if (output.length > 0 && output[output.length - 1] !== "..") {
        output.pop();
      } else if (!rooted) {
        output.push(element);
      }
      continue;
    }
    output.push(element);
  }

  const result = rooted
    ? output.length === 0 ? "/" : `/${output.join("/")}`
    : output.length === 0 ? "." : output.join("/");
  return result === text ? path : GoString.fromText(result);
}

export function joinSlashPath(elements: readonly gostring[]): gostring {
  if (elements.every((element) => element.text().length === 0)) {
    return GoString.empty;
  }
  let combined = "";
  for (const element of elements) {
    if (combined.length > 0) {
      combined += "/";
    }
    combined += element.text();
  }
  return cleanSlashPath(GoString.fromText(combined));
}

export function slashDir(path: gostring): gostring {
  const slash = path.text().lastIndexOf("/");
  return cleanSlashPath(slash < 0 ? GoString.fromText(".") : path.slice(0, slash + 1));
}
