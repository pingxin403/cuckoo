//go:build property
// +build property

package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/pingxin403/cuckoo/apps/user-service/gen/userpb"
	"pgregory.net/rapid"
)

// Property 1: Batch get users returns all requested users that exist
// **Validates: Requirements 14.1**
func TestProperty_BatchGetUsersReturnsAllExisting(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		store := NewMockUserStore()
		svc := NewUserServiceServer(store)

		// Generate a random subset of existing user IDs
		existingIDs := []string{"user001", "user002", "user003"}
		numUsers := rapid.IntRange(1, 3).Draw(t, "num_users")

		// Randomly select user IDs
		selectedIDs := make([]string, 0, numUsers)
		for i := 0; i < numUsers; i++ {
			idx := rapid.IntRange(0, len(existingIDs)-1).Draw(t, fmt.Sprintf("user_idx_%d", i))
			selectedIDs = append(selectedIDs, existingIDs[idx])
		}

		// Remove duplicates
		uniqueIDs := make(map[string]bool)
		for _, id := range selectedIDs {
			uniqueIDs[id] = true
		}
		requestIDs := make([]string, 0, len(uniqueIDs))
		for id := range uniqueIDs {
			requestIDs = append(requestIDs, id)
		}

		req := &userpb.BatchGetUsersRequest{
			UserIds: requestIDs,
		}

		resp, err := svc.BatchGetUsers(context.Background(), req)

		// Property: All requested existing users must be returned
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if len(resp.Users) != len(requestIDs) {
			t.Fatalf("Expected %d users, got %d", len(requestIDs), len(resp.Users))
		}
		for _, userID := range requestIDs {
			if _, found := resp.Users[userID]; !found {
				t.Fatalf("Expected user %s to be in response", userID)
			}
		}
		if len(resp.NotFoundUserIds) != 0 {
			t.Fatalf("Expected no not-found users, got %d", len(resp.NotFoundUserIds))
		}
	})
}

// Property 2: Group membership validation is consistent
// **Validates: Requirements 2.1**
func TestProperty_GroupMembershipValidationConsistent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		store := NewMockUserStore()
		svc := NewUserServiceServer(store)

		// Test with known members and non-members
		testCases := []struct {
			userID   string
			groupID  string
			expected bool
		}{
			{"user001", "group001", true},      // Owner
			{"user002", "group001", true},      // Admin
			{"user003", "group001", true},      // Member
			{"user001", "nonexistent", false},  // Non-existent group
			{"nonexistent", "group001", false}, // Non-existent user
		}

		// Randomly select a test case
		idx := rapid.IntRange(0, len(testCases)-1).Draw(t, "test_case_idx")
		tc := testCases[idx]

		req := &userpb.ValidateGroupMembershipRequest{
			UserId:  tc.userID,
			GroupId: tc.groupID,
		}

		resp, err := svc.ValidateGroupMembership(context.Background(), req)

		// Property: Membership validation must be consistent
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if resp.IsMember != tc.expected {
			t.Fatalf("Expected is_member=%v for user=%s group=%s, got %v",
				tc.expected, tc.userID, tc.groupID, resp.IsMember)
		}
		if tc.expected && resp.Member == nil {
			t.Fatal("Expected member details when is_member=true")
		}
		if !tc.expected && resp.Member != nil {
			t.Fatal("Expected no member details when is_member=false")
		}
	})
}

