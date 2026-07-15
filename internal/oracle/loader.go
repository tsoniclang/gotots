package oracle

// LoaderSource is the documented host-execution normalization: generated
// product output uses explicit .js specifiers, and this Node module
// resolver maps them onto the .ts sources for execution. It is part of
// the closed input contract (the module-resolver identity) and never
// mutates generated output.
const LoaderSource = `import { pathToFileURL } from "node:url";
import { access } from "node:fs/promises";

export async function resolve(specifier, context, nextResolve) {
  if (specifier.startsWith(".") && specifier.endsWith(".js")) {
    const parent = context.parentURL ? new URL(context.parentURL) : pathToFileURL(process.cwd() + "/");
    const candidate = new URL(specifier.slice(0, -3) + ".ts", parent);
    try {
      await access(candidate);
      return nextResolve(candidate.href, context);
    } catch {}
  }
  return nextResolve(specifier, context);
}
`
