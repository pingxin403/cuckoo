package service

import (
	"context"
	"time"

	"github.com/pingxin403/cuckoo/libs/observability"
	"{{MODULE_PATH}}/gen/{{PROTO_PACKAGE}}"
	"{{MODULE_PATH}}/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// {{ServiceName}}ServiceServer implements the {{ServiceName}}Service gRPC service
type {{ServiceName}}ServiceServer struct {
	{{PROTO_PACKAGE}}.Unimplemented{{ServiceName}}ServiceServer
	store storage.YourStore
	obs   observability.Observability
}

// New{{ServiceName}}ServiceServer creates a new {{ServiceName}}ServiceServer
func New{{ServiceName}}ServiceServer(store storage.YourStore, obs observability.Observability) *{{ServiceName}}ServiceServer {
	return &{{ServiceName}}ServiceServer{
		store: store,
		obs:   obs,
	}
}

// recordMetrics records gRPC request metrics
func (s *{{ServiceName}}ServiceServer) recordMetrics(method string, statusCode string, duration time.Duration) {
	if s.obs == nil {
		return
	}
	labels := map[string]string{
		"method": method,
		"status": statusCode,
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
//     // Validate input
//     if req.Field == "" {
//         return nil, status.Error(codes.InvalidArgument, "field is required")
//     }
//
//     // Your business logic here
//
//     return &{{PROTO_PACKAGE}}.YourResponse{
//         Result: "Your result",
//     }, nil
// }