// Property 3: Pagination returns all members without duplicates
// **Validates: Requirements 2.1**
func TestProperty_PaginationReturnsAllMembersWithoutDuplicates(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		store := NewMockUserStore()
		svc := NewUserServiceServer(store)

		// Use a random page size between 1 and 5
		// Safe conversion: rapid.IntRange returns int, we need int32
		pageSizeInt := rapid.IntRange(1, 5).Draw(t, "page_size")
		if pageSizeInt < 0 || pageSizeInt > 1000 {
			t.Fatalf("Invalid page size: %d", pageSizeInt)
		}
		pageSize := int32(pageSizeInt) // #nosec G115 - validated range

		// Fetch all pages
		allMembers := make(map[string]*userpb.GroupMember)
		cursor := ""
		pageCount := 0
		maxPages := 10 // Safety limit

		for pageCount < maxPages {
			req := &userpb.GetGroupMembersRequest{
				GroupId: "group001",
				Cursor:  cursor,
				Limit:   pageSize,
			}

			resp, err := svc.GetGroupMembers(context.Background(), req)
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			// Add members to map (will detect duplicates)
			for _, member := range resp.Members {
				if _, exists := allMembers[member.UserId]; exists {
					t.Fatalf("Duplicate member found: %s", member.UserId)
				}
				allMembers[member.UserId] = member
			}

			pageCount++

			// Check if we have more pages
			if !resp.HasMore || resp.NextCursor == "" {
				break
			}
			cursor = resp.NextCursor
		}

		// Property: Must retrieve all members exactly once
		expectedCount := int32(3) // group001 has 3 members
		// Safe conversion: len returns int, we need int32 for comparison
		actualCount := len(allMembers)
		if actualCount < 0 || actualCount > 10000 {
			t.Fatalf("Invalid member count: %d", actualCount)
		}
		if int32(actualCount) != expectedCount { // #nosec G115 - validated range
			t.Fatalf("Expected %d total members, got %d", expectedCount, actualCount)
		}

		// Verify all expected members are present
		expectedMembers := []string{"user001", "user002", "user003"}
		for _, userID := range expectedMembers {
			if _, found := allMembers[userID]; !found {
				t.Fatalf("Expected member %s not found in paginated results", userID)
			}
		}
	})
}

// Property 4: GetUser returns consistent data across multiple calls
// **Validates: Requirements 14.1**
func TestProperty_GetUserConsistentAcrossCalls(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		store := NewMockUserStore()
		svc := NewUserServiceServer(store)

		// Pick a random existing user
		userIDs := []string{"user001", "user002", "user003"}
		idx := rapid.IntRange(0, len(userIDs)-1).Draw(t, "user_idx")
		userID := userIDs[idx]

		// Call GetUser multiple times
		numCalls := rapid.IntRange(2, 5).Draw(t, "num_calls")
		var firstResp *userpb.GetUserResponse

		for i := range numCalls {
			req := &userpb.GetUserRequest{
				UserId: userID,
			}

			resp, err := svc.GetUser(context.Background(), req)
			if err != nil {
				t.Fatalf("Expected no error on call %d, got: %v", i, err)
			}

			if i == 0 {
				firstResp = resp
			} else {
				// Property: All calls must return identical data
				if resp.User.UserId != firstResp.User.UserId {
					t.Fatalf("Inconsistent user_id: %s vs %s", resp.User.UserId, firstResp.User.UserId)
				}
				if resp.User.Username != firstResp.User.Username {
					t.Fatalf("Inconsistent username: %s vs %s", resp.User.Username, firstResp.User.Username)
				}
				if resp.User.DisplayName != firstResp.User.DisplayName {
					t.Fatalf("Inconsistent display_name: %s vs %s", resp.User.DisplayName, firstResp.User.DisplayName)
				}
				if resp.User.Status != firstResp.User.Status {
					t.Fatalf("Inconsistent status: %v vs %v", resp.User.Status, firstResp.User.Status)
				}
			}
		}
	})
}

// Property 5: Batch get with non-existent users returns correct not-found list
// **Validates: Requirements 14.1**
func TestProperty_BatchGetUsersNotFoundList(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		store := NewMockUserStore()
		svc := NewUserServiceServer(store)

		// Generate a mix of existing and non-existing user IDs
		existingIDs := []string{"user001", "user002", "user003"}
		numExisting := rapid.IntRange(0, 3).Draw(t, "num_existing")
		numNonExisting := rapid.IntRange(1, 5).Draw(t, "num_non_existing")

		requestIDs := make([]string, 0, numExisting+numNonExisting)
		expectedNotFound := make(map[string]bool)

		// Add existing users
		for i := 0; i < numExisting; i++ {
			idx := rapid.IntRange(0, len(existingIDs)-1).Draw(t, fmt.Sprintf("existing_idx_%d", i))
			requestIDs = append(requestIDs, existingIDs[idx])
		}

		// Add non-existing users
		for i := 0; i < numNonExisting; i++ {
			nonExistentID := rapid.StringMatching(`^nonexistent[0-9]{3}$`).Draw(t, fmt.Sprintf("non_existing_%d", i))
			requestIDs = append(requestIDs, nonExistentID)
			expectedNotFound[nonExistentID] = true
		}

		req := &userpb.BatchGetUsersRequest{
			UserIds: requestIDs,
		}

		resp, err := svc.BatchGetUsers(context.Background(), req)

		// Property: Not-found list must contain exactly the non-existent user IDs
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Check that all non-existent IDs are in not-found list
		notFoundMap := make(map[string]bool)
		for _, id := range resp.NotFoundUserIds {
			notFoundMap[id] = true
		}

		for expectedID := range expectedNotFound {
			if !notFoundMap[expectedID] {
				t.Fatalf("Expected %s in not-found list", expectedID)
			}
		}

		// Check that no existing IDs are in not-found list
		for _, id := range resp.NotFoundUserIds {
			for _, existingID := range existingIDs {
				if id == existingID {
					t.Fatalf("Existing user %s should not be in not-found list", id)
				}
			}
		}
	})
}

