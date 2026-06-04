<div align="center">
  <h1>Zebra-CICD</h1>
  <span>中文 | <a href="./README.en.md">English</a></span>
</div>

## 📖 项目简介

Zebra-CICD 是 ZebraOps 云原生运维平台的 CI/CD 管理服务，基于 Go 1.25 + Gin 开发。它统一管理代码仓库、构建模板、镜像仓库、部署模板、环境及 Kubernetes 集群，通过 Jenkins 触发构建、Harbor 存储镜像、Kubernetes 执行部署，并将整个流程编排为异步任务队列（Redis + Asynq），同时提供 WebSocket 实时推送构建日志。

## ✨ 功能特性

- **多资源管理**：代码仓库、应用服务、构建/部署模板、云厂商、镜像仓库、环境、K8s 集群、Linux 主机一站式管理
- **全链路部署**：GitLab 拉取代码 → Jenkins 触发构建 → Harbor 推送镜像 → Kubernetes 滚动发布
- **异步任务队列**：基于 Redis + Asynq，支持并发部署任务与指数退避重试
- **WebSocket 日志**：实时推送 Jenkins 构建日志至前端
- **Nacos 配置中心**：可选集成 Nacos，动态下发数据库连接、Token 等敏感配置
- **链路追踪**：集成 OpenZipkin，记录跨服务调用链路
- **结构化日志**：Zap + Lumberjack，支持 JSON 格式日志及自动轮转
- **接口文档**：Swagger UI，访问 `/docs`

## 🛠️ 技术栈

| 类别         | 组件                                          |
| ------------ | --------------------------------------------- |
| 后端框架     | Go 1.25 + Gin                                 |
| 数据库       | PostgreSQL + GORM v2                          |
| 任务队列     | Redis + Asynq（hibiken/asynq）                |
| 配置管理     | Viper（YAML 文件 + 环境变量）+ Nacos（可选）  |
| 日志         | Zap + Lumberjack                              |
| 外部集成     | GitLab API、Jenkins API、Harbor API、K8s SDK  |
| 实时通信     | WebSocket（gorilla/websocket）                |
| 链路追踪     | OpenZipkin（openzipkin/zipkin-go）            |
| API 文档     | Swaggo + Swagger UI                           |

## 🌳 目录结构

```text
ZebraCICD/
├── config/               # 配置加载（Viper）
│   ├── config.go         #   Config 结构体定义与环境变量映射
│   └── configs.yaml      #   本地默认配置
├── docs/                 # Swagger 文档（swag init 生成）
├── internal/
│   ├── api/              # Gin 路由注册 & 请求绑定
│   ├── core/             # 外部系统客户端（GitLab / Jenkins / Harbor / K8s）
│   ├── handler/          # GORM 数据库 CRUD
│   ├── model/            # 数据模型（GORM + JSON）
│   ├── service/          # 业务编排（构建、部署、应用等）
│   ├── types/            # 公共类型与统一响应结构
│   └── worker/           # Asynq Worker（异步部署任务处理）
├── pkg/
│   ├── log/              # Zap 日志初始化封装
│   ├── middleware/       # 请求日志中间件
│   ├── nacos/            # Nacos 客户端 & 配置加载器
│   ├── queue/            # Asynq Client / Server 封装
│   ├── ssh/              # SSH 客户端（Linux 主机操作）
│   └── timeutil/         # 时间工具（JSON 序列化格式）
├── scripts/              # 辅助脚本
├── main.go               # 服务入口
└── go.mod
```

## ⚡ 快速开始

### 前置依赖

- Go 1.25+
- PostgreSQL（服务运行时自动建表）
- Redis（任务队列，默认 `localhost:6379`）

### 配置

所有配置均可通过 `config/configs.yaml` 或环境变量覆盖，环境变量前缀为 `ZEBRA_`：

