# __TEMPLATE_REPO__

Go 微服务（`common` / `client` / `server` 三模块）。

## 模块路径

```
github.com/__TEMPLATE_ORG__/__TEMPLATE_REPO__/{common|client|server}
```

## 本地开发

```bash
export MYSQL_PASSWORD=your_password
cd server && go run main.go
```

## API

| 类型 | 地址 |
|------|------|
| 健康检查 | `GET /__SERVICE_SLUG__/v1/ping` |
| Swagger | `/__SERVICE_SLUG__/v1/swagger/index.html` |
| gRPC | `__GRPC_SERVICE__`（端口 `5001`） |
| HTTP | 端口 `8080` |

## 配置

- Go：`__GO_VERSION__`
- MySQL 库名：`__DB_NAME__`
- Proto：`common/proto/__PROTO_FILE__.proto`（package `__PROTO_PACKAGE__`）
