package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/fekuna/omnipos-pkg/broker"
	"github.com/fekuna/omnipos-pkg/cache"
	"github.com/fekuna/omnipos-pkg/database/postgres"
	"github.com/fekuna/omnipos-pkg/i18n"
	"github.com/fekuna/omnipos-pkg/logger"
	"github.com/fekuna/omnipos-pkg/middleware"
	"github.com/fekuna/omnipos-pkg/search"
	"github.com/fekuna/omnipos-product-service/config"
	productv1 "github.com/fekuna/omnipos-proto/gen/go/omnipos/product/v1"

	catH "github.com/fekuna/omnipos-product-service/internal/category/handler"
	catRepoPkg "github.com/fekuna/omnipos-product-service/internal/category/repository"
	catUCPkg "github.com/fekuna/omnipos-product-service/internal/category/usecase"

	invH "github.com/fekuna/omnipos-product-service/internal/inventory/handler"
	invListenerPkg "github.com/fekuna/omnipos-product-service/internal/inventory/listener"
	invRepoPkg "github.com/fekuna/omnipos-product-service/internal/inventory/repository"
	invUCPkg "github.com/fekuna/omnipos-product-service/internal/inventory/usecase"

	prodH "github.com/fekuna/omnipos-product-service/internal/product/handler"
	prodRepoPkg "github.com/fekuna/omnipos-product-service/internal/product/repository"
	prodUCPkg "github.com/fekuna/omnipos-product-service/internal/product/usecase"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// 1. Load Configuration
	_ = godotenv.Load() // Load .env file if it exists
	cfg := config.LoadEnv()

	// 1.5 Initialize i18n
	i18n.Init()
	// In development, load from relative path. In docker, this might need adjustment.
	if err := i18n.Load("../omnipos-pkg/i18n/locales/active.en.json"); err != nil {
		log.Printf("Failed to load en locales: %v", err)
	}
	if err := i18n.Load("../omnipos-pkg/i18n/locales/active.id.json"); err != nil {
		log.Printf("Failed to load id locales: %v", err)
	}

	// 2. Initialize Logger
	logConfig := &logger.ZapLoggerConfig{
		IsDevelopment:     false,
		Encoding:          "json",
		Level:             "info",
		DisableCaller:     false,
		DisableStacktrace: false,
	}

	if cfg.Server.AppEnv == "development" {
		logConfig.IsDevelopment = true
		logConfig.Encoding = "console"
		logConfig.Level = "debug"
	}

	appLogger := logger.NewZapLogger(logConfig)
	defer appLogger.Sync()

	// 3. Connect to Database
	db, err := postgres.NewPostgres(&postgres.Config{
		Host:            cfg.Postgres.Host,
		Port:            cfg.Postgres.Port,
		User:            cfg.Postgres.User,
		Password:        cfg.Postgres.Password,
		DBName:          cfg.Postgres.DBName,
		SSLMode:         cfg.Postgres.SSLMode,
		MaxOpenConns:    cfg.Postgres.MaxOpenConns,
		MaxIdleConns:    cfg.Postgres.MaxIdleConns,
		ConnMaxLifetime: time.Duration(cfg.Postgres.ConnMaxLifetime) * time.Second,
		ConnMaxIdleTime: time.Duration(cfg.Postgres.ConnMaxIdleTime) * time.Second,
	})
	if err != nil {
		appLogger.Fatal("Could not connect to database", zap.Error(err))
	}
	defer db.Close()
	appLogger.Info("Connected to PostgreSQL database", zap.String("db_name", cfg.Postgres.DBName))

	// 4. Initialize Repositories
	catRepo := catRepoPkg.NewPGRepository(db)
	prodRepo := prodRepoPkg.NewPGRepository(db)
	invRepo := invRepoPkg.NewPGRepository(db)

	// 5. Initialize Redis
	redisClient, err := cache.NewRedisClient(&cache.Config{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err != nil {
		appLogger.Fatal("Could not connect to Redis", zap.Error(err))
	}
	defer redisClient.Close()
	appLogger.Info("Connected to Redis", zap.String("addr", cfg.Redis.Addr))

	// 5.5 Initialize Kafka Consumer
	kafkaConsumer := broker.NewConsumer(&broker.Config{
		Brokers: cfg.Kafka.Brokers,
		Topic:   cfg.Kafka.Topic,
		GroupID: cfg.Kafka.GroupID,
	})
	defer kafkaConsumer.Close()
	appLogger.Info("Connected to Kafka Consumer", zap.Strings("brokers", cfg.Kafka.Brokers), zap.String("topic", cfg.Kafka.Topic))

	// 5.8 Initialize Elasticsearch
	esClient, err := search.NewClient(&search.Config{
		Addresses: cfg.Elastic.Addresses,
		Username:  cfg.Elastic.Username,
		Password:  cfg.Elastic.Password,
	})
	if err != nil {
		appLogger.Warn("Could not connect to Elasticsearch (Search features might be limited)", zap.Error(err))
		// We don't fail fatal here to allow service to run even if ES is down (best practice for resilience)
		esClient = nil
	} else {
		appLogger.Info("Connected to Elasticsearch", zap.Strings("addresses", cfg.Elastic.Addresses))
	}

	// 6. Initialize UseCases
	catUC := catUCPkg.NewCategoryUseCase(catRepo, appLogger)
	prodUC := prodUCPkg.NewProductUseCase(prodRepo, redisClient, esClient, appLogger) // Injection
	invUC := invUCPkg.NewInventoryUseCase(invRepo, redisClient, appLogger)

	// 6.5 Initialize Listeners
	invListener := invListenerPkg.NewInventoryListener(kafkaConsumer, invUC, appLogger)

	// Start Listener
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go invListener.Start(ctx)

	// 6. Initialize Handlers
	catHandler := catH.NewCategoryHandler(catUC, appLogger)
	prodHandler := prodH.NewProductHandler(prodUC, appLogger)
	invHandler := invH.NewInventoryHandler(invUC, appLogger)

	// 7. Start gRPC Server
	port := cfg.Server.GRPCPort
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.ContextInterceptor()),
	)

	// Register Services
	productv1.RegisterCategoryServiceServer(grpcServer, catHandler)
	productv1.RegisterProductServiceServer(grpcServer, prodHandler)
	productv1.RegisterProductVariantServiceServer(grpcServer, prodHandler)
	productv1.RegisterInventoryServiceServer(grpcServer, invHandler)

	// Register Reflection
	reflection.Register(grpcServer)

	appLogger.Info("Starting gRPC server", zap.String("port", port))

	// Graceful Shutdown
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			appLogger.Fatal("failed to serve", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	appLogger.Info("Shutting down server...")
	grpcServer.GracefulStop()
	appLogger.Info("Server stopped")
}
