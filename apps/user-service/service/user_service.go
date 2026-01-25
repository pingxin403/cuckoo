package service

import (
	"context"
	"fmt"
	"time"

	"github.com/pingxin403/cuckoo/apps/user-service/gen/userpb"
	"github.com/pingxin403/cuckoo/apps/user-service/storage"
	"github.com/pingxin403/cuckoo/libs/observability"
)

// UserServiceServer implements the UserService gRPC service
type UserServiceServer struct {
	userpb.UnimplementedUserServiceServer
	store storage.UserStore
	obs   observability.Observability
}

// NewUserServiceServer creates a new UserServiceServer
func NewUserServiceServer(store storage.UserStore, obs observability.Observability) *UserServiceServer {
	return &UserServiceServer{
		store: store,
		obs:   obs,
	}
}

// GetUser retrieves a single user's profile
func (s *UserServiceServer) GetUser(ctx context.Context, req *userpb.GetUserRequest) (*userpb.GetUserResponse, error) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		s.obs.Metrics().RecordDuration("user_grpc_request_duration_seconds", duration, map[string]string{
			"method": "GetUser",
		})
	}()

	// Record request count
	s.obs.Metrics().IncrementCounter("user_grpc_requests_total", map[string]string{
		"method": "GetUser",
	})

	if req.UserId == "" {
		s.obs.Metrics().IncrementCounter("user_operations_total", map[string]string{
			"operation": "get",
			"status":    "failure",
		})
		return &userpb.GetUserResponse{
			ErrorCode:    userpb.UserErrorCode_USER_ERROR_CODE_INVALID_REQUEST,
			ErrorMessage: "user_id is required",
		}, nil
	}

	dbStartTime := time.Now()
	user, err := s.store.GetUser(ctx, req.UserId)
	dbDuration := time.Since(dbStartTime)

	s.obs.Metrics().RecordDuration("user_db_operation_duration_seconds", dbDuration, map[string]string{
		"operation": "get",
	})

	if err != nil {
		// Check if user not found
		if err.Error() == fmt.Sprintf("user not found: %s", req.UserId) {
			s.obs.Metrics().IncrementCounter("user_operations_total", map[string]string{
				"operation": "get",
				"status":    "not_found",
			})
			s.obs.Metrics().IncrementCounter("user_db_operations_total", map[string]string{
				"operation": "get",
				"status":    "not_found",
			})
			return &userpb.GetUserResponse{
				ErrorCode:    userpb.UserErrorCode_USER_ERROR_CODE_USER_NOT_FOUND,
				ErrorMessage: fmt.Sprintf("User %s not found", req.UserId),
			}, nil
		}

		// Database error
		s.obs.Metrics().IncrementCounter("user_operations_total", map[string]string{
			"operation": "get",
			"status":    "failure",
		})
		s.obs.Metrics().IncrementCounter("user_db_operations_total", map[string]string{
			"operation": "get",
			"status":    "failure",
		})
		return &userpb.GetUserResponse{
			ErrorCode:    userpb.UserErrorCode_USER_ERROR_CODE_DATABASE_ERROR,
			ErrorMessage: fmt.Sprintf("Failed to retrieve user: %v", err),
		}, nil
	}

	s.obs.Metrics().IncrementCounter("user_operations_total", map[string]string{
		"operation": "get",
		"status":    "success",
	})
	s.obs.Metrics().IncrementCounter("user_db_operations_total", map[string]string{
		"operation": "get",
		"status":    "success",
	})

	return &userpb.GetUserResponse{
		User: user,
	}, nil
}

