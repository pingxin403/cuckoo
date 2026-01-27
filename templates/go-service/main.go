package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pingxin403/cuckoo/api/gen/go/{{PROTO_PACKAGE}}"
	"{{MODULE_PATH}}/config"
	"{{MODULE_PATH}}/service"
	"{{MODULE_PATH}}/storage"
	"github.com/pingxin403/cuckoo/libs/observability"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// 1. 加载配置
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// 2. 初始化可观测性
	obs, err := observability.New(observability.Config{
		ServiceName:    cfg.Observability.ServiceName,
		ServiceVersion: cfg.Observability.ServiceVersion,
		Environment:    cfg.Observability.Environment,
		EnableMetrics:  cfg.Observability.EnableMetrics,
		MetricsPort:    cfg.Observability.MetricsPort,
		LogLevel:       cfg.Observability.LogLevel,
		LogFormat:      cfg.Observability.LogFormat,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize observability: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := obs.Shutdown(shutdownCtx); err != nil {
			fmt.Fprintf(os.Stderr, "Observability shutdown error: %v\n", err)
		}
	}()

	ctx := context.Background()
	obs.Logger().Info(ctx, "Starting {{SERVICE_NAME}}",
		"service", cfg.Observability.ServiceName,
		"version", cfg.Observability.ServiceVersion,
		"port", cfg.Server.Port,
	)

	// 3. 创建 TCP 监听器
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		obs.Logger().Error(ctx, "Failed to listen", "port", cfg.Server.Port, "error", err)
		os.Exit(1)
	}

	// 4. 初始化存储
	store := storage.NewMemoryStore()
	obs.Logger().Info(ctx, "Initialized in-memory store")

	// 5. 创建服务实例
	svc := service.New{{ServiceName}}ServiceServer(store, obs)
	obs.Logger().Info(ctx, "Initialized {{SERVICE_NAME}} service")

	// 6. 创建 gRPC 服务器
	grpcServer := grpc.NewServer()

	// 7. 注册服务
	{{PROTO_PACKAGE}}.Register{{ServiceName}}ServiceServer(grpcServer, svc)

	// 注册反射服务用于调试（例如使用 grpcurl）
	reflection.Register(grpcServer)

	// 8. 设置优雅关闭
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// 9. 在 goroutine 中启动服务器
	go func() {
		obs.Logger().Info(ctx, "{{SERVICE_NAME}} listening", "port", cfg.Server.Port)
		obs.Logger().Info(ctx, "Service ready to accept requests")
		if err := grpcServer.Serve(lis); err != nil {
			obs.Logger().Error(ctx, "Failed to serve", "error", err)
			os.Exit(1)
		}
	}()

	// 10. 等待关闭信号
	sig := <-sigChan
	obs.Logger().Info(ctx, "Received shutdown signal", "signal", sig.String())

	// 11. 带超时的优雅关闭
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 停止接受新连接并等待现有连接完成
	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		obs.Logger().Info(shutdownCtx, "Server stopped gracefully")
	case <-shutdownCtx.Done():
		obs.Logger().Warn(shutdownCtx, "Shutdown timeout exceeded, forcing stop")
		grpcServer.Stop()
	}

	obs.Logger().Info(shutdownCtx, "{{SERVICE_NAME}} shutdown complete")
}
