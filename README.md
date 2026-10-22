# mail-api

最基础的 Gin HTTP 服务。

- 模块名：`mail-api`
- 默认监听：`:9998`
- 健康检查：`GET /health`

## 运行

```bash
go run .
```

可通过环境变量或参数改监听地址：

```bash
go run . -addr :9090
```

```bash
ADDR=:9090 go run .
```

`GIN_MODE` 默认 `release`，本地调试可设为 `debug`。

收到 `SIGINT` / `SIGTERM` 后，进程在 8 秒内优雅退出。

## 发送邮件

SMTP 主机、端口、账号、密码和加密方式都写在 `config.yaml`。`provider` 为 `mail` 或 `aliyun`，只会读取对应那一段。发件人使用该段的 `username`。

`POST /email` 只把收件人放入队列，返回 `{"status":"queued"}`。单个消费协程按顺序发送：任意 100 秒最多 150 封，每个自然日最多 4500 封。发送失败会留在队列里重试。

```bash
go run .
```