// BatchGetUsers retrieves multiple users' profiles
func (s *UserServiceServer) BatchGetUsers(ctx context.Context, req *userpb.BatchGetUsersRequest) (*userpb.BatchGetUsersResponse, error) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		s.obs.Metrics().RecordDuration("user_grpc_request_duration_seconds", duration, map[string]string{
			"method": "BatchGetUsers",
		})
	}()

	// Record request count
	s.obs.Metrics().IncrementCounter("user_grpc_requests_total", map[string]string{
		"method": "BatchGetUsers",
	})

	if len(req.UserIds) == 0 {
		return &userpb.BatchGetUsersResponse{
			Users: make(map[string]*userpb.UserProfile),
		}, nil
	}

	// Validate batch size
	if len(req.UserIds) > 100 {
		s.obs.Metrics().IncrementCounter("user_operations_total", map[string]string{
			"operation": "batch_get",
			"status":    "failure",
		})
		return &userpb.BatchGetUsersResponse{
			ErrorCode:    userpb.UserErrorCode_USER_ERROR_CODE_TOO_MANY_IDS,
			ErrorMessage: fmt.Sprintf("Too many user IDs: %d (max 100)", len(req.UserIds)),
		}, nil
	}

	dbStartTime := time.Now()
	users, err := s.store.BatchGetUsers(ctx, req.UserIds)
	dbDuration := time.Since(dbStartTime)

	s.obs.Metrics().RecordDuration("user_db_operation_duration_seconds", dbDuration, map[string]string{
		"operation": "batch_get",
	})

	if err != nil {
		s.obs.Metrics().IncrementCounter("user_operations_total", map[string]string{
			"operation": "batch_get",
			"status":    "failure",
		})
		s.obs.Metrics().IncrementCounter("user_db_operations_total", map[string]string{
			"operation": "batch_get",
			"status":    "failure",
		})
		return &userpb.BatchGetUsersResponse{
			ErrorCode:    userpb.UserErrorCode_USER_ERROR_CODE_DATABASE_ERROR,
			ErrorMessage: fmt.Sprintf("Failed to retrieve users: %v", err),
		}, nil
	}

	// Find not found user IDs
	notFoundIDs := make([]string, 0)
	for _, userID := range req.UserIds {
		if _, found := users[userID]; !found {
			notFoundIDs = append(notFoundIDs, userID)
		}
	}

	s.obs.Metrics().IncrementCounter("user_operations_total", map[string]string{
		"operation": "batch_get",
		"status":    "success",
	})
	s.obs.Metrics().IncrementCounter("user_db_operations_total", map[string]string{
		"operation": "batch_get",
		"status":    "success",
	})

	return &userpb.BatchGetUsersResponse{
		Users:           users,
		NotFoundUserIds: notFoundIDs,
	}, nil
}

// GetGroupMembers retrieves all members of a group with pagination
func (s *UserServiceServer) GetGroupMembers(ctx context.Context, req *userpb.GetGroupMembersRequest) (*userpb.GetGroupMembersResponse, error) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		s.obs.Metrics().RecordDuration("user_grpc_request_duration_seconds", duration, map[string]string{
			"method": "GetGroupMembers",
		})
	}()

	// Record request count
	s.obs.Metrics().IncrementCounter("user_grpc_requests_total", map[string]string{
		"method": "GetGroupMembers",
	})

	if req.GroupId == "" {
		s.obs.Metrics().IncrementCounter("user_operations_total", map[string]string{
			"operation": "get_group_members",
			"status":    "failure",
		})
		return &userpb.GetGroupMembersResponse{
			ErrorCode:    userpb.UserErrorCode_USER_ERROR_CODE_INVALID_REQUEST,
			ErrorMessage: "group_id is required",
		}, nil
	}

	// Default limit
	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	dbStartTime := time.Now()
	members, nextCursor, totalCount, err := s.store.GetGroupMembers(ctx, req.GroupId, req.Cursor, limit)
	dbDuration := time.Since(dbStartTime)

	s.obs.Metrics().RecordDuration("user_db_operation_duration_seconds", dbDuration, map[string]string{
		"operation": "get_group_members",
	})

	if err != nil {
		s.obs.Metrics().IncrementCounter("user_operations_total", map[string]string{
			"operation": "get_group_members",
			"status":    "failure",
		})
		s.obs.Metrics().IncrementCounter("user_db_operations_total", map[string]string{
			"operation": "get_group_members",
			"status":    "failure",
		})
		return &userpb.GetGroupMembersResponse{
			ErrorCode:    userpb.UserErrorCode_USER_ERROR_CODE_DATABASE_ERROR,
			ErrorMessage: fmt.Sprintf("Failed to retrieve group members: %v", err),
		}, nil
	}

	hasMore := nextCursor != ""

	s.obs.Metrics().IncrementCounter("user_operations_total", map[string]string{
		"operation": "get_group_members",
		"status":    "success",
	})
	s.obs.Metrics().IncrementCounter("user_db_operations_total", map[string]string{
		"operation": "get_group_members",
		"status":    "success",
	})

	return &userpb.GetGroupMembersResponse{
		Members:    members,
		NextCursor: nextCursor,
		TotalCount: totalCount,
		HasMore:    hasMore,
	}, nil
}

