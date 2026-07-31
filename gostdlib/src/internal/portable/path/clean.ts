import type { gostring } from "@gotots/runtime/scalars.js";

export function cleanSlashPath(path: gostring): gostring {
  if (path.length === 0) {
    return ".";
  }

  const rooted = path.startsWith("/");
  const output: gostring[] = [];
  for (const element of path.split("/")) {
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

  if (rooted) {
    return output.length === 0 ? "/" : `/${output.join("/")}`;
  }
  return output.length === 0 ? "." : output.join("/");
}

export function joinSlashPath(elements: readonly gostring[]): gostring {
  if (elements.every((element) => element.length === 0)) {
    return "";
  }
  let combined = "";
  for (const element of elements) {
    if (combined.length > 0) {
      combined += "/";
    }
    combined += element;
  }
  return cleanSlashPath(combined);
}

export function slashDir(path: gostring): gostring {
  const slash = path.lastIndexOf("/");
  return cleanSlashPath(slash < 0 ? "." : path.slice(0, slash + 1));
}
