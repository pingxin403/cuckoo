package service

import (
	"context"
	"fmt"

	"github.com/pingxin403/cuckoo/apps/user-service/gen/userpb"
	"github.com/pingxin403/cuckoo/apps/user-service/storage"
)

// UserServiceServer implements the UserService gRPC service
type UserServiceServer struct {
	userpb.UnimplementedUserServiceServer
	store storage.UserStore
}

// NewUserServiceServer creates a new UserServiceServer
func NewUserServiceServer(store storage.UserStore) *UserServiceServer {
	return &UserServiceServer{
		store: store,
	}
}

// GetUser retrieves a single user's profile
func (s *UserServiceServer) GetUser(ctx context.Context, req *userpb.GetUserRequest) (*userpb.GetUserResponse, error) {
	if req.UserId == "" {
		return &userpb.GetUserResponse{
			ErrorCode:    userpb.UserErrorCode_USER_ERROR_CODE_INVALID_REQUEST,
			ErrorMessage: "user_id is required",
		}, nil
	}

	user, err := s.store.GetUser(ctx, req.UserId)
	if err != nil {
		// Check if user not found
		if err.Error() == fmt.Sprintf("user not found: %s", req.UserId) {
			return &userpb.GetUserResponse{
				ErrorCode:    userpb.UserErrorCode_USER_ERROR_CODE_USER_NOT_FOUND,
				ErrorMessage: fmt.Sprintf("User %s not found", req.UserId),
			}, nil
		}

		// Database error
		return &userpb.GetUserResponse{
			ErrorCode:    userpb.UserErrorCode_USER_ERROR_CODE_DATABASE_ERROR,
			ErrorMessage: fmt.Sprintf("Failed to retrieve user: %v", err),
		}, nil
	}

	return &userpb.GetUserResponse{
		User: user,
	}, nil
}

// BatchGetUsers retrieves multiple users' profiles
func (s *UserServiceServer) BatchGetUsers(ctx context.Context, req *userpb.BatchGetUsersRequest) (*userpb.BatchGetUsersResponse, error) {
	if len(req.UserIds) == 0 {
		return &userpb.BatchGetUsersResponse{
			Users: make(map[string]*userpb.UserProfile),
		}, nil
	}

	// Validate batch size
	if len(req.UserIds) > 100 {
		return &userpb.BatchGetUsersResponse{
			ErrorCode:    userpb.UserErrorCode_USER_ERROR_CODE_TOO_MANY_IDS,
			ErrorMessage: fmt.Sprintf("Too many user IDs: %d (max 100)", len(req.UserIds)),
		}, nil
	}

	users, err := s.store.BatchGetUsers(ctx, req.UserIds)
	if err != nil {
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

	return &userpb.BatchGetUsersResponse{
		Users:           users,
		NotFoundUserIds: notFoundIDs,
	}, nil
}

// GetGroupMembers retrieves all members of a group with pagination
func (s *UserServiceServer) GetGroupMembers(ctx context.Context, req *userpb.GetGroupMembersRequest) (*userpb.GetGroupMembersResponse, error) {
	if req.GroupId == "" {
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

	members, nextCursor, totalCount, err := s.store.GetGroupMembers(ctx, req.GroupId, req.Cursor, limit)
	if err != nil {
		return &userpb.GetGroupMembersResponse{
			ErrorCode:    userpb.UserErrorCode_USER_ERROR_CODE_DATABASE_ERROR,
			ErrorMessage: fmt.Sprintf("Failed to retrieve group members: %v", err),
		}, nil
	}

	hasMore := nextCursor != ""

	return &userpb.GetGroupMembersResponse{
		Members:    members,
		NextCursor: nextCursor,
		TotalCount: totalCount,
		HasMore:    hasMore,
	}, nil
}

// ValidateGroupMembership checks if a user is a member of a group
func (s *UserServiceServer) ValidateGroupMembership(ctx context.Context, req *userpb.ValidateGroupMembershipRequest) (*userpb.ValidateGroupMembershipResponse, error) {
	if req.UserId == "" {
		return &userpb.ValidateGroupMembershipResponse{
			ErrorCode:    userpb.UserErrorCode_USER_ERROR_CODE_INVALID_REQUEST,
			ErrorMessage: "user_id is required",
		}, nil
	}

	if req.GroupId == "" {
		return &userpb.ValidateGroupMembershipResponse{
			ErrorCode:    userpb.UserErrorCode_USER_ERROR_CODE_INVALID_REQUEST,
			ErrorMessage: "group_id is required",
		}, nil
	}

	member, err := s.store.ValidateGroupMembership(ctx, req.UserId, req.GroupId)
	if err != nil {
		return &userpb.ValidateGroupMembershipResponse{
			ErrorCode:    userpb.UserErrorCode_USER_ERROR_CODE_DATABASE_ERROR,
			ErrorMessage: fmt.Sprintf("Failed to validate membership: %v", err),
		}, nil
	}

	if member == nil {
		// Not a member
		return &userpb.ValidateGroupMembershipResponse{
			IsMember: false,
		}, nil
	}

	return &userpb.ValidateGroupMembershipResponse{
		IsMember: true,
		Member:   member,
	}, nil
}
