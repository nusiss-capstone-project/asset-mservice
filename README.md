# asset-mservice

Go 微服务（`common` / `client` / `server` 三模块）。

## 模块路径

```
github.com/nusiss-capstone-project/asset-mservice/{common|client|server}
```

## 本地开发

```bash
export MYSQL_PASSWORD=your_password
cd server && go run main.go
```

## API

| 类型 | 地址 |
|------|------|
| 健康检查 | `GET /asset-ms/v1/ping` |
| Swagger | `/asset-ms/v1/swagger/index.html` |
| gRPC | `AssetService`（端口 `5001`） |
| HTTP | 端口 `8080` |

## 配置

- Go：`1.25.10`
- MySQL 库名：`asset_ms_db`
- Proto：`common/proto/asset.proto`（package `assetpb`）
