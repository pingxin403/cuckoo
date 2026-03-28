package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	authpb "github.com/pingxin403/cuckoo/api/gen/go/authpb"
	im_gatewaypb "github.com/pingxin403/cuckoo/api/gen/go/im-gatewaypb"
	impb "github.com/pingxin403/cuckoo/api/gen/go/impb"
	"github.com/pingxin403/cuckoo/apps/im-gateway-service/config"
	"github.com/pingxin403/cuckoo/apps/im-gateway-service/metrics"
	"github.com/pingxin403/cuckoo/apps/im-gateway-service/service"
	"github.com/pingxin403/cuckoo/libs/observability"
	"github.com/redis/go-redis/v9"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type authClientAdapter struct {
	client authpb.AuthServiceClient
}

func (a *authClientAdapter) ValidateToken(ctx context.Context, token string) (*service.TokenClaims, error) {
	if a == nil || a.client == nil {
		return nil, errors.New("auth client is not initialized")
	}

	resp, err := a.client.ValidateToken(ctx, &authpb.ValidateTokenRequest{AccessToken: token})
	if err != nil {
		return nil, err
	}
	if !resp.GetValid() {
		if msg := resp.GetErrorMessage(); msg != "" {
			return nil, errors.New(msg)
		}
		return nil, errors.New("token validation failed")
	}

	claims := &service.TokenClaims{
		UserID:   resp.GetUserId(),
		DeviceID: resp.GetDeviceId(),
	}
	if exp := resp.GetExpiresAt(); exp != nil {
		claims.ExpiresAt = exp.AsTime().Unix()
	}

	return claims, nil
}

type imClientAdapter struct {
	client impb.IMServiceClient
}

type etcdRegistryClient struct {
	client *clientv3.Client
	ttl    time.Duration
	mu     sync.RWMutex
	leases map[string]int64
}

func (a *etcdRegistryClient) RegisterUser(ctx context.Context, userID, deviceID, gatewayNode string) error {
	if a == nil || a.client == nil {
		return errors.New("registry client is not initialized")
	}
	if userID == "" || deviceID == "" || gatewayNode == "" {
		return errors.New("invalid registry registration arguments")
	}

	leaseResp, err := a.client.Grant(ctx, int64(a.ttl.Seconds()))
	if err != nil {
		return err
	}
	key := "/registry/users/" + userID + "/" + deviceID
	value := gatewayNode + "|" + strconv.FormatInt(time.Now().Unix(), 10)
	if _, err := a.client.Put(ctx, key, value, clientv3.WithLease(leaseResp.ID)); err != nil {
		return err
	}

	if a.leases == nil {
		a.leases = make(map[string]int64)
	}
	a.mu.Lock()
	a.leases[userID+"_"+deviceID] = int64(leaseResp.ID)
	a.mu.Unlock()
	return nil
}

func (a *etcdRegistryClient) UnregisterUser(ctx context.Context, userID, deviceID string) error {
	if a == nil || a.client == nil {
		return errors.New("registry client is not initialized")
	}

	a.mu.Lock()
	delete(a.leases, userID+"_"+deviceID)
	a.mu.Unlock()

	_, err := a.client.Delete(ctx, "/registry/users/"+userID+"/"+deviceID)
	return err
}

func (a *etcdRegistryClient) RenewLease(ctx context.Context, userID, deviceID string) error {
	if a == nil || a.client == nil {
		return errors.New("registry client is not initialized")
	}

	a.mu.RLock()
	leaseID, ok := a.leases[userID+"_"+deviceID]
	a.mu.RUnlock()
	if !ok || leaseID <= 0 {
		return errors.New("registry lease not found")
	}

	_, err := a.client.KeepAliveOnce(ctx, clientv3.LeaseID(leaseID))
	return err
}

func (a *etcdRegistryClient) LookupUser(ctx context.Context, userID string) ([]service.GatewayLocation, error) {
	if a == nil || a.client == nil {
		return nil, errors.New("registry client is not initialized")
	}

	prefix := "/registry/users/" + userID + "/"
	resp, err := a.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	result := make([]service.GatewayLocation, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		key := string(kv.Key)
		deviceID := strings.TrimPrefix(key, prefix)
		parts := strings.SplitN(string(kv.Value), "|", 2)
		if len(parts) == 0 {
			continue
		}
		connectedAt := int64(0)
		if len(parts) == 2 {
			if parsed, parseErr := strconv.ParseInt(parts[1], 10, 64); parseErr == nil {
				connectedAt = parsed
			}
		}
		result = append(result, service.GatewayLocation{
			GatewayNode: parts[0],
			DeviceID:    deviceID,
			ConnectedAt: connectedAt,
		})
	}

	return result, nil
}

