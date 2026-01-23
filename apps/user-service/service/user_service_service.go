package service

import (
	"context"

	"github.com/pingxin403/cuckoo/apps/user-service/gen/user_servicepb"
	"github.com/pingxin403/cuckoo/apps/user-service/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UuserUserviceServiceServer implements the UuserUserviceService gRPC service
type UuserUserviceServiceServer struct {
	user_servicepb.UnimplementedUuserUserviceServiceServer
	store storage.YourStore
}

// NewUuserUserviceServiceServer creates a new UuserUserviceServiceServer
func NewUuserUserviceServiceServer(store storage.YourStore) *UuserUserviceServiceServer {
	return &UuserUserviceServiceServer{
		store: store,
	}
}

// TODO: Implement your RPC methods here
//
// Example:
//
// func (s *UuserUserviceServiceServer) YourMethod(ctx context.Context, req *user_servicepb.YourRequest) (*user_servicepb.YourResponse, error) {
//     // Validate input
//     if req.Field == "" {
//         return nil, status.Error(codes.InvalidArgument, "field is required")
//     }
//
//     // Your business logic here
//
//     return &user_servicepb.YourResponse{
//         Result: "Your result",
//     }, nil
// }
