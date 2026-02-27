<div align="center">
  <h1>Zebra-CICD</h1>
  <span><a href="./README.md">Chinese</a> | English</span>
</div>

## 📖 Project Introduction

Zebra-CICD is a Continuous Integration and Continuous Deployment (CI/CD) platform developed in Go language, designed to simplify application building, testing, and deployment processes. The project adopts a modern technology stack including Gin framework, GORM ORM, PostgreSQL database, and Kubernetes cluster management.

## ✨ Key Features

- **Metadata Management**: Unified storage of project configuration metadata
- **Task Tracking**: Track and record deployment task status
- **Multi-cloud Integration**: Built-in GitLab / Harbor / Jenkins / Kubernetes client components
- **Asynchronous Tasks**: Simulate image building and deployment processes with task queue support
- **Structured Logging**: Global structured logging with Zap for easy troubleshooting
- **API Documentation**: Auto-generated Swagger API documentation for development collaboration

## 🛠️ Technology Stack

- Backend Framework: Gin
- Database: PostgreSQL + GORM
- Logging System: Zap + Lumberjack
- Configuration Management: Viper
- External Integrations: GitLab, Harbor, Jenkins, Kubernetes
- API Documentation: Swagger UI

## 🌳 Directory Structure

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

## ⚡ Quick Start

1. **Prepare PostgreSQL database** and set the following environment variables (in `.env` file or shell environment):

| Environment Variable | Description         | Example                                                    |
| -------------------- | ------------------- | ---------------------------------------------------------- |
| ZEBRA_DATABASE_URL   | Database connection | postgres://user:pass@localhost:5432/dbname?sslmode=disable |
| ZEBRA_GITLAB_TOKEN   | GitLab Token        | your_token                                                 |
| ZEBRA_GITLAB_URL     | GitLab URL          | https://gitlab.com                                         |
| ZEBRA_HARBOR_URL     | Harbor URL          | https://harbor.example.com                                 |
| ZEBRA_PORT           | Running port        | 9527                                                       |

2. **Download dependencies & run**:

```sh
  go mod tidy
  go run main.go
```

3. **Open API documentation (Swagger UI)**:

- Access: http://127.0.0.1:9527/docs
- If local access fails, please check port and firewall settings

4. **Common development dependencies (first time only)**:

```sh
  go get github.com/gin-contrib/cors
  go get github.com/swaggo/gin-swagger
  go get github.com/swaggo/files
  swag init -g main.go
```

## ☸️ Kubernetes Cluster Integration

- For initial deployment, please refer to the following to create ServiceAccount, ClusterRole and binding (requires admin privileges):

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

- Get ServiceAccount Token:

```sh
  SECRET_NAME=$(kubectl get serviceaccount zebra-sa -o jsonpath='{.secrets[0].name}')
kubectl get secret $SECRET_NAME -o jsonpath='{.data.token}' | base64 -d
```

## 🤝 Contribution Guidelines

Welcome developers to contribute code and suggestions. Before submitting a Pull Request, please ensure:

- Code has been formatted with `go fmt`
- All new features have good comments
- Passed basic unit tests

If you have any issues, please submit them in [Issues](github.com/ZebraOps/ZebraCICD/issues).

## 💬 Contact Information

- Email: iamnumachen@gmail.com
- GitHub Issue: [Submit Issue](github.com/ZebraOps/ZebraCICD/issues/new)

## 📄 License

[MIT](LICENSE)