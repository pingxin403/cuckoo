package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/pingxin403/cuckoo/api/gen/userpb"
	"github.com/pingxin403/cuckoo/libs/observability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Helper function to create a test observability instance
func createTestObservability() observability.Observability {
	obs, _ := observability.New(observability.Config{
		ServiceName:   "user-service-test",
		EnableMetrics: false,
		LogLevel:      "error",
	})
	return obs
}

// MockUserStore implements UserStore for testing
type MockUserStore struct {
	users        map[string]*userpb.UserProfile
	groupMembers map[string]map[string]*userpb.GroupMember // group_id -> user_id -> member
	groupCounts  map[string]int32
}

func NewMockUserStore() *MockUserStore {
	now := time.Now()
	return &MockUserStore{
		users: map[string]*userpb.UserProfile{
			"user001": {
				UserId:      "user001",
				Username:    "alice",
				DisplayName: "Alice Smith",
				AvatarUrl:   "https://example.com/avatars/alice.jpg",
				Status:      userpb.UserStatus_USER_STATUS_ONLINE,
				CreatedAt:   timestamppb.New(now),
				UpdatedAt:   timestamppb.New(now),
			},
			"user002": {
				UserId:      "user002",
				Username:    "bob",
				DisplayName: "Bob Johnson",
				AvatarUrl:   "https://example.com/avatars/bob.jpg",
				Status:      userpb.UserStatus_USER_STATUS_OFFLINE,
				CreatedAt:   timestamppb.New(now),
				UpdatedAt:   timestamppb.New(now),
			},
			"user003": {
				UserId:      "user003",
				Username:    "charlie",
				DisplayName: "Charlie Brown",
				AvatarUrl:   "https://example.com/avatars/charlie.jpg",
				Status:      userpb.UserStatus_USER_STATUS_ONLINE,
				CreatedAt:   timestamppb.New(now),
				UpdatedAt:   timestamppb.New(now),
			},
		},
		groupMembers: map[string]map[string]*userpb.GroupMember{
			"group001": {
				"user001": {
					UserId:   "user001",
					GroupId:  "group001",
					Role:     userpb.GroupRole_GROUP_ROLE_OWNER,
					JoinedAt: timestamppb.New(now),
					IsMuted:  false,
				},
				"user002": {
					UserId:   "user002",
					GroupId:  "group001",
					Role:     userpb.GroupRole_GROUP_ROLE_ADMIN,
					JoinedAt: timestamppb.New(now),
					IsMuted:  false,
				},
				"user003": {
					UserId:   "user003",
					GroupId:  "group001",
					Role:     userpb.GroupRole_GROUP_ROLE_MEMBER,
					JoinedAt: timestamppb.New(now),
					IsMuted:  false,
				},
			},
		},
		groupCounts: map[string]int32{
			"group001": 3,
		},
	}
}

func (m *MockUserStore) GetUser(ctx context.Context, userID string) (*userpb.UserProfile, error) {
	user, ok := m.users[userID]
	if !ok {
		return nil, fmt.Errorf("user not found: %s", userID)
	}
	return user, nil
}

func (m *MockUserStore) BatchGetUsers(ctx context.Context, userIDs []string) (map[string]*userpb.UserProfile, error) {
	result := make(map[string]*userpb.UserProfile)
	for _, userID := range userIDs {
		if user, ok := m.users[userID]; ok {
			result[userID] = user
		}
	}
	return result, nil
}

func (m *MockUserStore) GetGroupMembers(ctx context.Context, groupID string, cursor string, limit int32) ([]*userpb.GroupMember, string, int32, error) {
	members, ok := m.groupMembers[groupID]
	if !ok {
		return []*userpb.GroupMember{}, "", 0, nil
	}

	// Convert map to slice and sort by user_id for consistent ordering
	memberList := make([]*userpb.GroupMember, 0, len(members))
	userIDs := make([]string, 0, len(members))
	for userID := range members {
		userIDs = append(userIDs, userID)
	}

	// Sort user IDs for consistent ordering
	for i := 0; i < len(userIDs); i++ {
		for j := i + 1; j < len(userIDs); j++ {
			if userIDs[i] > userIDs[j] {
				userIDs[i], userIDs[j] = userIDs[j], userIDs[i]
			}
		}
	}

	// Build member list in sorted order, applying cursor filter
	for _, userID := range userIDs {
		if cursor == "" || userID > cursor {
			memberList = append(memberList, members[userID])
		}
	}

	// Simple pagination
	totalCount := m.groupCounts[groupID]
	// Safe conversion: limit is int32, len is int
	memberListLen := len(memberList)
	if memberListLen > 0 && limit > 0 && int32(memberListLen) > limit {
		nextCursor := memberList[limit-1].UserId
		return memberList[:limit], nextCursor, totalCount, nil
	}

	return memberList, "", totalCount, nil
}

