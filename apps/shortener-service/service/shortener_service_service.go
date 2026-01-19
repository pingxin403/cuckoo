package service

import (
	"github.com/pingxin403/cuckoo/apps/shortener-service/gen/shortener_servicepb"
	"github.com/pingxin403/cuckoo/apps/shortener-service/storage"
)

// ShortenerServiceImpl implements the ShortenerService gRPC service
type ShortenerServiceImpl struct {
	shortener_servicepb.UnimplementedShortenerServiceServer
	storage storage.Storage
}

// NewShortenerServiceImpl creates a new ShortenerServiceImpl
func NewShortenerServiceImpl(storage storage.Storage) *ShortenerServiceImpl {
	return &ShortenerServiceImpl{
		storage: storage,
	}
}

// TODO: Implement RPC methods in task 11
// - CreateShortLink
// - GetLinkInfo
// - DeleteShortLink
