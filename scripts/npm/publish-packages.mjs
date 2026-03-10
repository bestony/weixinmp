import path from "node:path";
import {
  ensureExists,
  loadManifest,
  npmCommand,
  parseArgs,
  repoRoot,
  runCommand,
} from "./lib.mjs";

const args = parseArgs(process.argv);
const manifestFile = path.resolve(
  args.manifest ?? process.env.MANIFEST_PATH ?? path.join(repoRoot, "dist", "npm", "manifest.json"),
);
const { manifest, directory } = loadManifest(manifestFile);
const packages = [...manifest.platforms, manifest.main];
const distTag = args.tag ?? process.env.NPM_DIST_TAG ?? inferDistTag(manifest.version);

for (const pkg of packages) {
  if (!pkg.tarball) {
    throw new Error(`package ${pkg.name} has not been packed yet`);
  }

  const tarballPath = path.resolve(directory, pkg.tarball);
  ensureExists(tarballPath, `tarball for ${pkg.name}`);

  const npmArgs = ["publish", tarballPath];
  if (pkg.name.startsWith("@")) {
    npmArgs.push("--access", "public");
  }
  if (distTag) {
    npmArgs.push("--tag", distTag);
  }
  if (args.dryRun) {
    npmArgs.push("--dry-run");
  }

  console.log(`Publishing ${pkg.name} from ${tarballPath}`);
  runCommand(npmCommand(), npmArgs);
}

function inferDistTag(version) {
  const prerelease = version.split("-", 2)[1];
  if (!prerelease) {
    return null;
  }

  return prerelease.split(".", 1)[0] || "next";
}
