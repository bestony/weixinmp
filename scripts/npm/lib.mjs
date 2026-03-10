import { spawnSync } from "node:child_process";
import {
  chmodSync,
  cpSync,
  existsSync,
  mkdirSync,
  readFileSync,
  readdirSync,
  rmSync,
  statSync,
  writeFileSync,
} from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

export const repoRoot = path.resolve(__dirname, "../..");
export const defaultBinaryName = "weixinmp";
export const mainTemplateDir = path.join(repoRoot, "npm", "weixinmp");
export const targetConfigPath = path.join(mainTemplateDir, "lib", "targets.json");
export const targets = JSON.parse(readFileSync(targetConfigPath, "utf8"));

const SEMVER_RE = /^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$/;

export function parseArgs(argv) {
  const args = {};

  for (let index = 2; index < argv.length; index += 1) {
    const token = argv[index];
    if (!token.startsWith("--")) {
      throw new Error(`unexpected argument: ${token}`);
    }

    const key = token.slice(2);
    if (key === "pack" || key === "dry-run") {
      args[toCamelCase(key)] = true;
      continue;
    }

    const value = argv[index + 1];
    if (value === undefined || value.startsWith("--")) {
      throw new Error(`missing value for --${key}`);
    }
    args[toCamelCase(key)] = value;
    index += 1;
  }

  return args;
}

export function normalizeVersion(input) {
  if (!input) {
    throw new Error("version is required");
  }

  const version = input.startsWith("v") ? input.slice(1) : input;
  if (!SEMVER_RE.test(version)) {
    throw new Error(`invalid npm package version: ${input}`);
  }

  return version;
}

export function resolveReleaseTag(versionInput, explicitTag) {
  if (explicitTag) {
    return explicitTag;
  }

  if (!versionInput) {
    throw new Error("version is required to infer a release tag");
  }

  return versionInput.startsWith("v") ? versionInput : `v${normalizeVersion(versionInput)}`;
}

export function selectTargets(spec) {
  if (!spec || spec === "all") {
    return targets;
  }

  if (spec === "current") {
    const target = getCurrentTarget();
    if (!target) {
      throw new Error(`current platform ${process.platform}/${process.arch} is not supported`);
    }
    return [target];
  }

  const requested = spec
    .split(",")
    .map((value) => value.trim())
    .filter(Boolean);

  const selected = requested.map((value) => {
    const match = targets.find(
      (target) => target.id === value || target.packageName === value,
    );
    if (!match) {
      throw new Error(`unknown target: ${value}`);
    }
    return match;
  });

  return selected;
}

export function getCurrentTarget() {
  return (
    targets.find(
      (target) => target.platform === process.platform && target.arch === process.arch,
    ) ?? null
  );
}

export function resetDirectory(directoryPath) {
  rmSync(directoryPath, { recursive: true, force: true });
  mkdirSync(directoryPath, { recursive: true });
}

export function stageMainPackage({ outputDir, version, binaryName, optionalDependencies }) {
  const packageDir = path.join(outputDir, "weixinmp");
  cpSync(mainTemplateDir, packageDir, { recursive: true });
  copyRootMetadata(packageDir, { readme: true });

  writeJSON(path.join(packageDir, "package.json"), {
    name: "weixinmp",
    version,
    description: "CLI toolbox for WeChat Official Account",
    license: "GPL-3.0-only",
    homepage: "https://github.com/bestony/weixinmp",
    repository: {
      type: "git",
      url: "git+https://github.com/bestony/weixinmp.git",
    },
    bugs: {
      url: "https://github.com/bestony/weixinmp/issues",
    },
    keywords: [
      "cli",
      "wechat",
      "weixin",
      "weixinmp",
      "official-account",
    ],
    engines: {
      node: ">=18",
    },
    publishConfig: {
      access: "public",
    },
    bin: {
      [binaryName]: "bin/weixinmp.js",
    },
    files: ["bin", "lib", "README.md", "LICENSE"],
    optionalDependencies,
  });

  return packageDir;
}

export function stagePlatformPackage({
  outputDir,
  version,
  binaryName,
  target,
  binarySourcePath,
}) {
  const packageDir = path.join(outputDir, "platform", target.id);
  const binaryNameInPackage = path.basename(target.binaryRelativePath);
  mkdirSync(path.join(packageDir, "bin"), { recursive: true });

  cpSync(binarySourcePath, path.join(packageDir, "bin", binaryNameInPackage));
  if (target.platform !== "win32") {
    chmodSyncSafe(path.join(packageDir, "bin", binaryNameInPackage), 0o755);
  }

  copyRootMetadata(packageDir);
  writeFileSync(
    path.join(packageDir, "README.md"),
    [
      `# ${target.packageName}`,
      "",
      `Prebuilt ${target.platform}/${target.arch} binary for \`${binaryName}\`.`,
      "",
      `This package is published as an internal distribution target for the main \`weixinmp\` npm package.`,
      "",
    ].join("\n"),
  );

  writeJSON(path.join(packageDir, "package.json"), {
    name: target.packageName,
    version,
    description: `${target.platform}/${target.arch} binary for weixinmp`,
    license: "GPL-3.0-only",
    homepage: "https://github.com/bestony/weixinmp",
    repository: {
      type: "git",
      url: "git+https://github.com/bestony/weixinmp.git",
    },
    bugs: {
      url: "https://github.com/bestony/weixinmp/issues",
    },
    os: [target.platform],
    cpu: [target.arch],
    publishConfig: {
      access: "public",
    },
    files: ["bin", "README.md", "LICENSE"],
  });

  return packageDir;
}

