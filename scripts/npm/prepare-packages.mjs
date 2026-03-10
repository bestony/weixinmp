import { writeFileSync } from "node:fs";
import path from "node:path";
import {
  defaultBinaryName,
  findBuiltBinary,
  manifestPath,
  normalizeVersion,
  packDirectory,
  parseArgs,
  relativeFrom,
  repoRoot,
  resetDirectory,
  resolveReleaseTag,
  selectTargets,
  stageMainPackage,
  stagePlatformPackage,
} from "./lib.mjs";

const args = parseArgs(process.argv);
const versionInput =
  args.version ?? process.env.PACKAGE_VERSION ?? process.env.VERSION ?? process.env.GITHUB_REF_NAME;
const version = normalizeVersion(versionInput);
const releaseTag = resolveReleaseTag(
  versionInput,
  args.tag ?? process.env.RELEASE_TAG ?? process.env.TAG,
);
const distDir = path.resolve(args.distDir ?? process.env.DIST_DIR ?? path.join(repoRoot, "dist"));
const outputDir = path.resolve(
  args.outputDir ?? process.env.OUTPUT_DIR ?? path.join(distDir, "npm"),
);
const binaryName = args.binaryName ?? process.env.BINARY_NAME ?? defaultBinaryName;
const selectedTargets = selectTargets(args.targets ?? process.env.NPM_TARGETS ?? "all");
const tarballDir = path.join(outputDir, "tarballs");

resetDirectory(outputDir);

const optionalDependencies = Object.fromEntries(
  selectedTargets.map((target) => [target.packageName, version]),
);

const mainPackageDir = stageMainPackage({
  outputDir,
  version,
  binaryName,
  optionalDependencies,
});

const platforms = selectedTargets.map((target) => {
  const binarySourcePath = findBuiltBinary({
    distDir,
    binaryName,
    releaseTag,
    target,
  });
  const packageDir = stagePlatformPackage({
    outputDir,
    version,
    binaryName,
    target,
    binarySourcePath,
  });

  return {
    id: target.id,
    name: target.packageName,
    platform: target.platform,
    arch: target.arch,
    packageDir: relativeFrom(outputDir, packageDir),
    binarySourcePath: relativeFrom(outputDir, binarySourcePath),
    tarball: null,
  };
});

const main = {
  name: "weixinmp",
  packageDir: relativeFrom(outputDir, mainPackageDir),
  tarball: null,
};

if (args.pack) {
  for (const platformTarget of platforms) {
    const packageDir = path.resolve(outputDir, platformTarget.packageDir);
    const tarballPath = packDirectory(packageDir, tarballDir);
    platformTarget.tarball = relativeFrom(outputDir, tarballPath);
  }

  const mainTarballPath = packDirectory(mainPackageDir, tarballDir);
  main.tarball = relativeFrom(outputDir, mainTarballPath);
}

const manifest = {
  version,
  releaseTag,
  binaryName,
  generatedAt: new Date().toISOString(),
  main,
  platforms,
  publishOrder: [...platforms.map((item) => item.name), main.name],
};

writeFileSync(manifestPath(outputDir), `${JSON.stringify(manifest, null, 2)}\n`);

console.log(
  [
    `Prepared npm packages for ${selectedTargets.length} target(s).`,
    `version=${version}`,
    `releaseTag=${releaseTag}`,
    `outputDir=${outputDir}`,
    args.pack ? `tarballs=${tarballDir}` : "tarballs=not-packed",
  ].join("\n"),
);
