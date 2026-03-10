const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const test = require("node:test");

const { resolveBinary } = require("../weixinmp/lib/resolve-binary");
const { getSupportedTarget } = require("../weixinmp/lib/targets");

function makeTempInstall(t, target, withPackage = true) {
  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "weixinmp-npm-"));
  t.after(() => {
    fs.rmSync(tempDir, { recursive: true, force: true });
  });

  const mainDir = path.join(tempDir, "node_modules", "weixinmp", "lib");
  fs.mkdirSync(mainDir, { recursive: true });

  let binaryPath = null;
  if (withPackage) {
    const packageDir = path.join(tempDir, "node_modules", ...target.packageName.split("/"));
    fs.mkdirSync(packageDir, { recursive: true });
    fs.writeFileSync(
      path.join(packageDir, "package.json"),
      JSON.stringify({ name: target.packageName, version: "1.2.3" }, null, 2),
    );

    binaryPath = path.join(packageDir, target.binaryRelativePath);
    fs.mkdirSync(path.dirname(binaryPath), { recursive: true });
    fs.writeFileSync(binaryPath, "binary");
  }

  return {
    binaryPath,
    mainDir,
  };
}

test("resolveBinary returns the current target binary path when installed", (t) => {
  const target = getSupportedTarget("linux", "x64");
  assert.ok(target);

  const install = makeTempInstall(t, target);
  const resolved = resolveBinary(install.mainDir, "linux", "x64");

  assert.equal(resolved.target.packageName, target.packageName);
  assert.equal(fs.realpathSync(resolved.binaryPath), fs.realpathSync(install.binaryPath));
});

test("resolveBinary fails for unsupported targets", () => {
  assert.throws(
    () => resolveBinary(__dirname, "linux", "ppc64"),
    /Unsupported platform linux\/ppc64/,
  );
});

test("resolveBinary fails when the platform package is missing", (t) => {
  const target = getSupportedTarget("darwin", "arm64");
  assert.ok(target);

  const install = makeTempInstall(t, target, false);
  assert.throws(
    () => resolveBinary(install.mainDir, "darwin", "arm64"),
    /Missing optional dependency "@bestony\/weixinmp-darwin-arm64"/,
  );
});
