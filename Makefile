SHELL := /bin/bash

GO ?= go
NODE ?= node
PYTHON ?= python3

BINARY_NAME ?= weixinmp
PKG ?= .
TEST_PKGS ?= ./...
COVERAGE_FILE ?= coverage.out
COVERMODE ?= atomic
DIST_DIR ?= dist

VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
TAG ?= $(VERSION)
NPM_PACKAGE_VERSION ?= $(if $(filter dev,$(VERSION)),0.0.0-dev,$(VERSION))
NPM_RELEASE_TAG ?= $(TAG)
NPM_OUTPUT_DIR ?= $(DIST_DIR)/npm
NPM_TARGETS ?= all

BUILD_FLAGS ?= -trimpath
GO_LD_FLAGS ?= -X main.version=$(VERSION) -X main.commit=$(COMMIT)

RUN_ARGS ?= --help
TEST_ARGS ?=

GOOS ?= $(shell $(GO) env GOOS)
GOARCH ?= $(shell $(GO) env GOARCH)
EXT = $(if $(filter windows,$(GOOS)),.exe,)
OUT_DIR = $(DIST_DIR)/$(BINARY_NAME)_$(TAG)_$(GOOS)_$(GOARCH)
BIN_PATH = $(OUT_DIR)/$(BINARY_NAME)$(EXT)
ZIP_PATH = $(DIST_DIR)/$(BINARY_NAME)_$(TAG)_$(GOOS)_$(GOARCH).zip

PLATFORMS ?= linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

.DEFAULT_GOAL := help

.PHONY: help build run test test-cli cover cover-check cover-func clean fmt package package-all package-npm pack-npm verify-npm-install

help: ## 显示可用目标
	@awk 'BEGIN {FS = ":.*## "}; /^[a-zA-Z0-9_-]+:.*## / {printf "\033[36m%-14s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## 本地构建二进制到当前目录
	$(GO) build $(BUILD_FLAGS) -ldflags "$(GO_LD_FLAGS)" -o $(BINARY_NAME) $(PKG)

run: ## 运行程序，默认参数为 --help，可用 RUN_ARGS 覆盖
	$(GO) run $(PKG) $(RUN_ARGS)

test: ## 运行全部测试
	$(GO) test $(TEST_PKGS) $(TEST_ARGS)

test-cli: ## 只运行 CLI 相关测试
	$(GO) test $(TEST_PKGS) -run TestCLI $(TEST_ARGS)

cover: ## 生成 coverage.out
	$(GO) test $(TEST_PKGS) -coverprofile=$(COVERAGE_FILE) -covermode=$(COVERMODE) $(TEST_ARGS)

cover-func: cover ## 输出按函数统计的覆盖率
	$(GO) tool cover -func=$(COVERAGE_FILE)

cover-check: cover ## 按 CI 规则校验总覆盖率达到 100%
	@set -euo pipefail; \
	total="$$( $(GO) tool cover -func=$(COVERAGE_FILE) | awk '/^total:/{gsub(/%/,"",$$3); print $$3}' )"; \
	if [[ -z "$$total" ]]; then \
		echo "Failed to parse total coverage from $(COVERAGE_FILE)"; \
		exit 1; \
	fi; \
	echo "Total coverage: $${total}%"; \
	awk -v total="$$total" 'BEGIN { if (total < 100) { printf("Coverage %.1f%% is below 100%%\n", total); exit 1 } }'

fmt: ## 格式化 Go 代码
	$(GO) fmt ./...

clean: ## 清理本地构建和覆盖率产物
	rm -f $(BINARY_NAME) $(COVERAGE_FILE)
	rm -rf $(DIST_DIR)

package: ## 按当前 GOOS/GOARCH 打包 zip，可传 VERSION/TAG/GOOS/GOARCH
	@set -euo pipefail; \
	mkdir -p "$(OUT_DIR)"; \
	CGO_ENABLED=0 GOOS="$(GOOS)" GOARCH="$(GOARCH)" \
	$(GO) build $(BUILD_FLAGS) -ldflags "$(GO_LD_FLAGS)" -o "$(BIN_PATH)" $(PKG); \
	ZIP_PATH="$(ZIP_PATH)" BIN_PATH="$(BIN_PATH)" $(PYTHON) -c 'import os, zipfile; zip_path = os.environ["ZIP_PATH"]; bin_path = os.environ["BIN_PATH"]; zf = zipfile.ZipFile(zip_path, "w", compression=zipfile.ZIP_DEFLATED); zf.write(bin_path, arcname=os.path.basename(bin_path)); zf.close(); print(zip_path)'

package-all: ## 按 release 工作流的默认平台矩阵全部打包
	@set -euo pipefail; \
	for platform in $(PLATFORMS); do \
		goos="$${platform%/*}"; \
		goarch="$${platform#*/}"; \
		echo "==> packaging $$goos/$$goarch"; \
		$(MAKE) package GOOS="$$goos" GOARCH="$$goarch" VERSION="$(VERSION)" TAG="$(TAG)" COMMIT="$(COMMIT)" GO_LD_FLAGS='$(GO_LD_FLAGS)'; \
	done

package-npm: ## 基于 dist 目录生成 npm 主包和平台包目录
	NPM_TARGETS="$(NPM_TARGETS)" $(NODE) ./scripts/npm/prepare-packages.mjs --version "$(NPM_PACKAGE_VERSION)" --tag "$(NPM_RELEASE_TAG)" --dist-dir "$(DIST_DIR)" --output-dir "$(NPM_OUTPUT_DIR)"

pack-npm: ## 基于 dist 目录生成并打包 npm 主包和平台包
	NPM_TARGETS="$(NPM_TARGETS)" $(NODE) ./scripts/npm/prepare-packages.mjs --pack --version "$(NPM_PACKAGE_VERSION)" --tag "$(NPM_RELEASE_TAG)" --dist-dir "$(DIST_DIR)" --output-dir "$(NPM_OUTPUT_DIR)"

verify-npm-install: ## 安装本地生成的 npm 包并校验 CLI 可执行
	$(NODE) ./scripts/npm/verify-install.mjs --manifest "$(NPM_OUTPUT_DIR)/manifest.json"
