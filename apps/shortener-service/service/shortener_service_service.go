package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pingxin403/cuckoo/apps/shortener-service/cache"
	"github.com/pingxin403/cuckoo/apps/shortener-service/gen/shortener_servicepb"
	"github.com/pingxin403/cuckoo/apps/shortener-service/idgen"
	"github.com/pingxin403/cuckoo/apps/shortener-service/logger"
	"github.com/pingxin403/cuckoo/apps/shortener-service/metrics"
	"github.com/pingxin403/cuckoo/apps/shortener-service/storage"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ShortenerServiceImpl implements the ShortenerService gRPC service
// Requirements: 1.4, 1.5, 2.1, 4.3, 9.3
type ShortenerServiceImpl struct {
	shortener_servicepb.UnimplementedShortenerServiceServer
	storage      storage.Storage
	idGen        idgen.IDGenerator
	validator    *URLValidator
	cacheManager *cache.CacheManager
	baseURL      string
}

// NewShortenerServiceImpl creates a new ShortenerServiceImpl
func NewShortenerServiceImpl(
	storage storage.Storage,
	idGen idgen.IDGenerator,
	validator *URLValidator,
	cacheManager *cache.CacheManager,
) *ShortenerServiceImpl {
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "https://ex.co"
	}

	return &ShortenerServiceImpl{
		storage:      storage,
		idGen:        idGen,
		validator:    validator,
		cacheManager: cacheManager,
		baseURL:      baseURL,
	}
}

// CreateShortLink creates a new short link from a long URL
// Requirements: 1.4, 1.5, 2.1, 4.3, 9.3
func (s *ShortenerServiceImpl) CreateShortLink(
	ctx context.Context,
	req *shortener_servicepb.CreateShortLinkRequest,
) (*shortener_servicepb.CreateShortLinkResponse, error) {
	// Validate input URL
	sanitizedURL, err := s.validator.ValidateAndSanitize(req.LongUrl)
	if err != nil {
		if errors.Is(err, ErrURLTooLong) {
			return nil, status.Errorf(codes.InvalidArgument, "URL too long: %v", err)
		}
		if errors.Is(err, ErrInvalidProtocol) {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid protocol: %v", err)
		}
		if errors.Is(err, ErrMaliciousPattern) {
			return nil, status.Errorf(codes.InvalidArgument, "Malicious pattern detected: %v", err)
		}
		return nil, status.Errorf(codes.InvalidArgument, "Invalid URL: %v", err)
	}

	// Generate or validate custom code
	var shortCode string
	if req.CustomCode != "" {
		// Validate custom code
		if err := s.idGen.ValidateCustomCode(ctx, req.CustomCode); err != nil {
			if errors.Is(err, idgen.ErrCustomCodeUnavailable) {
				return nil, status.Errorf(codes.AlreadyExists, "Custom code already in use: %s", req.CustomCode)
			}
			return nil, status.Errorf(codes.InvalidArgument, "Invalid custom code: %v", err)
		}
		shortCode = req.CustomCode
	} else {
		// Generate new code
		shortCode, err = s.idGen.Generate(ctx)
		if err != nil {
			if errors.Is(err, idgen.ErrMaxRetriesExceeded) {
				return nil, status.Errorf(codes.Internal, "Failed to generate unique code after retries")
			}
			return nil, status.Errorf(codes.Internal, "Failed to generate code: %v", err)
		}
	}

	// Extract creator IP from context
	creatorIP := extractIPFromContext(ctx)

	// Create URL mapping
	now := time.Now()
	mapping := &storage.URLMapping{
		ShortCode: shortCode,
		LongURL:   sanitizedURL,
		CreatedAt: now,
		CreatorIP: creatorIP,
	}

	// Handle expiration time
	if req.ExpiresAt != nil {
		expiresAt := req.ExpiresAt.AsTime()
		mapping.ExpiresAt = &expiresAt
	}

	// Write to MySQL (synchronous - wait for confirmation)
	// Requirements: 2.1, 13.2
	if err := s.storage.Create(ctx, mapping); err != nil {
		metrics.ErrorsTotal.WithLabelValues("storage_create").Inc()
		if strings.Contains(err.Error(), "Duplicate entry") {
			return nil, status.Errorf(codes.AlreadyExists, "Short code already exists: %s", shortCode)
		}
		return nil, status.Errorf(codes.Internal, "Failed to create mapping: %v", err)
	}

	// Audit log: Log creation request with source IP
	// Requirements: 14.5
	logger.Log.Info("Short link created",
		zap.String("short_code", shortCode),
		zap.String("long_url", sanitizedURL),
		zap.String("creator_ip", creatorIP),
		zap.Time("created_at", now),
	)

	// Preheat cache (Redis) - best effort, don't fail if cache write fails
	// Requirements: 4.3
	if s.cacheManager != nil {
		_ = s.cacheManager.Set(ctx, shortCode, sanitizedURL, now)
	}

	// Record metrics
	metrics.LinksCreated.Inc()
	metrics.RequestsTotal.WithLabelValues("CreateShortLink", "success").Inc()

	// Build response
	response := &shortener_servicepb.CreateShortLinkResponse{
		ShortUrl:  fmt.Sprintf("%s/%s", s.baseURL, shortCode),
		ShortCode: shortCode,
		CreatedAt: timestamppb.New(now),
	}

	if mapping.ExpiresAt != nil {
		response.ExpiresAt = timestamppb.New(*mapping.ExpiresAt)
	}

	return response, nil
}

