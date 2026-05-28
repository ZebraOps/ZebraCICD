<div align="center">
  <h1>Zebra-CICD</h1>
  <span>中文 | <a href="./README.en.md">English</a></span>
</div>

## 📖 项目简介

Zebra-CICD 是一个基于 Go 语言开发的持续集成与持续部署（CI/CD）平台，旨在简化应用程序的构建、测试和部署流程。项目采用了现代化的技术栈，包括 Gin 框架、GORM ORM、PostgreSQL 数据库以及 Kubernetes 集群管理。

## ✨ 项目特性

- **元数据管理**：支持项目配置的元数据统一存储。
- **任务追踪**：跟踪和记录部署任务的状态。
- **多云集成**：内置 GitLab / Harbor / Jenkins / Kubernetes 客户端组件。
- **异步任务**：模拟镜像构建与部署过程，支持任务队列。
- **结构化日志**：用 Zap 记录全局结构化日志，便于问题跟踪。
- **接口文档**：自动生成 Swagger API 文档，便于开发联调。

## 🛠️ 技术栈

- 后端框架：Gin
- 数据库：PostgreSQL + GORM
- 日志系统：Zap + Lumberjack
- 配置管理：Viper
- 外部集成：GitLab、Harbor、Jenkins、Kubernetes
- API 文档：Swagger UI

## 🌳 目录结构

```text
zebra-cicd/
├── config/           # 配置文件管理
│   ├── config.go
│   └── configs.yaml
├── docs/             # API 文档
├── internal/         # 核心业务逻辑
│   ├── api/          # API 控制器
│   ├── core/         # 各外部系统客户端
│   ├── handler/      # 数据库CRUD
│   ├── model/        # 数据模型定义
│   ├── service/      # 业务编排/逻辑
│   └── types/        # 公共类型
├── pkg/              # 通用组件/工具包
│   ├── log/
│   ├── middleware/
│   ├── ssh/
│   └── timeutil/
├── scripts/          # 启动和构建脚本
├── main.go           # 应用入口
├── go.mod            # Go模块依赖
└── README.md
```

## ⚡ 快速开始

1. **准备数据库 PostgreSQL**，并设置以下环境变量（如 `.env` 文件或 shell 环境）：

| 环境变量           | 说明         | 示例                                                       |
| ------------------ | ------------ | ---------------------------------------------------------- |
| ZEBRA_DATABASE_URL | 数据库连接串 | postgres://user:pass@localhost:5432/dbname?sslmode=disable |
| ZEBRA_GITLAB_TOKEN | GitLab Token | your_token                                                 |
| ZEBRA_GITLAB_URL   | GitLab 地址  | https://gitlab.com                                         |
| ZEBRA_HARBOR_URL   | 镜像仓库地址 | registry.cn-shanghai.aliyuncs.com                         |
| ZEBRA_PORT         | 运行端口     | 4123                                                       |

2. **依赖下载&运行**：

```sh
go mod tidy
go run main.go
```

3. **打开接口文档（Swagger UI）**：

- 访问：http://127.0.0.1:4123/docs
- 如果本地无法访问，请检查端口和防火墙设置

4. **常见开发依赖（仅首次）**：

```sh
go get github.com/gin-contrib/cors
go get github.com/swaggo/gin-swagger
go get github.com/swaggo/files
swag init -g main.go
```

## ☸️ 对接 Kubernetes 集群

- 首次部署请参考如下创建 ServiceAccount、ClusterRole 及绑定（需 admin 权限）：

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
    resources:
      [
        "nodes",
        "pods",
        "services",
        "namespaces",
        "configmaps",
        "secrets",
        "events",
        "jobs",
        "cronjobs",
      ]
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

- 获取 ServiceAccount Token：

```sh
SECRET_NAME=$(kubectl get serviceaccount zebra-sa -o jsonpath='{.secrets[0].name}')
kubectl get secret $SECRET_NAME -o jsonpath='{.data.token}' | base64 -d
```

## 🤝 贡献指南

欢迎开发者贡献代码和建议，请提交 Pull Request 前确保：

- 代码已过 `go fmt` 格式化
- 所有新功能有良好注释
- 已通过基本单元测试

如有问题请在 [Issues](github.com/ZebraOps/ZebraCICD/issues) 提交。

## 💬 联系方式

- 邮箱：iamnumachen@gmail.com
- GitHub Issue: [提交问题](github.com/ZebraOps/ZebraCICD/issues/new)

## 📄 License

[MIT](LICENSE)