func (m *MockUserStore) ValidateGroupMembership(ctx context.Context, userID, groupID string) (*userpb.GroupMember, error) {
	if members, ok := m.groupMembers[groupID]; ok {
		if member, ok := members[userID]; ok {
			return member, nil
		}
	}
	return nil, nil
}

// Test GetUser
func TestGetUser_Success(t *testing.T) {
	store := NewMockUserStore()
	obs := createTestObservability()
	svc := NewUserServiceServer(store, obs)

	req := &userpb.GetUserRequest{
		UserId: "user001",
	}

	resp, err := svc.GetUser(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotNil(t, resp.User)
	assert.Equal(t, "user001", resp.User.UserId)
	assert.Equal(t, "alice", resp.User.Username)
	assert.Equal(t, "Alice Smith", resp.User.DisplayName)
	assert.Equal(t, userpb.UserStatus_USER_STATUS_ONLINE, resp.User.Status)
}

func TestGetUser_NotFound(t *testing.T) {
	store := NewMockUserStore()
	obs := createTestObservability()
	svc := NewUserServiceServer(store, obs)

	req := &userpb.GetUserRequest{
		UserId: "nonexistent",
	}

	resp, err := svc.GetUser(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Nil(t, resp.User)
	assert.Equal(t, userpb.UserErrorCode_USER_ERROR_CODE_USER_NOT_FOUND, resp.ErrorCode)
	assert.Contains(t, resp.ErrorMessage, "not found")
}

func TestGetUser_EmptyUserID(t *testing.T) {
	store := NewMockUserStore()
	obs := createTestObservability()
	svc := NewUserServiceServer(store, obs)

	req := &userpb.GetUserRequest{
		UserId: "",
	}

	resp, err := svc.GetUser(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, userpb.UserErrorCode_USER_ERROR_CODE_INVALID_REQUEST, resp.ErrorCode)
	assert.Contains(t, resp.ErrorMessage, "required")
}

// Test BatchGetUsers
func TestBatchGetUsers_Success(t *testing.T) {
	store := NewMockUserStore()
	obs := createTestObservability()
	svc := NewUserServiceServer(store, obs)

	req := &userpb.BatchGetUsersRequest{
		UserIds: []string{"user001", "user002", "user003"},
	}

	resp, err := svc.BatchGetUsers(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Users, 3)
	assert.Contains(t, resp.Users, "user001")
	assert.Contains(t, resp.Users, "user002")
	assert.Contains(t, resp.Users, "user003")
	assert.Empty(t, resp.NotFoundUserIds)
}

func TestBatchGetUsers_PartialResults(t *testing.T) {
	store := NewMockUserStore()
	obs := createTestObservability()
	svc := NewUserServiceServer(store, obs)

	req := &userpb.BatchGetUsersRequest{
		UserIds: []string{"user001", "nonexistent1", "user002", "nonexistent2"},
	}

	resp, err := svc.BatchGetUsers(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Users, 2)
	assert.Contains(t, resp.Users, "user001")
	assert.Contains(t, resp.Users, "user002")
	assert.Len(t, resp.NotFoundUserIds, 2)
	assert.Contains(t, resp.NotFoundUserIds, "nonexistent1")
	assert.Contains(t, resp.NotFoundUserIds, "nonexistent2")
}

func TestBatchGetUsers_EmptyRequest(t *testing.T) {
	store := NewMockUserStore()
	obs := createTestObservability()
	svc := NewUserServiceServer(store, obs)

	req := &userpb.BatchGetUsersRequest{
		UserIds: []string{},
	}

	resp, err := svc.BatchGetUsers(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Empty(t, resp.Users)
	assert.Empty(t, resp.NotFoundUserIds)
}

func TestBatchGetUsers_TooManyIDs(t *testing.T) {
	store := NewMockUserStore()
	obs := createTestObservability()
	svc := NewUserServiceServer(store, obs)

	// Create 101 user IDs
	userIDs := make([]string, 101)
	for i := range 101 {
		userIDs[i] = fmt.Sprintf("user%03d", i)
	}

	req := &userpb.BatchGetUsersRequest{
		UserIds: userIDs,
	}

	resp, err := svc.BatchGetUsers(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, userpb.UserErrorCode_USER_ERROR_CODE_TOO_MANY_IDS, resp.ErrorCode)
	assert.Contains(t, resp.ErrorMessage, "Too many")
}

// Test GetGroupMembers
func TestGetGroupMembers_Success(t *testing.T) {
	store := NewMockUserStore()
	obs := createTestObservability()
	svc := NewUserServiceServer(store, obs)

	req := &userpb.GetGroupMembersRequest{
		GroupId: "group001",
		Limit:   100,
	}

	resp, err := svc.GetGroupMembers(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Members, 3)
	assert.Equal(t, int32(3), resp.TotalCount)
	assert.False(t, resp.HasMore)
	assert.Empty(t, resp.NextCursor)
}

func TestGetGroupMembers_WithPagination(t *testing.T) {
	store := NewMockUserStore()
	obs := createTestObservability()
	svc := NewUserServiceServer(store, obs)

	// First page
	req := &userpb.GetGroupMembersRequest{
		GroupId: "group001",
		Limit:   2,
	}

	resp, err := svc.GetGroupMembers(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Members, 2)
	assert.Equal(t, int32(3), resp.TotalCount)
	assert.True(t, resp.HasMore)
	assert.NotEmpty(t, resp.NextCursor)

	// Second page
	req2 := &userpb.GetGroupMembersRequest{
		GroupId: "group001",
		Cursor:  resp.NextCursor,
		Limit:   2,
	}

	resp2, err := svc.GetGroupMembers(context.Background(), req2)
	require.NoError(t, err)
	require.NotNil(t, resp2)
	assert.LessOrEqual(t, len(resp2.Members), 2)
	assert.Equal(t, int32(3), resp2.TotalCount)
}

func TestGetGroupMembers_EmptyGroupID(t *testing.T) {
	store := NewMockUserStore()
	obs := createTestObservability()
	svc := NewUserServiceServer(store, obs)

	req := &userpb.GetGroupMembersRequest{
		GroupId: "",
		Limit:   100,
	}

	resp, err := svc.GetGroupMembers(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, userpb.UserErrorCode_USER_ERROR_CODE_INVALID_REQUEST, resp.ErrorCode)
	assert.Contains(t, resp.ErrorMessage, "required")
}

func TestGetGroupMembers_DefaultLimit(t *testing.T) {
	store := NewMockUserStore()
	obs := createTestObservability()
	svc := NewUserServiceServer(store, obs)

	req := &userpb.GetGroupMembersRequest{
		GroupId: "group001",
		Limit:   0, // Should default to 100
	}

	resp, err := svc.GetGroupMembers(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Members, 3)
}

func TestGetGroupMembers_MaxLimit(t *testing.T) {
	store := NewMockUserStore()
	obs := createTestObservability()
	svc := NewUserServiceServer(store, obs)

	req := &userpb.GetGroupMembersRequest{
		GroupId: "group001",
		Limit:   2000, // Should be capped at 1000
	}

	resp, err := svc.GetGroupMembers(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	// Should succeed with capped limit
	assert.NotNil(t, resp.Members)
}

// Test ValidateGroupMembership
func TestValidateGroupMembership_IsMember(t *testing.T) {
	store := NewMockUserStore()
	obs := createTestObservability()
	svc := NewUserServiceServer(store, obs)

	req := &userpb.ValidateGroupMembershipRequest{
		UserId:  "user001",
		GroupId: "group001",
	}

	resp, err := svc.ValidateGroupMembership(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.IsMember)
	assert.NotNil(t, resp.Member)
	assert.Equal(t, "user001", resp.Member.UserId)
	assert.Equal(t, "group001", resp.Member.GroupId)
	assert.Equal(t, userpb.GroupRole_GROUP_ROLE_OWNER, resp.Member.Role)
}

func TestValidateGroupMembership_NotMember(t *testing.T) {
	store := NewMockUserStore()
	obs := createTestObservability()
	svc := NewUserServiceServer(store, obs)

	req := &userpb.ValidateGroupMembershipRequest{
		UserId:  "user001",
		GroupId: "nonexistent_group",
	}

	resp, err := svc.ValidateGroupMembership(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.False(t, resp.IsMember)
	assert.Nil(t, resp.Member)
}

func TestValidateGroupMembership_EmptyUserID(t *testing.T) {
	store := NewMockUserStore()
	obs := createTestObservability()
	svc := NewUserServiceServer(store, obs)

	req := &userpb.ValidateGroupMembershipRequest{
		UserId:  "",
		GroupId: "group001",
	}

	resp, err := svc.ValidateGroupMembership(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, userpb.UserErrorCode_USER_ERROR_CODE_INVALID_REQUEST, resp.ErrorCode)
	assert.Contains(t, resp.ErrorMessage, "required")
}

func TestValidateGroupMembership_EmptyGroupID(t *testing.T) {
	store := NewMockUserStore()
	obs := createTestObservability()
	svc := NewUserServiceServer(store, obs)

	req := &userpb.ValidateGroupMembershipRequest{
		UserId:  "user001",
		GroupId: "",
	}

	resp, err := svc.ValidateGroupMembership(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, userpb.UserErrorCode_USER_ERROR_CODE_INVALID_REQUEST, resp.ErrorCode)
	assert.Contains(t, resp.ErrorMessage, "required")
}
