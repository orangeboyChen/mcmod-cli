<!--
File: README.zh-CN.md
Created: 2026-06-20
Description: mcmod CLI 工具中文文档。
-->

# mcmod

> [English](./README.md)

一个 Go CLI，用于管理 Minecraft 整合包规范（`packspec.json`）、依赖锁、
Jar 解析与下载、构建产物（client / server zip）和发布索引。
项目根目录的 `packspec.json` 是唯一人工维护的源配置；CLI 负责把它转成
按 loader 拆分的 lock 文件、可发布的 zip 以及发布索引。

## 功能

- `packspec.json` 作为唯一人工编辑的输入
- 加载器：**NeoForge** 和 **Fabric**（支持 mod 级 loader 限定）
- 来源类型：**CurseForge**（query）、**GitHub Release**（tag + assetPattern）、
  **Git packspec 包**（递归读取 `packspec.json`）、**Local**（本地 jar 模板）和
  **URL**（用户指定下载地址）
- `mcmod lock` 解析 source、对照已有 lock 跑增量对账（`kept / added /
  removed / failed`），写入 `locks/dependencies/<mcVersion>-<loader>.json`
- `mcmod build` 读取 lock、校验所有目标 jar（汇总 class 冲突、损坏 jar 和元数据
  必需依赖缺失），生成 client / server zip
- `mcmod lock release` 维护 `locks/releases/<mcVersion>.json`
- `mcmod tree` 渲染已解析的依赖树
- 跨平台二进制，可通过 `go install` 或 GitHub Releases 安装
- 也提供简短命令名 `mcm`（`go install ./cmd/mcm`）。
- `mcm version`、`mcm -v` 和 `mcm --version` 都会输出写死在代码中的 CLI 版本。

## 快速开始

```bash
# 1. 配置 CurseForge API 密钥（用户级，推荐）
mcmod set cf-key <your-key>

# 2. 写 packspec.json
cat > packspec.json <<'JSON'
{
  "packName": "my-pack",
  "serverPackName": "my-pack-server",
  "packVersion": "0.1.0",
  "minecraftVersion": "1.21.1",
  "loaderName": ["neoforge:21.1.219"],
  "mods": {
    "jei": {
      "name": "Just Enough Items",
      "scope": "client",
      "source": { "type": "curseforge", "query": "Just Enough Items" }
    },
    "create": {
      "name": "Create",
      "scope": "shared",
      "source": { "type": "curseforge", "query": "Create" }
    }
  }
}
JSON

# 3. 解析并锁定依赖
mcmod lock 1.21.1 neoforge

# 4. 构建 client + server zip
mcmod build 1.21.1 neoforge
```

### Git packspec 递归依赖

当一个仓库发布可复用的 `packspec.json` 时，可以这样引用：

```json
{
  "mods": {
    "shared-bundle": {
      "scope": "shared",
      "source": { "type": "git", "repo": "owner/shared-bundle" }
    }
  }
}
```

执行 `mcmod lock` 时会递归展开嵌套 Git 包，按 loader 过滤，然后解析其中的
非 Git mod。展开后的 key 会在 lock 中按仓库命名空间隔离，不会回写根目录
`packspec.json`。Git 仓库是 packspec 输入，不是 jar 下载源。

产物落在 `locks/dependencies/1.21.1-neoforge.json` 和
`releases/v0.1.0/my-pack-1.21.1-neoforge-21.1.219-{client,server}.zip`。

## 命令一览

| 命令                                          | 说明                                       |
|-----------------------------------------------|--------------------------------------------|
| `mcmod set cf-key <key> [--project]`          | 配置 CurseForge API 密钥                   |
| `mcmod list`                                  | 按 scope 分组列出 `packspec.json` 中的 mod |
| `mcmod lock [<mc>] [<loader>]`                | 解析 / 增量更新依赖锁                      |
| `mcmod lock list\|show\|add\|update\|delete\|tree` | 管理 lock 文件和条目                  |
| `mcmod lock release set\|list\|show\|delete`  | 维护 `locks/releases/<mcVersion>.json`     |
| `mcmod build [<mc>] [<loader>]`               | 根据 lock 构建 client 和 server zip        |
| `mcmod tree [<mc>] [<loader>]`                | 渲染已解析的依赖树                         |
| `mcmod validate`                              | 校验 packspec / lock / release index       |
| `mcmod --help`                                | 显示完整命令树（见 [docs/000-index.md](./docs/000-index.md)） |

