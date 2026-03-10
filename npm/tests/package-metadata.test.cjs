const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const test = require("node:test");
const { pathToFileURL } = require("node:url");

const libModuleURL = pathToFileURL(
  path.resolve(__dirname, "../../scripts/npm/lib.mjs"),
).href;

test("stageMainPackage emits a publishable CLI package with the weixinmp bin", async (t) => {
  const {
    stageMainPackage,
    targets,
  } = await import(libModuleURL);

  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "weixinmp-main-pkg-"));
  t.after(() => {
    fs.rmSync(tempDir, { recursive: true, force: true });
  });

  const packageDir = stageMainPackage({
    outputDir: tempDir,
    version: "1.2.3",
    binaryName: "weixinmp",
    optionalDependencies: {
      [targets[0].packageName]: "1.2.3",
    },
  });

  const packageJSON = JSON.parse(
    fs.readFileSync(path.join(packageDir, "package.json"), "utf8"),
  );

  assert.equal(packageJSON.name, "weixinmp");
  assert.equal(packageJSON.bin.weixinmp, "bin/weixinmp.js");
  assert.equal(packageJSON.publishConfig.access, "public");
  assert.ok(packageJSON.optionalDependencies[targets[0].packageName]);
  assert.ok(fs.existsSync(path.join(packageDir, "bin", "weixinmp.js")));
});

test("stagePlatformPackage keeps platform binary packages non-CLI", async (t) => {
  const {
    stagePlatformPackage,
    targets,
  } = await import(libModuleURL);

  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "weixinmp-platform-pkg-"));
  t.after(() => {
    fs.rmSync(tempDir, { recursive: true, force: true });
  });

  const target = targets.find((item) => item.id === "linux-x64");
  assert.ok(target);

  const fakeBinaryPath = path.join(tempDir, "weixinmp");
  fs.writeFileSync(fakeBinaryPath, "binary");

  const packageDir = stagePlatformPackage({
    outputDir: tempDir,
    version: "1.2.3",
    binaryName: "weixinmp",
    target,
    binarySourcePath: fakeBinaryPath,
  });

  const packageJSON = JSON.parse(
    fs.readFileSync(path.join(packageDir, "package.json"), "utf8"),
  );

  assert.equal(packageJSON.name, target.packageName);
  assert.equal(packageJSON.os[0], target.platform);
  assert.equal(packageJSON.cpu[0], target.arch);
  assert.equal(packageJSON.publishConfig.access, "public");
  assert.equal(packageJSON.bin, undefined);
  assert.ok(fs.existsSync(path.join(packageDir, target.binaryRelativePath)));
});