func (a *etcdRegistryClient) Watch(ctx context.Context, prefix string, callback func(clientv3.WatchResponse)) error {
	if a == nil || a.client == nil {
		return errors.New("registry client is not initialized")
	}
	if callback == nil {
		return errors.New("watch callback is required")
	}

	go func() {
		watchChan := a.client.Watch(ctx, prefix, clientv3.WithPrefix())
		for watchResp := range watchChan {
			callback(watchResp)
		}
	}()

	return nil
}

func (a *etcdRegistryClient) Close() error {
	if a == nil || a.client == nil {
		return nil
	}
	return a.client.Close()
}

func (a *imClientAdapter) RoutePrivateMessage(ctx context.Context, req *service.RoutePrivateMessageRequest) (*service.RoutePrivateMessageResponse, error) {
	if a == nil || a.client == nil {
		return nil, errors.New("im client is not initialized")
	}

	messageType := impb.MessageType_MESSAGE_TYPE_TEXT
	if req.MessageType == "" {
		messageType = impb.MessageType_MESSAGE_TYPE_UNSPECIFIED
	}

	resp, err := a.client.RoutePrivateMessage(ctx, &impb.RoutePrivateMessageRequest{
		MsgId:       req.MsgID,
		SenderId:    req.SenderID,
		RecipientId: req.RecipientID,
		Content:     req.Content,
		MessageType: messageType,
	})
	if err != nil {
		return nil, err
	}

	result := &service.RoutePrivateMessageResponse{
		SequenceNumber: resp.GetSequenceNumber(),
		DeliveryStatus: convertDeliveryStatus(resp.GetDeliveryStatus()),
		ErrorCode:      convertIMErrorCode(resp.GetErrorCode()),
		ErrorMessage:   resp.GetErrorMessage(),
	}
	if ts := resp.GetServerTimestamp(); ts != nil {
		result.ServerTimestamp = ts.AsTime().Unix()
	}

	return result, nil
}

func (a *imClientAdapter) RouteGroupMessage(ctx context.Context, req *service.RouteGroupMessageRequest) (*service.RouteGroupMessageResponse, error) {
	if a == nil || a.client == nil {
		return nil, errors.New("im client is not initialized")
	}

	messageType := impb.MessageType_MESSAGE_TYPE_TEXT
	if req.MessageType == "" {
		messageType = impb.MessageType_MESSAGE_TYPE_UNSPECIFIED
	}

	resp, err := a.client.RouteGroupMessage(ctx, &impb.RouteGroupMessageRequest{
		MsgId:       req.MsgID,
		SenderId:    req.SenderID,
		GroupId:     req.GroupID,
		Content:     req.Content,
		MessageType: messageType,
	})
	if err != nil {
		return nil, err
	}

	result := &service.RouteGroupMessageResponse{
		SequenceNumber:     resp.GetSequenceNumber(),
		OnlineMemberCount:  resp.GetOnlineMemberCount(),
		OfflineMemberCount: resp.GetOfflineMemberCount(),
		ErrorCode:          convertIMErrorCode(resp.GetErrorCode()),
		ErrorMessage:       resp.GetErrorMessage(),
	}
	if ts := resp.GetServerTimestamp(); ts != nil {
		result.ServerTimestamp = ts.AsTime().Unix()
	}

	return result, nil
}

func convertDeliveryStatus(status impb.DeliveryStatus) string {
	switch status {
	case impb.DeliveryStatus_DELIVERY_STATUS_PENDING:
		return "pending"
	case impb.DeliveryStatus_DELIVERY_STATUS_DELIVERED:
		return "delivered"
	case impb.DeliveryStatus_DELIVERY_STATUS_READ:
		return "read"
	case impb.DeliveryStatus_DELIVERY_STATUS_FAILED:
		return "failed"
	case impb.DeliveryStatus_DELIVERY_STATUS_OFFLINE:
		return "offline"
	default:
		return ""
	}
}

