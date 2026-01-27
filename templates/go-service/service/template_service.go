package service

import (
	"context"
	"time"

	"github.com/pingxin403/cuckoo/api/gen/go/{{PROTO_PACKAGE}}"
	"github.com/pingxin403/cuckoo/libs/observability"
	"{{MODULE_PATH}}/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// {{ServiceName}}ServiceServer 实现 gRPC 服务
type {{ServiceName}}ServiceServer struct {
	{{PROTO_PACKAGE}}.Unimplemented{{ServiceName}}ServiceServer
	store storage.Store
	obs   observability.Observability
}

// New{{ServiceName}}ServiceServer 创建服务实例
func New{{ServiceName}}ServiceServer(store storage.Store, obs observability.Observability) *{{ServiceName}}ServiceServer {
	return &{{ServiceName}}ServiceServer{
		store: store,
		obs:   obs,
	}
}

// recordMetrics 记录请求指标
func (s *{{ServiceName}}ServiceServer) recordMetrics(method string, status string, duration time.Duration) {
	if s.obs == nil {
		return
	}
	labels := map[string]string{
		"method": method,
		"status": status,
	}
	s.obs.Metrics().IncrementCounter("{{SERVICE_NAME_SNAKE}}_grpc_requests_total", labels)
	s.obs.Metrics().RecordDuration("{{SERVICE_NAME_SNAKE}}_grpc_request_duration_seconds", duration, map[string]string{
		"method": method,
	})
}

// TODO: Implement your RPC methods here
//
// Example:
//
// func (s *{{ServiceName}}ServiceServer) YourMethod(ctx context.Context, req *{{PROTO_PACKAGE}}.YourRequest) (*{{PROTO_PACKAGE}}.YourResponse, error) {
//     startTime := time.Now()
//     defer func() {
//         s.recordMetrics("YourMethod", "success", time.Since(startTime))
//     }()
//
//     // Validate input
//     if req.Field == "" {
//         s.recordMetrics("YourMethod", "invalid_argument", time.Since(startTime))
//         return nil, status.Error(codes.InvalidArgument, "field is required")
//     }
//
//     // Your business logic here using s.store
//
//     return &{{PROTO_PACKAGE}}.YourResponse{
//         Result: "Your result",
//     }, nil
// }

// Ensure unused imports are referenced (remove these lines when implementing actual methods)
var (
	_ context.Context
	_ codes.Code
	_ = status.Error
)
