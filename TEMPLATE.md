# Go 微服务模板说明

本仓库为 **GitHub Template**，用于一键创建新的 Go 微服务。业务仓库初始化后请阅读根目录 `README.md`；本文档仅在模板仓库中保留。

## 项目结构

```
.
├── common/          # protobuf、gRPC 公共定义
├── client/          # gRPC 客户端 SDK
├── server/          # HTTP + gRPC 主服务
├── scripts/         # 初始化与 CI 辅助脚本
└── .github/workflows/
```

## 占位符

由 `init-from-template` workflow 自动替换（不含 `.github/workflows/`）：

| 占位符 | 说明 | 示例（`order-mservice`） |
|--------|------|--------------------------|
| `__TEMPLATE_ORG__` | GitHub org / 用户名 | `lianjin` |
| `__TEMPLATE_REPO__` | 仓库名 | `order-mservice` |
| `__GO_VERSION__` | Go 版本 | `1.25.10` |
| `__SERVICE_SLUG__` | HTTP 路由前缀 | `order-ms` |
| `__DB_NAME__` | MySQL 数据库名 | `order_ms_db` |
| `__SONAR_PROJECT_KEY__` | Sonar 项目 Key | `lianjin_order-mservice` |
| `__PROTO_FILE__` | proto 文件名（domain） | `order` |
| `__PROTO_PACKAGE__` | proto Go 包名 | `orderpb` |
| `__GRPC_SERVICE__` | gRPC service 名称 | `OrderService` |

仓库名约定为 `<domain>-mservice`。CI workflow 使用 `go-version-file: server/go.mod` 与 `${{ github.repository }}` 等表达式，**不在 init 时修改**（避免 `GITHUB_TOKEN` 无法 push workflow）。

## 创建新微服务

### 方式一：GitHub Template（推荐）

1. 点击 **Use this template** 创建新仓库（建议命名为 `xxx-mservice`）
2. 首次 push 到 `main` 后自动运行 **Init from template**
3. 替换占位符、发布 `common/v0.0.1` tag、`go get` 关联 client/server，并删除模板专用文件（`TEMPLATE.md`、`init-from-template.yml`、`provision-microservice.yml` 及 `scripts/replace-template-vars.sh` 等 init 脚本；保留 `scripts/gotest-summary.sh` 供 CI 使用）

### 方式二：手动触发

**Actions → Init from template → Run workflow**，可自定义 `org` / `repo` / `go_version` / `service_slug` / `db_name`。

### 方式三：从模板仓库远程触发

在模板仓库运行 **Provision microservice**，向目标仓库发送 `repository_dispatch`。

需配置 secret：`REPO_DISPATCH_TOKEN`（对目标仓库有 `contents` 写权限的 PAT）。

### 本地替换

```bash
chmod +x scripts/replace-template-vars.sh
./scripts/replace-template-vars.sh <org> <repo> <go_version> [service_slug] [db_name]
```

## CI/CD

| Workflow | 说明 |
|----------|------|
| `build.yml` | build + test + lint |
| `codeql.yml` | CodeQL |
| `sonar.yml` | SonarQube |
| `trivy.yml` | Trivy |
| `deploy.yml` | Railway 手动部署 |
