import { existsSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import os from "node:os";
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
const currentPlatform = manifest.platforms.find(
  (item) => item.platform === process.platform && item.arch === process.arch,
);

if (!currentPlatform) {
  throw new Error(`no packaged target for ${process.platform}/${process.arch} in ${manifestFile}`);
}
if (!manifest.main.tarball || !currentPlatform.tarball) {
  throw new Error("manifest is missing packed tarballs; run with --pack first");
}

const mainTarballPath = path.resolve(directory, manifest.main.tarball);
const platformTarballPath = path.resolve(directory, currentPlatform.tarball);
ensureExists(mainTarballPath, "main package tarball");
ensureExists(platformTarballPath, "platform package tarball");

const expectedVersions = Array.from(
  new Set([manifest.releaseTag, manifest.version, `v${manifest.version}`].filter(Boolean)),
);
const tempDir = mkdtempSync(path.join(os.tmpdir(), "weixinmp-install-"));
const prefixDir = path.join(tempDir, "global-prefix");

try {
  writeFileSync(
    path.join(tempDir, "package.json"),
    `${JSON.stringify({ name: "weixinmp-install-smoke", private: true }, null, 2)}\n`,
  );

  runCommand(
    npmCommand(),
    [
      "install",
      "--no-package-lock",
      "--fund=false",
      "--audit=false",
      platformTarballPath,
      mainTarballPath,
    ],
    { cwd: tempDir },
  );

  const result = runCommand(npmCommand(), ["exec", "--", "weixinmp", "--version"], {
    cwd: tempDir,
  });
  const execOutput = `${result.stdout}${result.stderr}`;
  assertExpectedVersion(execOutput);

  runCommand(
    npmCommand(),
    [
      "install",
      "--global",
      "--prefix",
      prefixDir,
      "--fund=false",
      "--audit=false",
      platformTarballPath,
      mainTarballPath,
    ],
    { cwd: tempDir },
  );

  const globalCommandPath = resolveGlobalCommandPath(prefixDir, manifest.binaryName);
  ensureExists(globalCommandPath, "global npm CLI command");
  const globalResult = runInstalledCommand(globalCommandPath, ["--version"]);
  const globalOutput = `${globalResult.stdout}${globalResult.stderr}`;
  assertExpectedVersion(globalOutput);

  console.log(`npm exec output: ${execOutput.trim()}`);
  console.log(`global install output: ${globalOutput.trim()}`);
} finally {
  rmSync(tempDir, { recursive: true, force: true });
}

function assertExpectedVersion(output) {
  if (!expectedVersions.some((version) => output.includes(version))) {
    throw new Error(
      `unexpected version output, expected one of ${expectedVersions.join(", ")} in:\n${output.trim()}`,
    );
  }
}

function resolveGlobalCommandPath(prefix, binaryName) {
  if (process.platform === "win32") {
    const candidates = [
      path.join(prefix, `${binaryName}.cmd`),
      path.join(prefix, `${binaryName}.ps1`),
      path.join(prefix, binaryName),
    ];
    return firstExistingPath(candidates) ?? candidates[0];
  }

  return path.join(prefix, "bin", binaryName);
}

function firstExistingPath(candidates) {
  return candidates.find((candidate) => existsSync(candidate)) ?? null;
}

function runInstalledCommand(commandPath, args) {
  if (process.platform === "win32") {
    return runCommand("cmd.exe", ["/d", "/s", "/c", commandPath, ...args]);
  }

  return runCommand(commandPath, args);
}