func convertIMErrorCode(code impb.IMErrorCode) string {
	if code == impb.IMErrorCode_IM_ERROR_CODE_UNSPECIFIED {
		return ""
	}
	return code.String()
}

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer func() { _ = redisClient.Close() }()

	// Test Redis connection
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis connection failed: %v", err)
	} else {
		log.Println("Connected to Redis")
	}

	authConn, err := grpc.NewClient(cfg.ServiceDiscovery.AuthServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect Auth service: %v", err)
	}
	defer func() { _ = authConn.Close() }()

	imConn, err := grpc.NewClient(cfg.ServiceDiscovery.IMServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect IM service: %v", err)
	}
	defer func() { _ = imConn.Close() }()

	dialTimeout := cfg.Etcd.DialTimeout
	if dialTimeout <= 0 {
		dialTimeout = 5 * time.Second
	}
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints:   cfg.Etcd.Endpoints,
		DialTimeout: dialTimeout,
	})
	if err != nil {
		log.Fatalf("Failed to initialize registry client: %v", err)
	}
	defer func() { _ = etcdClient.Close() }()

	authClient := &authClientAdapter{client: authpb.NewAuthServiceClient(authConn)}
	registryClient := &etcdRegistryClient{client: etcdClient, ttl: 90 * time.Second, leases: make(map[string]int64)}
	imClient := &imClientAdapter{client: impb.NewIMServiceClient(imConn)}

	// Initialize observability with OpenTelemetry metrics
	obs, err := observability.New(observability.Config{
		ServiceName:         cfg.Observability.ServiceName,
		ServiceVersion:      cfg.Observability.ServiceVersion,
		Environment:         cfg.Observability.Environment,
		EnableMetrics:       cfg.Observability.EnableMetrics,
		UseOTelMetrics:      true,                                     // Use OpenTelemetry metrics
		PrometheusEnabled:   true,                                     // Enable Prometheus exporter
		MetricsPort:         cfg.Observability.MetricsPort,            // Separate port for metrics
		OTLPMetricsEndpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"), // OTLP endpoint for metrics
		OTLPInsecure:        true,                                     // Use insecure connection for development
		EnableTracing:       false,
		LogLevel:            cfg.Observability.LogLevel,
		LogFormat:           cfg.Observability.LogFormat,
	})
	if err != nil {
		log.Fatalf("Failed to initialize observability: %v", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := obs.Shutdown(shutdownCtx); err != nil {
			log.Printf("Observability shutdown error: %v", err)
		}
	}()

	obs.Logger().Info(ctx, "Observability initialized",
		"service", cfg.Observability.ServiceName,
		"version", cfg.Observability.ServiceVersion,
		"metrics_port", cfg.Observability.MetricsPort,
		"otel_metrics", true,
	)

	// Create metrics instance with observability
	gatewayMetrics := metrics.NewMetrics(obs)

	// Create gateway service with default config
	gatewayConfig := service.DefaultGatewayConfig()
	gatewayConfig.AllowedOrigins = cfg.Security.AllowedOrigins
	gatewayConfig.AllowEmptyOrigin = cfg.Security.AllowEmptyOrigin
	gateway := service.NewGatewayService(
		authClient,
		registryClient,
		imClient,
		redisClient,
		gatewayConfig,
	)
	gateway.SetRemoteForwarder(service.NewGRPCRemoteForwarder(cfg.ServiceDiscovery.GatewayNodes))

	_ = gatewayMetrics

	kafkaConfig := service.KafkaConfig{
		Brokers:                 cfg.Kafka.Brokers,
		GroupID:                 cfg.Kafka.ConsumerGroup,
		Topic:                   cfg.Kafka.Topic,
		ReadReceiptTopic:        "read_receipt",
		ReadReceiptGroupID:      "im-gateway-read-receipt",
		MembershipChangeTopic:   "group_membership_change",
		MembershipChangeGroupID: "im-gateway-membership-change",
		MinBytes:                1024,
		MaxBytes:                10 * 1024 * 1024,
		CommitInterval:          time.Second,
		EnableReadReceipts:      true,
		EnableMembershipChange:  true,
	}
	if err := gateway.Start(kafkaConfig); err != nil {
		log.Fatalf("Failed to start gateway service: %v", err)
	}

	// Setup HTTP server with timeouts
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", gateway.HandleWebSocket)
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("OK"))
	})

	grpcServer := grpc.NewServer()
	im_gatewaypb.RegisterUimUgatewayUserviceServiceServer(grpcServer, newGatewayRPCServer(gateway))
	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.GRPCPort))
	if err != nil {
		log.Fatalf("Failed to listen on gRPC port: %v", err)
	}

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.HTTPPort),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		obs.Logger().Info(ctx, "Starting HTTP server",
			"port", cfg.Server.HTTPPort,
			"websocket_endpoint", "/ws",
			"health_endpoint", "/health",
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			obs.Logger().Error(ctx, "HTTP server error", "error", err)
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	go func() {
		obs.Logger().Info(ctx, "Starting gRPC server", "port", cfg.Server.GRPCPort)
		if err := grpcServer.Serve(grpcListener); err != nil {
			obs.Logger().Error(ctx, "gRPC server error", "error", err)
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	// Wait for shutdown signal
	sig := <-sigChan
	obs.Logger().Info(ctx, "Received shutdown signal", "signal", sig.String())

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown gateway service
	if err := gateway.Shutdown(shutdownCtx); err != nil {
		obs.Logger().Error(shutdownCtx, "Gateway shutdown error", "error", err)
	}

	// Shutdown metrics
	if err := gatewayMetrics.Shutdown(shutdownCtx); err != nil {
		obs.Logger().Error(shutdownCtx, "Metrics shutdown error", "error", err)
	}

	// Shutdown HTTP server
	if err := server.Shutdown(shutdownCtx); err != nil {
		obs.Logger().Error(shutdownCtx, "HTTP server shutdown error", "error", err)
	}

	grpcServer.GracefulStop()
	_ = grpcListener.Close()

	obs.Logger().Info(shutdownCtx, "Shutdown complete")
}