export function findBuiltBinary({ distDir, binaryName, releaseTag, target }) {
  const expectedDir = `${binaryName}_${releaseTag}_${target.goos}_${target.goarch}`;
  const expectedFileName = path.basename(target.binaryRelativePath);
  const matches = [];

  walkFiles(distDir, (filePath) => {
    if (
      path.basename(filePath) === expectedFileName &&
      path.basename(path.dirname(filePath)) === expectedDir
    ) {
      matches.push(filePath);
    }
  });

  if (matches.length === 0) {
    throw new Error(
      `missing built binary for ${target.id}: expected ${expectedDir}/${expectedFileName} under ${distDir}`,
    );
  }

  matches.sort((left, right) => left.length - right.length || left.localeCompare(right));
  return matches[0];
}

export function packDirectory(packageDir, tarballDir) {
  mkdirSync(tarballDir, { recursive: true });

  const result = runCommand(npmCommand(), ["pack", "--json", "--pack-destination", tarballDir], {
    cwd: packageDir,
  });
  const parsed = JSON.parse(result.stdout.trim());
  if (!Array.isArray(parsed) || parsed.length !== 1 || !parsed[0].filename) {
    throw new Error(`unexpected npm pack output for ${packageDir}: ${result.stdout}`);
  }

  return path.join(tarballDir, parsed[0].filename);
}

export function runCommand(command, args, options = {}) {
  const result = spawnSync(command, args, {
    cwd: options.cwd,
    env: options.env ?? process.env,
    encoding: "utf8",
  });

  if (result.status !== 0) {
    throw new Error(
      [
        `command failed: ${command} ${args.join(" ")}`,
        result.stdout?.trim(),
        result.stderr?.trim(),
      ]
        .filter(Boolean)
        .join("\n"),
    );
  }

  return result;
}

export function npmCommand() {
  return process.platform === "win32" ? "npm.cmd" : "npm";
}

export function manifestPath(outputDir) {
  return path.join(outputDir, "manifest.json");
}

export function loadManifest(manifestFile) {
  const absolutePath = path.resolve(manifestFile);
  const manifest = JSON.parse(readFileSync(absolutePath, "utf8"));
  return {
    absolutePath,
    directory: path.dirname(absolutePath),
    manifest,
  };
}

export function relativeFrom(baseDir, targetPath) {
  return toPosix(path.relative(baseDir, targetPath));
}

export function ensureExists(filePath, description) {
  if (!existsSync(filePath)) {
    throw new Error(`${description} does not exist: ${filePath}`);
  }
}

function copyRootMetadata(packageDir, options = {}) {
  if (options.readme) {
    cpSync(path.join(repoRoot, "README.md"), path.join(packageDir, "README.md"));
  }
  cpSync(path.join(repoRoot, "LICENSE"), path.join(packageDir, "LICENSE"));
  chmodSyncSafe(path.join(packageDir, "bin", "weixinmp.js"), 0o755);
}

function writeJSON(filePath, data) {
  writeFileSync(filePath, `${JSON.stringify(data, null, 2)}\n`);
}

function walkFiles(directoryPath, onFile) {
  if (!existsSync(directoryPath)) {
    throw new Error(`directory does not exist: ${directoryPath}`);
  }

  for (const entry of readdirSync(directoryPath)) {
    const entryPath = path.join(directoryPath, entry);
    const stats = statSync(entryPath);
    if (stats.isDirectory()) {
      walkFiles(entryPath, onFile);
      continue;
    }
    if (stats.isFile()) {
      onFile(entryPath);
    }
  }
}

function chmodSyncSafe(filePath, mode) {
  if (!existsSync(filePath)) {
    return;
  }

  try {
    chmodSync(filePath, mode);
  } catch (error) {
    // chmod is best-effort for publish tarballs and can be ignored on unsupported filesystems.
  }
}

function toCamelCase(value) {
  return value.replace(/-([a-z])/g, (_, char) => char.toUpperCase());
}

function toPosix(value) {
  return value.split(path.sep).join("/");
}