// ValidateGroupMembership checks if a user is a member of a group
func (s *UserServiceServer) ValidateGroupMembership(ctx context.Context, req *userpb.ValidateGroupMembershipRequest) (*userpb.ValidateGroupMembershipResponse, error) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		s.obs.Metrics().RecordDuration("user_grpc_request_duration_seconds", duration, map[string]string{
			"method": "ValidateGroupMembership",
		})
	}()

	// Record request count
	s.obs.Metrics().IncrementCounter("user_grpc_requests_total", map[string]string{
		"method": "ValidateGroupMembership",
	})

	if req.UserId == "" {
		s.obs.Metrics().IncrementCounter("user_operations_total", map[string]string{
			"operation": "validate_membership",
			"status":    "failure",
		})
		return &userpb.ValidateGroupMembershipResponse{
			ErrorCode:    userpb.UserErrorCode_USER_ERROR_CODE_INVALID_REQUEST,
			ErrorMessage: "user_id is required",
		}, nil
	}

	if req.GroupId == "" {
		s.obs.Metrics().IncrementCounter("user_operations_total", map[string]string{
			"operation": "validate_membership",
			"status":    "failure",
		})
		return &userpb.ValidateGroupMembershipResponse{
			ErrorCode:    userpb.UserErrorCode_USER_ERROR_CODE_INVALID_REQUEST,
			ErrorMessage: "group_id is required",
		}, nil
	}

	dbStartTime := time.Now()
	member, err := s.store.ValidateGroupMembership(ctx, req.UserId, req.GroupId)
	dbDuration := time.Since(dbStartTime)

	s.obs.Metrics().RecordDuration("user_db_operation_duration_seconds", dbDuration, map[string]string{
		"operation": "validate_membership",
	})

	if err != nil {
		s.obs.Metrics().IncrementCounter("user_operations_total", map[string]string{
			"operation": "validate_membership",
			"status":    "failure",
		})
		s.obs.Metrics().IncrementCounter("user_db_operations_total", map[string]string{
			"operation": "validate_membership",
			"status":    "failure",
		})
		return &userpb.ValidateGroupMembershipResponse{
			ErrorCode:    userpb.UserErrorCode_USER_ERROR_CODE_DATABASE_ERROR,
			ErrorMessage: fmt.Sprintf("Failed to validate membership: %v", err),
		}, nil
	}

	if member == nil {
		// Not a member
		s.obs.Metrics().IncrementCounter("user_operations_total", map[string]string{
			"operation": "validate_membership",
			"status":    "not_member",
		})
		s.obs.Metrics().IncrementCounter("user_db_operations_total", map[string]string{
			"operation": "validate_membership",
			"status":    "not_member",
		})
		return &userpb.ValidateGroupMembershipResponse{
			IsMember: false,
		}, nil
	}

	s.obs.Metrics().IncrementCounter("user_operations_total", map[string]string{
		"operation": "validate_membership",
		"status":    "success",
	})
	s.obs.Metrics().IncrementCounter("user_db_operations_total", map[string]string{
		"operation": "validate_membership",
		"status":    "success",
	})

	return &userpb.ValidateGroupMembershipResponse{
		IsMember: true,
		Member:   member,
	}, nil
}
