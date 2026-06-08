<div align="center">
  <img src="docs/logo.png" alt="ZebraCICD Logo" width="120" height="120">

  <h1>ZebraCICD</h1>

  <p>
    <strong>ZebraOps 云原生 CI/CD 管理服务</strong>
  </p>

  <p>
    <a href="#-功能特性">功能特性</a> •
    <a href="#-快速开始">快速开始</a> •
    <a href="#-架构设计">架构设计</a> •
    <a href="#-api-端点">API 端点</a> •
    <a href="#-开发指南">开发指南</a>
  </p>

  <p>
    <a href="./README.md">中文</a> | <a href="./README.en.md">English</a>
  </p>

  <p>
    <img src="https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go" alt="Go Version">
    <img src="https://img.shields.io/badge/Gin-1.11-00ADD8?style=flat" alt="Gin Version">
    <img src="https://img.shields.io/badge/PostgreSQL-14+-336791?style=flat&logo=postgresql" alt="PostgreSQL">
    <img src="https://img.shields.io/badge/Redis-6+-DC382D?style=flat&logo=redis" alt="Redis">
    <img src="https://img.shields.io/badge/License-MIT-green?style=flat" alt="License">
  </p>
</div>

---

## 📖 项目简介

ZebraCICD 是 [ZebraOps](https://github.com/ZebraOps) 云原生运维平台的持续集成与持续部署管理服务。它统一管理代码仓库、构建模板、镜像仓库、部署模板、环境及 Kubernetes 集群，通过 Jenkins 触发构建、镜像仓库存储镜像、Kubernetes/Docker/Linux 执行部署，并将整个流程编排为异步任务队列，支持阶段历史追踪与失败重试。

### 核心能力

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

---

## ✨ 功能特性

### 🎯 核心功能

| 功能 | 描述 |
|------|------|
| **多平台配置管理** | 支持多 Jenkins 平台、多 Git 平台（GitLab/GitHub/Gitea）、多镜像仓库（Harbor/ACR/V2）的统一管理 |
| **多部署目标支持** | 支持 Kubernetes、Docker Compose、Linux+Nginx 三种部署方式 |
| **应用生命周期管理** | 从代码仓库、构建模板、部署模板到环境配置的全链路管理 |
| **凭据自动注入** | 部署时自动将平台凭据注入 Jenkins，支持自动创建和手动选择两种模式 |

### 🔄 部署流程

```
PENDING → BUILDING → PUSHING → DEPLOYING → SUCCESS/FAILED
            │           │           │
            ▼           ▼           ▼
        Jenkins     Registry     K8s/Docker/Linux
        构建        验证          部署
```

| 阶段 | 说明 |
|------|------|
| **BUILDING** | 触发 Jenkins Pipeline 构建 Docker 镜像，自动注入 Git/Registry 凭据 |
| **PUSHING** | 验证镜像已推送到仓库 |
| **DEPLOYING** | 根据部署目标执行部署：K8s (Server-Side Apply)、Docker (SSH + compose)、Linux (SSH + Nginx) |

### 🛠️ 技术特性

| 特性 | 实现方式 |
|------|----------|
| 异步任务队列 | Redis + Asynq，支持并发部署与指数退避重试 |
| 阶段历史追踪 | 记录 BUILDING → PUSHING → DEPLOYING 各阶段状态 |
| 构建日志查询 | Jenkins 控制台输出 API，支持前端轮询 |
| 配置中心 | 可选集成 Nacos，动态下发配置 |
| 结构化日志 | Zap + Lumberjack，JSON 格式 + 自动轮转 |
| API 文档 | Swagger UI，访问 `/docs` |

---

## 🛠️ 技术栈

| 类别 | 组件 | 版本 |
|------|------|------|
| 后端框架 | Go + Gin | 1.25+ / 1.11 |
| 数据库 | PostgreSQL + GORM | 14+ / v2 |
| 任务队列 | Redis + Asynq | 6+ / v0.26 |
| 配置管理 | Viper + Nacos（可选） | 1.21 |
| 日志 | Zap + Lumberjack | 1.27 |
| 外部集成 | GitLab API、Jenkins API、Harbor API、K8s SDK | - |
| API 文档 | Swaggo + Swagger UI | 1.16 |

---

## 🌳 目录结构

```text
ZebraCICD/
├── config/                   # 配置层
│   ├── config.go             #   配置加载（Viper）
│   └── configs.yaml          #   本地默认配置
├── docs/                     # Swagger 文档（swag init 生成）
├── internal/                 # 内部模块
│   ├── api/                  #   Gin 路由注册 & 请求绑定
│   ├── core/                 #   外部系统客户端（GitLab/Jenkins/Registry/K8s）
│   ├── handler/              #   GORM 数据库 CRUD（Repository 层）
│   ├── model/                #   数据模型（GORM 实体）
│   ├── service/              #   业务编排（构建、部署、应用等）
│   ├── types/                #   公共类型与统一响应结构
│   └── worker/               #   Asynq Worker（异步任务处理）
├── pkg/                      # 公共包
│   ├── log/                  #   Zap 日志封装
│   ├── middleware/           #   HTTP 中间件
│   ├── nacos/                #   Nacos 客户端 & 配置加载器
│   ├── queue/                #   Asynq Client/Server 封装
│   ├── ssh/                  #   SSH 客户端（远程主机操作）
│   └── timeutil/             #   时间工具
├── scripts/                  # 辅助脚本
├── logs/                     # 日志输出目录
├── main.go                   # 服务入口
├── start.sh                  # 启动脚本（含 Nacos 环境变量）
└── go.mod                    # Go 模块定义
```

---

## 🏗️ 架构设计

### 分层架构

```
┌─────────────────────────────────────────────────────────────────────┐
│                         HTTP Request                                 │
└─────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│  API Layer (internal/api/)                                          │
│  ├── Gin Router 注册                                                 │
│  ├── 请求参数绑定与验证                                               │
│  └── 响应封装                                                        │
└─────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│  Service Layer (internal/service/)                                  │
│  ├── 业务逻辑编排                                                    │
│  ├── 外部系统调用协调                                                │
│  └── 事务管理                                                        │
└─────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│  Handler Layer (internal/handler/)                                  │
│  ├── Repository 模式                                                 │
│  ├── GORM 数据库操作                                                 │
│  └── 查询构建                                                        │
└─────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│  Model Layer (internal/model/)                                      │
│  ├── GORM 实体定义                                                   │
│  └── 数据库表映射                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 核心集成

```
┌─────────────────────────────────────────────────────────────────────┐
│                        ZebraCICD Core                                │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │
│  │  GitClient   │  │JenkinsClient │  │RegistryClient│              │
│  │              │  │              │  │              │              │
│  │ • GitLab     │  │ • Job CRUD   │  │ • V2 API     │              │
│  │ • GitHub     │  │ • Build      │  │ • Harbor     │              │
│  │ • Gitea      │  │ • Credential │  │ • ACR        │              │
│  └──────────────┘  └──────────────┘  └──────────────┘              │
│                                                                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │
│  │  K8sClient   │  │  SSHClient   │  │ QueueClient  │              │
│  │              │  │              │  │              │              │
│  │ • Deploy     │  │ • Command    │  │ • Asynq      │              │
│  │ • Pods       │  │ • SFTP       │  │ • Redis      │              │
│  │ • Logs       │  │ • Nginx      │  │ • Retry      │              │
│  └──────────────┘  └──────────────┘  └──────────────┘              │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## ⚡ 快速开始

### 前置依赖

| 依赖 | 版本要求 | 用途 |
|------|---------|------|
| Go | 1.25+ | 运行时环境 |
| PostgreSQL | 14+ | 数据存储 |
| Redis | 6+ | 任务队列 |
| Jenkins | 2.x | 构建触发（可选） |
| Nacos | 2.x | 配置中心（可选） |

### 安装运行

```bash
# 1. 克隆项目
git clone https://github.com/ZebraOps/ZebraCICD.git
cd ZebraCICD

# 2. 下载依赖
go mod tidy

# 3. 配置数据库连接（二选一）

# 方式一：修改 config/configs.yaml
vim config/configs.yaml

# 方式二：使用环境变量
export ZEBRA_APP_DATABASEURL="postgres://user:pass@host:5432/db?sslmode=disable"

# 4. 启动服务

# 本地开发（无 Nacos）
go run main.go

# 或使用启动脚本（含 Nacos 配置）
./start.sh
```

### 服务端口

| 服务 | 端口 |
|------|------|
| ZebraCICD API | 4123 |
| Swagger UI | http://localhost:4123/docs |
| PostgreSQL | 5432 |
| Redis | 6379 |
| Nacos | 8848 |

---

## ⚙️ 配置说明

### 配置方式

ZebraCICD 支持三种配置方式，优先级从高到低：

1. **环境变量** - 格式：`ZEBRA_<SECTION>_<KEY>`，如 `ZEBRA_APP_PORT`
2. **Nacos 配置中心** - 动态配置，支持热更新
3. **YAML 配置文件** - `config/configs.yaml`，本地默认值

### 环境变量

#### 基础配置

| 环境变量 | 说明 | 默认值 |
|----------|------|--------|
| `ZEBRA_APP_PORT` | 服务端口 | `4123` |
| `ZEBRA_APP_DATABASEURL` | PostgreSQL 连接串 | - |

#### Git 平台配置

| 环境变量 | 说明 | 默认值 |
|----------|------|--------|
| `ZEBRA_APP_GITLABURL` | GitLab 地址 | `https://gitlab.com` |
| `ZEBRA_APP_GITLABTOKEN` | GitLab Private Token | - |

#### Jenkins 配置

| 环境变量 | 说明 | 默认值 |
|----------|------|--------|
| `ZEBRA_APP_JENKINSURL` | Jenkins 地址 | - |
| `ZEBRA_APP_JENKINSUSER` | Jenkins 用户名 | - |
| `ZEBRA_APP_JENKINSPASS` | Jenkins 密码/Token | - |

#### Redis 配置

| 环境变量 | 说明 | 默认值 |
|----------|------|--------|
| `ZEBRA_REDIS_ADDR` | Redis 地址 | `127.0.0.1:6379` |
| `ZEBRA_REDIS_PASSWORD` | Redis 密码 | - |
| `ZEBRA_REDIS_DB` | Redis 数据库 | `0` |

#### Worker 配置

| 环境变量 | 说明 | 默认值 |
|----------|------|--------|
| `ZEBRA_WORKER_CONCURRENCY` | Worker 并发数 | `3` |

#### Nacos 配置（可选）

| 环境变量 | 说明 | 默认值 |
|----------|------|--------|
| `NACOS_SERVER_ADDR` | Nacos 服务地址 | - |
| `NACOS_NAMESPACE` | 命名空间 | `public` |
| `NACOS_USERNAME` | 用户名 | `nacos` |
| `NACOS_PASSWORD` | 密码 | `nacos` |
| `NACOS_GROUP` | 配置分组 | `DEFAULT_GROUP` |

### Nacos 配置示例

在 Nacos 中创建 `zebra-cicd.yaml`（Data ID）：

```yaml
# 数据库配置
database:
  url: "postgres://postgres:password@192.168.1.100:5432/postgres?sslmode=disable"

# GitLab 配置
gitlab:
  url: "https://git.example.com"
  token: "your-gitlab-token"
  timeout: "30s"

# Jenkins 配置
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

# 镜像仓库配置
registry:
  url: "registry.example.com"
  username: "admin"
  password: "your-registry-password"

# Redis 配置
redis:
  addr: "192.168.1.100:6379"
  password: ""
  db: 0

# 超时配置
timeout:
  gitlab_http: "15s"
  jenkins_http: "30s"
  ssh_connect: "10s"

# 路径配置
paths:
  deploy_base: "/opt/zebra-deploy"
  nginx_conf: "/etc/nginx/conf.d"

# 日志配置
logging:
  level: "info"
  encoding: "json"
  max_size: 100
  max_age: 30
  max_backups: 10
  compress: true
```

---

## 📦 数据模型

### 核心模型

| 模型 | 说明 | 关键字段 |
|------|------|----------|
| `Application` | 应用服务定义 | name, repo_id, language_id |
| `ApplicationDeployment` | 应用部署配置 | app_id, env_id, cluster_id, template_id |
| `DeployTask` | 部署任务 | status, stage, retry_count |
| `StageHistory` | 阶段历史 | task_id, stage, status, duration |
| `BuildTemplate` | 构建模板 | name, jenkinsfile, dockerfile |
| `DeploymentTemplate` | 部署模板 | name, content, deploy_target |

### 平台配置模型

| 模型 | 说明 | 关键字段 |
|------|------|----------|
| `GitPlatform` | Git 平台配置 | name, url, token, platform_type |
| `JenkinsPlatform` | Jenkins 平台配置 | name, url, username, password |
| `ImageRepository` | 镜像仓库配置 | name, url, registry_type |
| `K8SCluster` | K8s 集群配置 | name, api_server, token |
| `Server` | Linux 主机配置 | name, host, ssh_port |

### 辅助模型

| 模型 | 说明 |
|------|------|
| `Environment` | 部署环境（dev/test/prod） |
| `Language` | 开发语言定义 |
| `CloudProvider` | 云厂商配置 |

---

## 📡 API 端点

### 应用与部署

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/applications` | 应用列表 |
| POST | `/api/applications` | 创建应用 |
| GET | `/api/applications/:id` | 应用详情 |
| PUT | `/api/applications/:id` | 更新应用 |
| DELETE | `/api/applications/:id` | 删除应用 |
| GET | `/api/applications/:id/deployments` | 应用部署配置列表 |
| POST | `/api/applications/:id/deployments` | 创建部署配置 |

### 部署任务

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/deploys` | 任务列表 |
| POST | `/api/deploys` | 创建任务 |
| GET | `/api/deploys/:id` | 任务详情 |
| DELETE | `/api/deploys/:id` | 删除任务 |
| POST | `/api/deploys/:id/retry` | 重试任务 |
| GET | `/api/deploys/:id/console` | Jenkins 控制台输出 |
| GET | `/api/deploys/:id/stages` | 阶段历史 |

### 仓库管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/repos` | 仓库列表 |
| POST | `/api/repos/import` | 导入仓库 |
| GET | `/api/repos/:id/branches` | 分支列表 |
| GET | `/api/repos/:id/tags` | 标签列表 |

### 模板管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/build-templates` | 构建模板列表 |
| POST | `/api/build-templates` | 创建构建模板 |
| GET | `/api/deploy-templates` | 部署模板列表 |
| POST | `/api/deploy-templates` | 创建部署模板 |

### 集群与服务器

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/k8s/clusters` | K8s 集群列表 |
| POST | `/api/k8s/clusters` | 添加集群 |
| GET | `/api/k8s/clusters/:id/namespaces` | 命名空间列表 |
| GET | `/api/k8s/clusters/:id/pods` | Pod 列表 |
| GET | `/api/servers` | 服务器列表 |
| POST | `/api/servers` | 添加服务器 |

### 平台配置

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/git-platforms` | Git 平台列表 |
| POST | `/api/git-platforms` | 添加 Git 平台 |
| GET | `/api/jenkins-platforms` | Jenkins 平台列表 |
| POST | `/api/jenkins-platforms` | 添加 Jenkins 平台 |
| GET | `/api/image-repositories` | 镜像仓库列表 |
| POST | `/api/image-repositories` | 添加镜像仓库 |

### 其他

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/environments` | 环境列表 |
| GET | `/api/languages` | 语言列表 |
| GET | `/api/cloud-providers` | 云厂商列表 |
| GET | `/health` | 健康检查 |
| GET | `/docs` | Swagger UI |

---

## ☸️ 对接 Kubernetes

### 创建 ServiceAccount

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

### 获取 Token

```bash
# Kubernetes 1.24+
kubectl create token zebra-sa --duration=87600h

# Kubernetes 1.24 以下
SECRET_NAME=$(kubectl get serviceaccount zebra-sa -o jsonpath='{.secrets[0].name}')
kubectl get secret $SECRET_NAME -o jsonpath='{.data.token}' | base64 -d
```

### 获取 CA 证书

```bash
kubectl get secret $SECRET_NAME -o jsonpath='{.data.ca\.crt}'
```

---

## 🔧 开发指南

### 本地开发

```bash
# 安装依赖
go mod tidy

# 运行测试
go test ./...

# 代码格式化
go fmt ./...

# 静态检查
go vet ./...
```

### 生成 Swagger 文档

```bash
# 安装 swag
go install github.com/swaggo/swag/cmd/swag@latest

# 生成文档
swag init -g main.go
```

### 添加新的 API

1. 在 `internal/model/` 定义数据模型
2. 在 `internal/handler/` 创建 Repository
3. 在 `internal/service/` 创建 Service
4. 在 `internal/api/` 注册路由
5. 添加 Swagger 注释并执行 `swag init`

### 代码规范

- 遵循 [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- 使用 `gofmt` 格式化代码
- 新增 API 必须添加 Swagger 注释
- 错误处理使用 `log.L().Error()` 记录

---

## 📊 监控与运维

### 日志位置

| 日志文件 | 说明 |
|---------|------|
| `logs/app.log` | 应用日志 |
| `logs/error.log` | 错误日志 |

### 健康检查

```bash
# HTTP 健康检查
curl http://localhost:4123/health

# 检查数据库连接
curl http://localhost:4123/health/db
```

### 任务队列监控

通过 Asynq 监控工具查看任务状态：

```bash
# 安装 asynqmon
go install github.com/hibiken/asynq/tools/asynqmon@latest

# 启动监控面板
asynqmon --redis-addr=127.0.0.1:6379
```

---

## ❓ 常见问题

<details>
<summary><b>Q: 如何在不使用 Nacos 的情况下运行？</b></summary>

不设置 `NACOS_SERVER_ADDR` 环境变量，服务将自动跳过 Nacos 集成，仅使用本地配置文件和环境变量。

</details>

<details>
<summary><b>Q: 部署任务失败如何排查？</b></summary>

1. 查看任务详情：`GET /api/deploys/:id`
2. 查看阶段历史：`GET /api/deploys/:id/stages`
3. 查看 Jenkins 控制台输出：`GET /api/deploys/:id/console`
4. 检查应用日志：`logs/app.log`

</details>

<details>
<summary><b>Q: 如何支持多个 Git 平台？</b></summary>

通过 `GitPlatform` 模型配置多个 Git 平台，每个平台独立管理 token 和 URL。创建应用时选择对应的 Git 平台即可。

</details>

<details>
<summary><b>Q: Jenkins 凭据如何管理？</b></summary>

支持两种模式：
- **auto_create**: 系统自动创建 Jenkins 凭据
- **manual_select**: 手动选择已存在的 Jenkins 凭据

凭据通过 Jenkins Groovy Script Console 自动注入。

</details>

<details>
<summary><b>Q: 如何添加新的镜像仓库类型？</b></summary>

1. 在 `internal/core/` 实现 `RegistryClient` 接口
2. 在 `registry_factory.go` 添加新类型的工厂方法
3. 在 `internal/model/imageRepoModel.go` 添加新的 `RegistryType`

</details>

---

## 🗺️ 路线图

- [ ] 支持 GitLab CI/CD 直接触发
- [ ] 支持 GitHub Actions 集成
- [ ] 部署回滚功能
- [ ] 多环境配置对比
- [ ] 部署审批流程
- [ ] Prometheus 指标导出
- [ ] Grafana 监控面板

---

## 🤝 贡献指南

欢迎提交 Pull Request！

### 贡献流程

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 提交 Pull Request

### 代码要求

- 代码通过 `go fmt` 格式化
- 新增 API 已更新 Swagger 注释
- 关键业务逻辑有单元测试覆盖

---

## 📄 许可证

本项目采用 [MIT](LICENSE) 许可证。

---

## 📬 联系方式

- 📧 Email: iamnumachen@gmail.com
- 🐛 Issues: [GitHub Issues](https://github.com/ZebraOps/ZebraCICD/issues)

---

<div align="center">
  <sub>Built with ❤️ by ZebraOps Team</sub>
</div>
