const fs = require("node:fs");
const path = require("node:path");

const targets = JSON.parse(
  fs.readFileSync(path.join(__dirname, "targets.json"), "utf8"),
);

function getSupportedTarget(platform = process.platform, arch = process.arch) {
  return targets.find((item) => item.platform === platform && item.arch === arch) ?? null;
}

module.exports = {
  getSupportedTarget,
  targets,
};
