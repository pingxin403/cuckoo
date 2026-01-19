package service

import (
	"context"

	"github.com/pingxin403/cuckoo/apps/shortener-service/gen/shortener_servicepb"
	"github.com/pingxin403/cuckoo/apps/shortener-service/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UshortenerUserviceServiceServer implements the UshortenerUserviceService gRPC service
type UshortenerUserviceServiceServer struct {
	shortener_servicepb.UnimplementedUshortenerUserviceServiceServer
	store storage.YourStore
}

// NewUshortenerUserviceServiceServer creates a new UshortenerUserviceServiceServer
func NewUshortenerUserviceServiceServer(store storage.YourStore) *UshortenerUserviceServiceServer {
	return &UshortenerUserviceServiceServer{
		store: store,
	}
}

// TODO: Implement your RPC methods here
//
// Example:
//
// func (s *UshortenerUserviceServiceServer) YourMethod(ctx context.Context, req *shortener_servicepb.YourRequest) (*shortener_servicepb.YourResponse, error) {
//     // Validate input
//     if req.Field == "" {
//         return nil, status.Error(codes.InvalidArgument, "field is required")
//     }
//
//     // Your business logic here
//
//     return &shortener_servicepb.YourResponse{
//         Result: "Your result",
//     }, nil
// }