| 配置项（YAML 路径）         | 环境变量                  | 说明                 | 默认值                        |
| --------------------------- | ------------------------- | -------------------- | ----------------------------- |
| `app.Port`                  | `ZEBRA_APP_PORT`          | 服务端口             | `4123`                        |
| `app.DatabaseURL`           | `ZEBRA_APP_DATABASEURL`   | PostgreSQL 连接串    | —                             |
| `app.GitLabToken`           | `ZEBRA_APP_GITLABTOKEN`   | GitLab Private Token | —                             |
| `app.GitLabURL`             | `ZEBRA_APP_GITLABURL`     | GitLab 地址          | `https://gitlab.com`          |
| `app.HarborURL`             | `ZEBRA_APP_HARBORURL`     | Harbor 镜像仓库地址  | —                             |
| `app.JenkinsURL`            | `ZEBRA_APP_JENKINSURL`    | Jenkins 地址         | —                             |
| `app.JenkinsUser`           | `ZEBRA_APP_JENKINSUSER`   | Jenkins 用户名       | —                             |
| `app.JenkinsPass`           | `ZEBRA_APP_JENKINSPASS`   | Jenkins 密码/Token   | —                             |
| `redis.addr`                | `ZEBRA_REDIS_ADDR`        | Redis 地址           | —                             |
| `redis.password`            | `ZEBRA_REDIS_PASSWORD`    | Redis 密码           | —                             |
| `worker.concurrency`        | `ZEBRA_WORKER_CONCURRENCY`| Worker 并发数        | `3`                           |

### 运行

```sh
# 1. 下载依赖
go mod tidy

# 2. 修改 config/configs.yaml 填入数据库、GitLab、Jenkins、Harbor 信息
# 3. 启动服务
go run main.go

# 或使用启动脚本（会自动配置 Nacos 相关环境变量）
./start.sh
```

服务启动后访问：
- API 文档：http://127.0.0.1:4123/docs
- Swagger JSON：http://127.0.0.1:4123/docs/swagger.json

### 更新 Swagger 文档

```sh
go install github.com/swaggo/swag/cmd/swag@latest
swag init -g main.go
```

## 🔧 Nacos 配置中心（可选）

配置 Nacos 后，服务启动时将从 Nacos 拉取数据库连接串、GitLab Token 等敏感配置，覆盖本地 `configs.yaml`。

```sh
export NACOS_SERVER_ADDR="localhost:8848"
export NACOS_NAMESPACE="zebra-dev"   # 留空则使用 public 命名空间
export NACOS_USERNAME="nacos"
export NACOS_PASSWORD="nacos"
export NACOS_GROUP="DEFAULT_GROUP"
```

未设置 `NACOS_SERVER_ADDR` 时，Nacos 集成自动跳过，服务仅使用本地配置。

## ☸️ 对接 Kubernetes 集群

通过前端或 API 将集群的 `apiServer`、`token`、`CA 证书` 录入系统，部署时自动获取客户端。

首次接入集群，需创建专用 ServiceAccount 并获取 Token：

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: zebra-sa
  namespace: default
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: zebra-cluster-role
rules:
  - apiGroups: [""]
    resources: ["nodes", "pods", "pods/log", "services", "namespaces", "configmaps", "secrets", "events", "jobs", "cronjobs"]
    verbs: ["create", "get", "list", "watch", "update", "patch", "delete"]
  - apiGroups: ["apps"]
    resources: ["deployments", "statefulsets", "daemonsets"]
    verbs: ["create", "get", "list", "watch", "update", "patch", "delete"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: zebra-cluster-binding
subjects:
  - kind: ServiceAccount
    name: zebra-sa
    namespace: default
roleRef:
  kind: ClusterRole
  name: zebra-cluster-role
  apiGroup: rbac.authorization.k8s.io
```

```sh
# 获取 Token（Kubernetes 1.24 以下）
SECRET_NAME=$(kubectl get serviceaccount zebra-sa -o jsonpath='{.secrets[0].name}')
kubectl get secret $SECRET_NAME -o jsonpath='{.data.token}' | base64 -d

# Kubernetes 1.24+
kubectl create token zebra-sa --duration=87600h
```

## 🤝 贡献指南

欢迎提交 Pull Request，请确保：

- 代码通过 `go fmt` 格式化
- 新增 API 已更新 Swagger 注释并执行 `swag init`
- 关键业务逻辑有单元测试覆盖

如有问题请在 [Issues](https://github.com/ZebraOps/ZebraCICD/issues) 提交。

## 💬 联系方式

- 邮箱：iamnumachen@gmail.com
- GitHub Issue：[提交问题](https://github.com/ZebraOps/ZebraCICD/issues/new)

## 📄 License

[MIT](LICENSE)