// GetLinkInfo retrieves metadata for a short link
// Requirements: 9.4
func (s *ShortenerServiceImpl) GetLinkInfo(
	ctx context.Context,
	req *shortener_servicepb.GetLinkInfoRequest,
) (*shortener_servicepb.GetLinkInfoResponse, error) {
	// Validate input
	if req.ShortCode == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Short code cannot be empty")
	}

	// Get mapping from storage (bypass cache for accurate metadata)
	mapping, err := s.storage.Get(ctx, req.ShortCode)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Errorf(codes.NotFound, "Short code not found: %s", req.ShortCode)
		}
		return nil, status.Errorf(codes.Internal, "Failed to get mapping: %v", err)
	}

	// Build response
	response := &shortener_servicepb.GetLinkInfoResponse{
		ShortCode:  mapping.ShortCode,
		LongUrl:    mapping.LongURL,
		CreatedAt:  timestamppb.New(mapping.CreatedAt),
		ClickCount: mapping.ClickCount,
		IsExpired:  mapping.ExpiresAt != nil && time.Now().After(*mapping.ExpiresAt),
	}

	if mapping.ExpiresAt != nil {
		response.ExpiresAt = timestamppb.New(*mapping.ExpiresAt)
	}

	return response, nil
}

// DeleteShortLink removes a short link (soft delete)
// Requirements: 4.6
func (s *ShortenerServiceImpl) DeleteShortLink(
	ctx context.Context,
	req *shortener_servicepb.DeleteShortLinkRequest,
) (*shortener_servicepb.DeleteShortLinkResponse, error) {
	// Validate input
	if req.ShortCode == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Short code cannot be empty")
	}

	// Soft delete in MySQL
	if err := s.storage.Delete(ctx, req.ShortCode); err != nil {
		metrics.ErrorsTotal.WithLabelValues("storage_delete").Inc()
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Errorf(codes.NotFound, "Short code not found: %s", req.ShortCode)
		}
		return nil, status.Errorf(codes.Internal, "Failed to delete mapping: %v", err)
	}

	// Invalidate all cache layers
	// Requirements: 4.6
	if s.cacheManager != nil {
		_ = s.cacheManager.Delete(ctx, req.ShortCode)
	}

	// Record metrics
	metrics.LinksDeleted.Inc()
	metrics.RequestsTotal.WithLabelValues("DeleteShortLink", "success").Inc()

	return &shortener_servicepb.DeleteShortLinkResponse{
		Success: true,
	}, nil
}

// extractIPFromContext extracts the client IP address from the gRPC context
func extractIPFromContext(ctx context.Context) string {
	// Try to get IP from peer info
	if p, ok := peer.FromContext(ctx); ok {
		return p.Addr.String()
	}

	// Try to get IP from metadata (X-Forwarded-For header)
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if xff := md.Get("x-forwarded-for"); len(xff) > 0 {
			// X-Forwarded-For can contain multiple IPs, take the first one
			ips := strings.Split(xff[0], ",")
			if len(ips) > 0 {
				return strings.TrimSpace(ips[0])
			}
		}
	}

	return "unknown"
}
