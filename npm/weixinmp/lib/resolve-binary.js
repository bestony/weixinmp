const fs = require("node:fs");
const path = require("node:path");
const { getSupportedTarget, targets } = require("./targets");

function resolveBinary(fromDir = __dirname, platform = process.platform, arch = process.arch) {
  const target = getSupportedTarget(platform, arch);
  if (!target) {
    const supported = targets.map((item) => `${item.platform}/${item.arch}`).join(", ");
    throw new Error(
      `Unsupported platform ${platform}/${arch}. Supported targets: ${supported}.`,
    );
  }

  let packageJSONPath;
  try {
    packageJSONPath = require.resolve(`${target.packageName}/package.json`, {
      paths: [fromDir],
    });
  } catch (error) {
    throw new Error(
      `Missing optional dependency "${target.packageName}" for ${platform}/${arch}. Reinstall "weixinmp" on the target platform.`,
    );
  }

  const binaryPath = path.join(path.dirname(packageJSONPath), target.binaryRelativePath);
  if (!fs.existsSync(binaryPath)) {
    throw new Error(
      `Resolved "${target.packageName}" but the binary is missing at ${binaryPath}. Reinstall "weixinmp".`,
    );
  }

  return {
    binaryPath,
    target,
  };
}

module.exports = {
  resolveBinary,
};