// Property 6: Group member roles are preserved correctly
// **Validates: Requirements 2.1**
func TestProperty_GroupMemberRolesPreserved(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		store := NewMockUserStore()
		svc := NewUserServiceServer(store)

		// Known member roles in group001
		expectedRoles := map[string]userpb.GroupRole{
			"user001": userpb.GroupRole_GROUP_ROLE_OWNER,
			"user002": userpb.GroupRole_GROUP_ROLE_ADMIN,
			"user003": userpb.GroupRole_GROUP_ROLE_MEMBER,
		}

		// Randomly select a member to validate
		userIDs := []string{"user001", "user002", "user003"}
		idx := rapid.IntRange(0, len(userIDs)-1).Draw(t, "user_idx")
		userID := userIDs[idx]

		req := &userpb.ValidateGroupMembershipRequest{
			UserId:  userID,
			GroupId: "group001",
		}

		resp, err := svc.ValidateGroupMembership(context.Background(), req)

		// Property: Member role must match expected role
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if !resp.IsMember {
			t.Fatalf("Expected user %s to be a member", userID)
		}
		if resp.Member.Role != expectedRoles[userID] {
			t.Fatalf("Expected role %v for user %s, got %v",
				expectedRoles[userID], userID, resp.Member.Role)
		}
	})
}

// Property 7: Empty batch request returns empty results
// **Validates: Requirements 14.1**
func TestProperty_EmptyBatchRequestReturnsEmpty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		store := NewMockUserStore()
		svc := NewUserServiceServer(store)

		req := &userpb.BatchGetUsersRequest{
			UserIds: []string{},
		}

		resp, err := svc.BatchGetUsers(context.Background(), req)

		// Property: Empty request must return empty results
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if len(resp.Users) != 0 {
			t.Fatalf("Expected empty users map, got %d users", len(resp.Users))
		}
		if len(resp.NotFoundUserIds) != 0 {
			t.Fatalf("Expected empty not-found list, got %d IDs", len(resp.NotFoundUserIds))
		}
	})
}

// Property 8: Total count in GetGroupMembers is consistent across pages
// **Validates: Requirements 2.1**
func TestProperty_GroupMembersTotalCountConsistent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		store := NewMockUserStore()
		svc := NewUserServiceServer(store)

		// Use a random page size
		// Safe conversion: rapid.IntRange returns int, we need int32
		pageSizeInt := rapid.IntRange(1, 5).Draw(t, "page_size")
		if pageSizeInt < 0 || pageSizeInt > 1000 {
			t.Fatalf("Invalid page size: %d", pageSizeInt)
		}
		pageSize := int32(pageSizeInt) // #nosec G115 - validated range

		// Fetch multiple pages and verify total_count is consistent
		cursor := ""
		var firstTotalCount int32
		pageCount := 0
		maxPages := 10

		for pageCount < maxPages {
			req := &userpb.GetGroupMembersRequest{
				GroupId: "group001",
				Cursor:  cursor,
				Limit:   pageSize,
			}

			resp, err := svc.GetGroupMembers(context.Background(), req)
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if pageCount == 0 {
				firstTotalCount = resp.TotalCount
			} else {
				// Property: Total count must be consistent across all pages
				if resp.TotalCount != firstTotalCount {
					t.Fatalf("Inconsistent total_count: page 0 had %d, page %d has %d",
						firstTotalCount, pageCount, resp.TotalCount)
				}
			}

			pageCount++

			if !resp.HasMore || resp.NextCursor == "" {
				break
			}
			cursor = resp.NextCursor
		}
	})
}
