package service

import (
	"context"

	"{{MODULE_PATH}}/gen/{{PROTO_PACKAGE}}"
	"{{MODULE_PATH}}/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// {{ServiceName}}ServiceServer implements the {{ServiceName}}Service gRPC service
type {{ServiceName}}ServiceServer struct {
	{{PROTO_PACKAGE}}.Unimplemented{{ServiceName}}ServiceServer
	store storage.YourStore
}

// New{{ServiceName}}ServiceServer creates a new {{ServiceName}}ServiceServer
func New{{ServiceName}}ServiceServer(store storage.YourStore) *{{ServiceName}}ServiceServer {
	return &{{ServiceName}}ServiceServer{
		store: store,
	}
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
