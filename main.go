package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/ZebraOps/ZebraCICD/config"
	"github.com/ZebraOps/ZebraCICD/internal/api"
	"github.com/ZebraOps/ZebraCICD/internal/core"
	"github.com/ZebraOps/ZebraCICD/internal/handler"
	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/service"
	"github.com/ZebraOps/ZebraCICD/internal/worker"
	"github.com/ZebraOps/ZebraCICD/pkg/log"
	"github.com/ZebraOps/ZebraCICD/pkg/middleware"
	nacosClient "github.com/ZebraOps/ZebraCICD/pkg/nacos"
	"github.com/ZebraOps/ZebraCICD/pkg/queue"
	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// @title Zebra-CICD API
// @version 0.1.0
// @description Minimal OpenAPI spec for Zebra-CICD endpoints
// @host localhost:4123
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
	
	logger := log.L()
	logger.Info("========================================")
	logger.Info("ZebraCICD 正在启动...")
	logger.Info("========================================")
	
	// --- 初始化 Nacos 客户端（可选） ---
	var nacos *nacosClient.Client
	var nacosLoader *nacosClient.CICDConfigLoader
	
	if cfg.NacosServerAddr != "" {
		logger.Info("检测到 Nacos 配置，开始初始化 Nacos 客户端",
			zap.String("server", cfg.NacosServerAddr),
			zap.String("namespace", cfg.NacosNamespace),
		)
		
		nc, err := nacosClient.NewClient(nacosClient.Config{
			ServerAddr: cfg.NacosServerAddr,
			Namespace:  cfg.NacosNamespace,
			Username:   cfg.NacosUsername,
			Password:   cfg.NacosPassword,
			Group:      cfg.NacosGroup,
			LogLevel:   cfg.Logging.Level,
		}, logger)
		
		if err != nil {
			logger.Error("Nacos 客户端初始化失败，将使用本地配置", zap.Error(err))
		} else {
			nacos = nc
			nacosLoader = nacosClient.NewCICDConfigLoader(nc, logger)
			
			// 从 Nacos 加载敏感配置
			logger.Info("正在从 Nacos 加载配置...")
			
			configMap := map[string]string{
				"database_url":      cfg.DatabaseURL,
				"gitlab_token":      cfg.GitLabToken,
				"gitlab_url":        cfg.GitLabURL,
				"jenkins_url":       cfg.JenkinsURL,
				"jenkins_password":  cfg.JenkinsPass,
				"harbor_url":        cfg.HarborURL,
				"redis_addr":        cfg.RedisAddr,
				"redis_password":    cfg.RedisPassword,
			}
			
			nacosLoader.LoadAllConfigs(configMap)
			
			// 更新配置
			cfg.DatabaseURL = configMap["database_url"]
			cfg.GitLabToken = configMap["gitlab_token"]
			cfg.GitLabURL = configMap["gitlab_url"]
			cfg.JenkinsURL = configMap["jenkins_url"]
			cfg.JenkinsPass = configMap["jenkins_password"]
			cfg.HarborURL = configMap["harbor_url"]
			cfg.RedisAddr = configMap["redis_addr"]
			cfg.RedisPassword = configMap["redis_password"]
			
			logger.Info("✓ Nacos 配置加载完成")
		}
	} else {
		logger.Info("未配置 Nacos，使用本地配置")
	}
	
	logger.Info("当前配置",
		zap.String("port", cfg.Port),
		zap.String("gitlabURL", cfg.GitLabURL),
		zap.String("jenkinsURL", cfg.JenkinsURL),
		zap.String("harborURL", cfg.HarborURL),
	)

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
		&model.Application{}, // 添加新的模型
		&model.ApplicationDeployment{},
		&model.Language{},
		&model.GitPlatform{},
		&model.JenkinsPlatform{},
	); err != nil {
		log.S().Fatalf("auto migrate failed: %v", err)
	}

	// 初始化 Asynq 队列客户端
	queueClient := queue.NewClient(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	defer queueClient.Close()

	// Repositories and services
	gitlabClient := core.NewGitLabClient(cfg.GitLabURL, cfg.GitLabToken)
	deploySvc := service.NewDeployService(db, cfg, queueClient)
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

	// Setup Gin router
	r := gin.New()
	r.Use(gin.Recovery())
	// CORS 由上游 ZebraGateway 统一处理，本服务不再单独设置以避免响应头重复
	r.Use(middleware.RequestLogger(log.L()))
	r.Use(middleware.UserIdentity())

	// API routes
	api.RegisterDeployRoutes(r, deploySvc)
	api.RegisterRepoRoutes(r, repoSvc)
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

	// 应用服务
	appRepo := handler.NewApplicationRepository(db)
	deployRepo := handler.NewApplicationDeploymentRepository(db)
	appSvc := service.NewApplicationService(appRepo, deployRepo, db)

	// 开发语言
	languageRepo := handler.NewLanguageRepository(db)
	languageSvc := service.NewLanguageService(languageRepo)

	// Git平台配置
	gitPlatformRepo := handler.NewGitPlatformRepository(db)
	gitPlatformSvc := service.NewGitPlatformService(gitPlatformRepo)

	// Jenkins平台配置
	jenkinsPlatformRepo := handler.NewJenkinsPlatformRepository(db)
	jenkinsPlatformSvc := service.NewJenkinsPlatformService(jenkinsPlatformRepo)

	// 注册路由
	api.RegisterK8SRoutes(r, k8sSvc)
	api.RegisterServerRoutes(r, serverSvc)
	api.RegisterContainerRoutes(r, serverSvc)
	api.RegisterEnvRoutes(r, envSvc)
	api.RegisterCloudProviderRoutes(r, cloudProviderSvc)
	api.RegisterDeploymentTemplateRoutes(r, deploymentTemplateSvc)
	api.RegisterImageRepositoryRoutes(r, imageRepoSvc)
	api.RegisterHealthRoutes(r, db)
	api.RegisterApplicationRoutes(r, appSvc)
	api.RegisterLanguageRoutes(r, languageSvc)
	api.RegisterGitPlatformRoutes(r, gitPlatformSvc)
	api.RegisterJenkinsPlatformRoutes(r, jenkinsPlatformSvc)
	api.RegisterDocsRoutes(r)

	// --- 启动 Asynq worker server ---
	asynqSrv := queue.NewServer(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB, cfg.WorkerConcurrency)
	deployWorker := worker.NewDeployWorker(deploySvc)
	mux := asynq.NewServeMux()
	mux.HandleFunc(queue.TypeDeployTask, deployWorker.HandleDeployTask)
	if err := asynqSrv.Start(mux); err != nil {
		log.S().Fatalf("failed to start asynq server: %v", err)
	}
	logger.Info("✓ Asynq worker 启动成功", zap.Int("concurrency", cfg.WorkerConcurrency))

	// --- 注册服务到 Nacos ---
	if nacos != nil {
		serviceIP := getLocalIP()
		servicePort := getPortNumber(cfg.Port)
		
		err := nacos.RegisterInstance("zebra-cicd", serviceIP, uint64(servicePort), map[string]string{
			"version":     "0.1.0",
			"endpoints":   "/api,/health",
			"description": "ZebraCICD 持续集成部署服务",
		})
		
		if err != nil {
			logger.Error("服务注册失败", zap.Error(err))
		} else {
			logger.Info("✓ 服务注册成功",
				zap.String("service", "zebra-cicd"),
				zap.String("ip", serviceIP),
				zap.Uint64("port", uint64(servicePort)),
			)
		}
	}

	port := cfg.Port
	if port == "" {
		port = "4123"
	}

	addr := fmt.Sprintf("0.0.0.0:%s", port)
	logger.Info("========================================")
	logger.Info("ZebraCICD 启动成功", zap.String("addr", addr))
	logger.Info("========================================")
	
	// --- 启动服务器，支持优雅关闭 ---
	srv := make(chan error, 1)
	go func() {
		srv <- r.Run(addr)
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-srv:
		logger.Fatal("服务器启动失败", zap.Error(err))
	case sig := <-quit:
		logger.Info("收到退出信号，开始优雅关闭...", zap.String("signal", sig.String()))
		
		// 注销 Nacos 服务
		if nacos != nil {
			serviceIP := getLocalIP()
			servicePort := getPortNumber(cfg.Port)
			
			err := nacos.DeregisterInstance("zebra-cicd", serviceIP, uint64(servicePort))
			if err != nil {
				logger.Error("服务注销失败", zap.Error(err))
			} else {
				logger.Info("✓ 服务注销成功")
			}
		}

		// 关闭 Asynq worker
		asynqSrv.Shutdown()
		logger.Info("✓ Asynq worker 已关闭")

		logger.Info("ZebraCICD 已关闭")
	}
}

// getLocalIP 获取本机 IP 地址
func getLocalIP() string {
	// 优先使用环境变量
	if ip := os.Getenv("SERVICE_IP"); ip != "" {
		return ip
	}
	
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

// getPortNumber 从端口字符串提取端口号
func getPortNumber(port string) int {
	var p int
	fmt.Sscanf(port, "%d", &p)
	if p == 0 {
		return 4123
	}
	return p
}
