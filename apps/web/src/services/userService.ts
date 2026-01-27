/**
 * User Service
 * 
 * Client for user profile and group management operations.
 * Note: This is a mock implementation for demo purposes.
 * In production, integrate with real user-service gRPC API.
 */

export interface User {
  userId: string;
  username: string;
  email: string;
  avatarUrl?: string;
  createdAt: number;
}

export interface GroupMember {
  userId: string;
  role: 'owner' | 'admin' | 'member';
}

export interface GetUserResponse {
  user?: User;
  errorMessage?: string;
}

export interface BatchGetUsersResponse {
  users: User[];
  notFound: string[];
}

export interface GetGroupMembersResponse {
  members: GroupMember[];
  totalCount: number;
  nextPageToken?: string;
}

export interface ValidateGroupMembershipResponse {
  isMember: boolean;
  role?: 'owner' | 'admin' | 'member';
}

class UserService {
  // Mock data for demo
  private mockUsers: Map<string, User> = new Map([
    ['user_123', {
      userId: 'user_123',
      username: 'john_doe',
      email: 'john@example.com',
      avatarUrl: 'https://api.dicebear.com/7.x/avataaars/svg?seed=john',
      createdAt: Date.now() - 86400000 * 30,
    }],
    ['user_456', {
      userId: 'user_456',
      username: 'jane_smith',
      email: 'jane@example.com',
      avatarUrl: 'https://api.dicebear.com/7.x/avataaars/svg?seed=jane',
      createdAt: Date.now() - 86400000 * 60,
    }],
    ['user_789', {
      userId: 'user_789',
      username: 'bob_wilson',
      email: 'bob@example.com',
      avatarUrl: 'https://api.dicebear.com/7.x/avataaars/svg?seed=bob',
      createdAt: Date.now() - 86400000 * 90,
    }],
  ]);

  private mockGroups: Map<string, GroupMember[]> = new Map([
    ['group_001', [
      { userId: 'user_123', role: 'owner' },
      { userId: 'user_456', role: 'admin' },
      { userId: 'user_789', role: 'member' },
    ]],
    ['group_002', [
      { userId: 'user_456', role: 'owner' },
      { userId: 'user_789', role: 'member' },
    ]],
  ]);

  /**
   * Get user profile by user ID
   */
  async getUser(userId: string): Promise<GetUserResponse> {
    try {
      // Mock implementation - replace with real API call
      // In production: call user-service gRPC GetUser endpoint
      
      await new Promise(resolve => setTimeout(resolve, 200));

      const user = this.mockUsers.get(userId);
      
      if (!user) {
        return {
          errorMessage: 'User not found',
        };
      }

      return { user };
    } catch (error) {
      return {
        errorMessage: error instanceof Error ? error.message : 'Failed to get user',
      };
    }
  }

  /**
   * Batch get multiple users
   */
  async batchGetUsers(userIds: string[]): Promise<BatchGetUsersResponse> {
    try {
      // Mock implementation - replace with real API call
      // In production: call user-service gRPC BatchGetUsers endpoint
      
      await new Promise(resolve => setTimeout(resolve, 300));

      const users: User[] = [];
      const notFound: string[] = [];

      for (const userId of userIds) {
        const user = this.mockUsers.get(userId);
        if (user) {
          users.push(user);
        } else {
          notFound.push(userId);
        }
      }

      return { users, notFound };
    } catch {
      return {
        users: [],
        notFound: userIds,
      };
    }
  }

  /**
   * Get all users (for demo purposes)
   */
  async getAllUsers(): Promise<User[]> {
    try {
      await new Promise(resolve => setTimeout(resolve, 200));
      return Array.from(this.mockUsers.values());
    } catch {
      return [];
    }
  }

  /**
   * Get group members
   */
  async getGroupMembers(groupId: string, _pageSize = 100, _pageToken?: string): Promise<GetGroupMembersResponse> {
    try {
      // Mock implementation - replace with real API call
      // In production: call user-service gRPC GetGroupMembers endpoint
      
      await new Promise(resolve => setTimeout(resolve, 200));

      const members = this.mockGroups.get(groupId) || [];

      return {
        members,
        totalCount: members.length,
      };
    } catch {
      return {
        members: [],
        totalCount: 0,
      };
    }
  }

  /**
   * Get all groups (for demo purposes)
   */
  async getAllGroups(): Promise<Array<{ groupId: string; memberCount: number }>> {
    try {
      await new Promise(resolve => setTimeout(resolve, 200));
      
      return Array.from(this.mockGroups.entries()).map(([groupId, members]) => ({
        groupId,
        memberCount: members.length,
      }));
    } catch {
      return [];
    }
  }

  /**
   * Validate group membership
   */
  async validateGroupMembership(groupId: string, userId: string): Promise<ValidateGroupMembershipResponse> {
    try {
      // Mock implementation - replace with real API call
      // In production: call user-service gRPC ValidateGroupMembership endpoint
      
      await new Promise(resolve => setTimeout(resolve, 150));

      const members = this.mockGroups.get(groupId);
      
      if (!members) {
        return { isMember: false };
      }

      const member = members.find(m => m.userId === userId);
      
      if (!member) {
        return { isMember: false };
      }

      return {
        isMember: true,
        role: member.role,
      };
    } catch {
      return { isMember: false };
    }
  }
}

// Export singleton instance
export const userService = new UserService();
