#!/usr/bin/env bash

set -euo pipefail

TAG="${TAG:-}"
DIST_DIR="${DIST_DIR:-dist}"
OUTPUT_PATH="${OUTPUT_PATH:-Formula/weixinmp.rb}"
REPO_SLUG="${REPO_SLUG:-bestony/weixinmp}"
BINARY_NAME="${BINARY_NAME:-weixinmp}"

if [[ -z "${TAG}" ]]; then
  echo "TAG is required, for example: TAG=v0.0.0" >&2
  exit 1
fi

version="${TAG#v}"

asset_sha() {
  local goos="$1"
  local goarch="$2"
  local asset_name="${BINARY_NAME}_${TAG}_${goos}_${goarch}.zip"
  local asset_path

  asset_path="$(find "${DIST_DIR}" -type f -name "${asset_name}" -print -quit)"
  if [[ -z "${asset_path}" ]]; then
    echo "missing asset: ${asset_name} under ${DIST_DIR}" >&2
    exit 1
  fi

  shasum -a 256 "${asset_path}" | awk '{print $1}'
}

darwin_amd64_sha="$(asset_sha darwin amd64)"
darwin_arm64_sha="$(asset_sha darwin arm64)"
linux_amd64_sha="$(asset_sha linux amd64)"
linux_arm64_sha="$(asset_sha linux arm64)"

mkdir -p "$(dirname "${OUTPUT_PATH}")"

cat >"${OUTPUT_PATH}" <<EOF
class Weixinmp < Formula
  desc "CLI toolbox for WeChat Official Account"
  homepage "https://github.com/${REPO_SLUG}"
  license "GPL-3.0-only"
  version "${version}"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/${REPO_SLUG}/releases/download/v#{version}/weixinmp_v#{version}_darwin_arm64.zip"
      sha256 "${darwin_arm64_sha}"
    else
      url "https://github.com/${REPO_SLUG}/releases/download/v#{version}/weixinmp_v#{version}_darwin_amd64.zip"
      sha256 "${darwin_amd64_sha}"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/${REPO_SLUG}/releases/download/v#{version}/weixinmp_v#{version}_linux_arm64.zip"
      sha256 "${linux_arm64_sha}"
    else
      url "https://github.com/${REPO_SLUG}/releases/download/v#{version}/weixinmp_v#{version}_linux_amd64.zip"
      sha256 "${linux_amd64_sha}"
    end
  end

  def install
    bin.install "${BINARY_NAME}"
  end

  test do
    output = shell_output("#{bin}/${BINARY_NAME} --version")
    assert_match "v#{version}", output
  end
end
EOF
