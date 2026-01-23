package service

import (
	"context"

	"github.com/pingxin403/cuckoo/apps/im-service/gen/im_servicepb"
	"github.com/pingxin403/cuckoo/apps/im-service/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UimUserviceServiceServer implements the UimUserviceService gRPC service
type UimUserviceServiceServer struct {
	im_servicepb.UnimplementedUimUserviceServiceServer
	store storage.YourStore
}

// NewUimUserviceServiceServer creates a new UimUserviceServiceServer
func NewUimUserviceServiceServer(store storage.YourStore) *UimUserviceServiceServer {
	return &UimUserviceServiceServer{
		store: store,
	}
}

// TODO: Implement your RPC methods here
//
// Example:
//
// func (s *UimUserviceServiceServer) YourMethod(ctx context.Context, req *im_servicepb.YourRequest) (*im_servicepb.YourResponse, error) {
//     // Validate input
//     if req.Field == "" {
//         return nil, status.Error(codes.InvalidArgument, "field is required")
//     }
//
//     // Your business logic here
//
//     return &im_servicepb.YourResponse{
//         Result: "Your result",
//     }, nil
// }
