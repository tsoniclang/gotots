import { readFile, readdir } from "node:fs/promises";

const sourceRoot = new URL("../src/", import.meta.url);
const forbidden = [
  /\$argument/u,
  /__from_/u,
  /\$cooperative_/u,
  /\$contract/u,
  /\$state/u,
  /\b[A-Za-z_][A-Za-z0-9_]*_[A-Z][A-Za-z0-9_]*\s*\(/u,
];

for (const file of await sourceFiles(sourceRoot)) {
  if (file.pathname.includes("/internal/")) {
    continue;
  }
  const source = await readFile(file, "utf8");
  for (const pattern of forbidden) {
    if (pattern.test(source)) {
      throw new Error(
        `public provider source ${file.pathname} contains ${pattern}`,
      );
    }
  }
}

async function sourceFiles(directory) {
  const result = [];
  for (const entry of await readdir(directory, { withFileTypes: true })) {
    const child = new URL(entry.name + (entry.isDirectory() ? "/" : ""), directory);
    if (entry.isDirectory()) {
      result.push(...await sourceFiles(child));
    } else if (entry.name.endsWith(".ts")) {
      result.push(child);
    }
  }
  return result;
}
