package main

import (
	"fmt"
	"os"

	"github.com/ZebraOps/ZebraCICD/config"
	"github.com/ZebraOps/ZebraCICD/internal/api"
	"github.com/ZebraOps/ZebraCICD/internal/core"
	"github.com/ZebraOps/ZebraCICD/internal/handler"
	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/service"
	"github.com/ZebraOps/ZebraCICD/pkg/log"
	"github.com/ZebraOps/ZebraCICD/pkg/middleware"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// @title Zebra-CICD API
// @version 0.1.0
// @description Minimal OpenAPI spec for Zebra-CICD endpoints
// @host localhost:9527
// @BasePath /
func main() {

	defer log.Sync()

	// Load config (env or config file)
	cfg := config.Load()

	// 初始化日志系统
	if err := log.InitWithConfig(cfg.Logging); err != nil {
		log.S().Error("Failed to init logger")
		os.Exit(1)
	}

	// Setup DB
	dsn := cfg.DatabaseURL
	if dsn == "" {
		log.S().Fatal("DATABASE_URL is required")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.S().Fatalf("failed to connect db: %v", err)
	}

	// Auto migrate models
	if err := db.AutoMigrate(
		&model.DeployTask{},
		&model.Repo{},
		&model.BuildTemplate{},
		&model.TemplateHistory{},
		&model.K8SCluster{},
		&model.Server{},
		&model.Environment{},
		&model.CloudProvider{},
		&model.DeploymentTemplate{},
		&model.DeploymentTemplateHistory{},
		&model.ImageRepository{},
	); err != nil {
		log.S().Fatalf("auto migrate failed: %v", err)
	}

	// Repositories and services
	gitlabClient := core.NewGitLabClient(cfg.GitLabURL, cfg.GitLabToken)
	deploySvc := service.NewDeployService(db, cfg)
	repoRepo := handler.NewRepoRepository(db)
	repoSvc := service.NewRepoService(repoRepo, gitlabClient, cfg.GitLabURL)

	// 模板相关的 Repository 和 Service
	buildTemplateRepo := handler.NewBuildTemplateRepository(db)
	templateHistoryRepo := handler.NewTemplateHistoryRepository(db)
	buildTemplateSvc := service.NewBuildTemplateService(buildTemplateRepo, templateHistoryRepo)

	// K8s 集群相关的 Repository 和 Service
	k8sClusterRepo := handler.NewK8SClusterRepository(db)
	k8sSvc := service.NewK8SService(k8sClusterRepo)

	// 服务器相关的 Repository 和 Service
	serverRepo := handler.NewServerRepository(db)
	serverSvc := service.NewServerService(serverRepo)

	// 镜像仓库
	imageRepoRepo := handler.NewImageRepositoryRepository(db)
	imageRepoSvc := service.NewImageRepositoryService(imageRepoRepo)

	// Start background worker that picks up pending tasks every interval (service starts its own goroutines)
	deploySvc.StartWorker()

	// Setup Gin router
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(cors.Default())                    // 添加CORS支持
	r.Use(middleware.RequestLogger(log.L())) // 添加请求日志中间件

	// API routes
	api.RegisterDeployRoutes(r, deploySvc)
	api.RegisterRepoRoutes(r, repoSvc, buildTemplateSvc)
	api.RegisterTemplateRoutes(r, buildTemplateSvc)

	// 环境相关
	envRepo := handler.NewEnvRepository(db)
	envSvc := service.NewEnvService(envRepo)

	// 云厂商
	cloudProviderRepo := handler.NewCloudProviderRepository(db)
	cloudProviderSvc := service.NewCloudProviderService(cloudProviderRepo)

	// 部署模板
	deploymentTemplateRepo := handler.NewDeploymentTemplateRepository(db)
	deploymentTemplateHistoryRepo := handler.NewDeploymentTemplateHistoryRepository(db)
	deploymentTemplateSvc := service.NewDeploymentTemplateService(deploymentTemplateRepo, deploymentTemplateHistoryRepo)

	// 注册 K8s、服务器、容器相关路由
	api.RegisterK8SRoutes(r, k8sSvc)
	api.RegisterServerRoutes(r, serverSvc)
	api.RegisterContainerRoutes(r, serverSvc)
	api.RegisterEnvRoutes(r, envSvc)
	api.RegisterCloudProviderRoutes(r, cloudProviderSvc)
	api.RegisterDeploymentTemplateRoutes(r, deploymentTemplateSvc)
	api.RegisterImageRepositoryRoutes(r, imageRepoSvc)
	api.RegisterHealthRoutes(r, db)

	api.RegisterDocsRoutes(r)

	port := cfg.Port
	if port == "" {
		port = "9527"
	}

	addr := fmt.Sprintf("0.0.0.0:%s", port)
	log.S().Infof("starting server on %s", addr)
	if err := r.Run(addr); err != nil {
		log.S().Fatalf("failed to run server: %v", err)
	}
}