完整命令树（含位置参数）请运行 `mcmod --help`。

## 发布

可以手动运行 `Bump Stable Release Version` workflow，选择 `major`、
`minor`（默认）或 `patch`；也可以填写可选的 `base_version`（`x.y.z`）作为递增基准。
workflow 会创建带 `release` label 的版本 PR。
PR 合入后，`Tag Stable Release` 会创建 `vX.Y.Z` tag，随后并行使用原生
Linux amd64/arm64、Windows amd64、macOS amd64/arm64 runner 构建并发布 GitHub Release。

`Publish Beta Release` workflow 也支持可选的 `base_version`，会直接创建 `vX.Y.Z-canary.N` tag，并使用相同的平台矩阵发布 prerelease。

## 项目目录

```text
packspec.json                       # 人工编辑的规范
locks/
  dependencies/<mc>-<loader>.json   # 解析后的 lock 文件
  releases/<mc>.json                # 构建发布索引
releases/                           # 构建产物 zip（不提交）
.mcmod/config.json                  # 项目级 CurseForge key（不提交）
.mcmod/cache/                       # jar 与 resolver 缓存（不提交）
.mcmod/cache/resolved/              # resolver id 缓存（不提交）
internal/
  cli/                              # cobra 命令
  domain/                           # 数据模型、校验、存储
  resolver/                         # 来源解析（CF、GitHub、Git、Local）
  downloader/                       # 带缓存的 jar 下载器
  metadata/                         # jar 元数据（NeoForge / Fabric TOML、fabric.mod.json）
  graph/                            # 依赖图与版本决议
  service/                          # 业务逻辑（lock、build、release、tree）
  cache/                            # 缓存辅助
  config/                           # 项目 / 环境配置
cmd/mcmod/                          # CLI 入口
```

## 文档

- [docs/000-index.md](./docs/000-index.md) — 文档索引与阅读顺序
- [docs/001-spec.md](./docs/001-spec.md) — `packspec.json` schema 与规则
- [docs/002-cli-overview.md](./docs/002-cli-overview.md) — 命令总览与退出码
- [docs/003-lock-files.md](./docs/003-lock-files.md) — 依赖 lock 格式与对账规则
- [docs/004-release-index.md](./docs/004-release-index.md) — 发布索引格式
- [docs/005-source-resolution.md](./docs/005-source-resolution.md) — source 解析方式
- [docs/008-build-pipeline.md](./docs/008-build-pipeline.md) — 构建流水线与校验

## TODO

`mcmod build` 目前只打包 `mods/` 目录。以下整合包产物仍在 TODO 列表,
尚未实现:

- **Mod** — 已完整支持。schema 与 lock/release 文档已说明。
- **Shaderpack** (`shaderpacks/`) — 不支持。`packspec.json` 暂未提供
  shaderpack 的 source 类型,resolver 也不会处理,zip 中不会写入对应目录。
- **Resourcepack** (`resourcepacks/`) — 如果项目根目录存在 `resourcepacks/`,
  客户端 zip 会原样拷贝进去;但 resolver 与 lock 流水线目前不会跟踪
  packspec 级的 resourcepack 条目。
- **Datapack** (`datapacks/`) — 不支持。
- **世界存档** — 不支持。
- **CurseForge 整合包布局** (`manifest.json` + `modlist.html` +
  `overrides/{config,resourcepacks}/`) — 已实现。`mcmod build
  --build-type cf` 产出 manifest-only 的 zip(不带 mod jar;启动器导入时
  按 `manifest.files[]` 自行下载)。详细契约见 `docs/002-cli-overview.md`。
- **Modrinth `.mrpack` 布局** — 文档中保留,CLI 暂不接受。

请在 issue 中跟进;按 "docs win" 规则,关闭其中任一项的 PR 必须同时更新
`docs/002-cli-overview.md` 与 `AGENTS.md`。

## 构建

```bash
go build ./cmd/mcmod
go test ./...
```

需要 Go 1.26+。

## 许可证

MIT
