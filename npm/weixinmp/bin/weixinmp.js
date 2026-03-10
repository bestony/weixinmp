#!/usr/bin/env node

const { spawnSync } = require("node:child_process");
const { resolveBinary } = require("../lib/resolve-binary");

try {
  const { binaryPath } = resolveBinary(__dirname);
  const result = spawnSync(binaryPath, process.argv.slice(2), {
    stdio: "inherit",
    windowsHide: false,
  });

  if (result.error) {
    throw result.error;
  }

  if (result.signal) {
    console.error(`weixinmp terminated with signal ${result.signal}`);
    process.exit(1);
  }

  process.exit(result.status ?? 1);
} catch (error) {
  console.error(error instanceof Error ? error.message : String(error));
  process.exit(1);
}
