<div align="center">
  <h1>Zebra CICD</h1>
  <span>中文 | <a href="./README.en.md">English</a></span>
  <br/>
  <strong>ZebraOps 云原生 CI/CD 管理服务</strong>
</div>

---

## 📖 项目简介

Zebra CICD 是 ZebraOps 云原生运维平台的持续集成与持续部署管理服务，基于 Go 1.25 + Gin 开发。它统一管理代码仓库、构建模板、镜像仓库、部署模板、环境及 Kubernetes 集群，通过 Jenkins 触发构建、镜像仓库存储镜像、Kubernetes/Docker/Linux 执行部署，并将整个流程编排为异步任务队列（Redis + Asynq），支持阶段历史追踪与失败重试。

## ✨ 功能特性

### 核心功能

- **多平台配置管理**：支持多 Jenkins 平台、多 Git 平台（GitLab/GitHub/Gitea）、多镜像仓库（Harbor/ACR/V2）的统一管理
- **多部署目标支持**：支持 Kubernetes、Docker Compose、Linux+Nginx 三种部署方式
- **应用生命周期管理**：从代码仓库、构建模板、部署模板到环境配置的全链路管理
- **凭据自动注入**：部署时自动将平台凭据注入 Jenkins，支持自动创建和手动选择两种模式

### 技术特性

- **异步任务队列**：基于 Redis + Asynq，支持并发部署任务与指数退避重试
- **阶段历史追踪**：记录 BUILDING → PUSHING → DEPLOYING 各阶段状态，支持失败重试
- **构建日志查询**：提供 Jenkins 控制台输出 API，前端可轮询获取实时日志
- **Nacos 配置中心**：可选集成 Nacos，动态下发数据库连接、超时配置、部署路径等
- **结构化日志**：Zap + Lumberjack，支持 JSON 格式日志及自动轮转
- **接口文档**：Swagger UI，访问 `/docs`

## 🛠️ 技术栈

| 类别           | 组件                                          |
| -------------- | --------------------------------------------- |
| 后端框架       | Go 1.25 + Gin                                 |
| 数据库         | PostgreSQL + GORM v2                          |
| 任务队列       | Redis + Asynq（hibiken/asynq）                |
| 配置管理       | Viper（YAML 文件 + 环境变量）+ Nacos（可选）  |
| 日志           | Zap + Lumberjack                              |
| 外部集成       | GitLab API、Jenkins API、Harbor API、K8s SDK  |
| API 文档       | Swaggo + Swagger UI                           |

## 🌳 目录结构

```text
ZebraCICD/
├── config/               # 配置加载（Viper）
│   ├── config.go         #   Config 结构体定义与环境变量映射
│   └── configs.yaml      #   本地默认配置
├── docs/                 # Swagger 文档（swag init 生成）
├── internal/
│   ├── api/              # Gin 路由注册 & 请求绑定
│   ├── core/             # 外部系统客户端（GitLab / Jenkins / Registry / K8s）
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
- PostgreSQL 14+
- Redis 6+
- Jenkins（用于构建触发）
- Nacos 2.x（可选，用于配置中心）

### 配置

所有配置均可通过 `config/configs.yaml` 或环境变量覆盖，环境变量前缀为 `ZEBRA_`：

#### 基础配置

| 配置项（YAML 路径）         | 环境变量                  | 说明                 | 默认值                        |
| --------------------------- | ------------------------- | -------------------- | ----------------------------- |
| `app.Port`                  | `ZEBRA_APP_PORT`          | 服务端口             | `4123`                        |
| `app.DatabaseURL`           | `ZEBRA_APP_DATABASEURL`   | PostgreSQL 连接串    | —                             |
| `app.GitLabToken`           | `ZEBRA_APP_GITLABTOKEN`   | GitLab Private Token | —                             |
| `app.GitLabURL`             | `ZEBRA_APP_GITLABURL`     | GitLab 地址          | `https://gitlab.com`          |
| `app.JenkinsURL`            | `ZEBRA_APP_JENKINSURL`    | Jenkins 地址         | —                             |
| `app.JenkinsUser`           | `ZEBRA_APP_JENKINSUSER`   | Jenkins 用户名       | —                             |
| `app.JenkinsPass`           | `ZEBRA_APP_JENKINSPASS`   | Jenkins 密码/Token   | —                             |
| `redis.addr`                | `ZEBRA_REDIS_ADDR`        | Redis 地址           | `127.0.0.1:6379`              |
| `redis.password`            | `ZEBRA_REDIS_PASSWORD`    | Redis 密码           | —                             |
| `worker.concurrency`        | `ZEBRA_WORKER_CONCURRENCY`| Worker 并发数        | `3`                           |

#### 超时与路径配置（可通过 Nacos 动态下发）

| 配置项                     | 说明                     | 默认值               |
| -------------------------- | ------------------------ | -------------------- |
| `jenkins.build_wait_timeout`  | Jenkins 构建等待超时    | `10m`                |
| `jenkins.build_poll_interval` | Jenkins 构建轮询间隔    | `10s`                |
| `timeout.gitlab_http`         | GitLab HTTP 超时        | `15s`                |
| `timeout.jenkins_http`        | Jenkins HTTP 超时       | `30s`                |
| `timeout.ssh_connect`         | SSH 连接超时            | `10s`                |
| `paths.deploy_base`           | Docker 部署基础路径     | `/opt/zebra-deploy`  |
| `paths.nginx_conf`            | Nginx 配置目录          | `/etc/nginx/conf.d`  |

### 运行

```sh
# 1. 下载依赖
go mod tidy

# 2. 修改 config/configs.yaml 填入数据库、GitLab、Jenkins 信息
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

配置 Nacos 后，服务启动时将从 Nacos 拉取配置，覆盖本地 `configs.yaml`。

### 环境变量配置

```sh
export NACOS_SERVER_ADDR="localhost:8848"
export NACOS_NAMESPACE="zebra-dev"   # 留空则使用 public 命名空间
export NACOS_USERNAME="nacos"
export NACOS_PASSWORD="nacos"
export NACOS_GROUP="DEFAULT_GROUP"
```

### Nacos 配置文件示例

在 Nacos 中创建 `zebra-cicd.yaml`（Data ID），内容如下：

```yaml
# ==================== 数据库配置 ====================
database:
  url: "postgres://postgres:password@192.168.1.100:5432/postgres?sslmode=disable"

# ==================== GitLab 配置 ====================
gitlab:
  url: "https://git.example.com"
  token: "your-gitlab-token"
  timeout: "30s"

# ==================== Jenkins 配置 ====================
jenkins:
  url: "http://192.168.1.100:8080/"
  username: "admin"
  password: "your-jenkins-api-token"
  timeout: "60s"
  build_wait_timeout: "10m"
  build_poll_interval: "10s"
  default_credentials:
    registry: "registry-creds"
    git: "gitlab-creds"

# ==================== 镜像仓库配置 ====================
registry:
  url: "registry.example.com"
  username: "admin"
  password: "your-registry-password"

# ==================== Redis 配置 ====================
redis:
  addr: "192.168.1.100:6379"
  password: ""
  db: 0

# ==================== 超时配置 ====================
timeout:
  gitlab_http: "15s"
  jenkins_http: "30s"
  ssh_connect: "10s"

# ==================== 路径配置 ====================
paths:
  deploy_base: "/opt/zebra-deploy"
  nginx_conf: "/etc/nginx/conf.d"

# ==================== 日志配置 ====================
logging:
  level: "info"
  encoding: "json"
  max_size: 100
  max_age: 30
  max_backups: 10
  compress: true
```

未设置 `NACOS_SERVER_ADDR` 时，Nacos 集成自动跳过，服务仅使用本地配置。

## 📦 数据模型

| 模型                      | 说明                          |
| ------------------------- | ----------------------------- |
| `Application`             | 应用服务定义                  |
| `ApplicationDeployment`   | 应用部署配置（关联平台、模板） |
| `Repo`                    | 代码仓库                      |
| `BuildTemplate`            | Jenkins Pipeline 构建模板     |
| `DeploymentTemplate`       | 部署模板（K8s/Docker/Linux） |
| `Environment`              | 部署环境（dev/test/prod）     |
| `K8SCluster`              | Kubernetes 集群配置           |
| `Server`                  | Linux 主机配置                |
| `GitPlatform`             | Git 平台配置（GitLab/GitHub） |
| `JenkinsPlatform`         | Jenkins 平台配置              |
| `ImageRepository`         | 镜像仓库配置                  |
| `DeployTask`              | 部署任务                      |
| `StageHistory`            | 部署阶段历史                  |
| `Language`                | 开发语言                      |
| `CloudProvider`           | 云厂商                        |

## 🚀 部署流程

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   GitLab    │───▶│   Jenkins   │───▶│   Registry  │───▶│   K8s/Docker│
│  (代码拉取) │    │  (构建镜像) │    │  (推送镜像) │    │  (滚动部署) │
└─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘
       ▲                  ▲                  ▲                  ▲
       │                  │                  │                  │
       └──────────────────┴──────────────────┴──────────────────┘
                              Zebra CICD 统一编排
```

1. **BUILDING**：触发 Jenkins Pipeline 构建 Docker 镜像
2. **PUSHING**：验证镜像已推送到镜像仓库
3. **DEPLOYING**：根据部署目标执行部署
   - **K8s**：通过 Server-Side Apply 部署 Deployment/Service/ConfigMap
   - **Docker**：通过 SSH 上传 docker-compose.yml 并启动
   - **Linux**：通过 SSH 提取静态文件 + 配置 Nginx

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
    resources: ["nodes", "pods", "pods/log", "services", "namespaces", "configmaps", "secrets", "events"]
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

## 📡 API 端点

| 模块           | 路径前缀         | 说明               |
| -------------- | ---------------- | ------------------ |
| 应用管理       | `/api/applications` | 应用 CRUD        |
| 部署任务       | `/api/tasks`       | 任务创建、查询、重试 |
| 仓库管理       | `/api/repos`        | 代码仓库管理     |
| 构建模板       | `/api/build-templates` | Jenkins Pipeline 模板 |
| 部署模板       | `/api/deploy-templates` | K8s/Docker/Linux 模板 |
| K8s 集群       | `/api/k8s/clusters` | 集群配置管理     |
| Linux 主机     | `/api/servers`      | 主机配置管理     |
| Jenkins 平台   | `/api/jenkins-platforms` | 多 Jenkins 实例 |
| Git 平台       | `/api/git-platforms` | 多 Git 源管理    |
| 镜像仓库       | `/api/image-repositories` | 多仓库管理   |
| 环境           | `/api/environments` | 环境管理         |
| 开发语言       | `/api/languages`    | 语言管理         |

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
