/**
 * User Page
 * 
 * User profile and group management interface.
 */

import { useState, useEffect } from 'react';
import { userService, type User } from '@/services/userService';

export function UserPage() {
  const [users, setUsers] = useState<User[]>([]);
  const [groups, setGroups] = useState<Array<{ groupId: string; memberCount: number }>>([]);
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [selectedGroup, setSelectedGroup] = useState<string | null>(null);
  const [groupMembers, setGroupMembers] = useState<Array<{ userId: string; role: string }>>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Load users and groups on mount
  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    setLoading(true);
    setError(null);

    try {
      const [usersData, groupsData] = await Promise.all([
        userService.getAllUsers(),
        userService.getAllGroups(),
      ]);

      setUsers(usersData);
      setGroups(groupsData);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load data');
    } finally {
      setLoading(false);
    }
  };

  const handleUserClick = async (userId: string) => {
    setLoading(true);
    setError(null);

    try {
      const response = await userService.getUser(userId);
      
      if (response.user) {
        setSelectedUser(response.user);
      } else {
        setError(response.errorMessage || 'User not found');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load user');
    } finally {
      setLoading(false);
    }
  };

  const handleGroupClick = async (groupId: string) => {
    setLoading(true);
    setError(null);
    setSelectedGroup(groupId);

    try {
      const response = await userService.getGroupMembers(groupId);
      setGroupMembers(response.members);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load group members');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{ maxWidth: '1200px', margin: '0 auto' }}>
      <h2 style={{ textAlign: 'center', marginBottom: '24px' }}>
        User & Group Management
      </h2>

      {/* Demo Notice */}
      <div style={{
        padding: '12px',
        backgroundColor: '#fff3cd',
        border: '1px solid #ffc107',
        borderRadius: '4px',
        marginBottom: '24px',
        fontSize: '14px',
      }}>
        <strong>Demo Mode:</strong> This is a mock user service with sample data.
        In production, this would connect to the real user-service gRPC API.
      </div>

      {/* Error Display */}
      {error && (
        <div style={{
          padding: '12px',
          backgroundColor: '#f8d7da',
          color: '#721c24',
          border: '1px solid #f5c6cb',
          borderRadius: '4px',
          marginBottom: '16px',
        }}>
          {error}
        </div>
      )}

      {/* Loading Indicator */}
      {loading && (
        <div style={{ textAlign: 'center', padding: '20px', color: '#666' }}>
          Loading...
        </div>
      )}

      <div style={{
        display: 'grid',
        gridTemplateColumns: '1fr 1fr',
        gap: '24px',
      }}>
        {/* Users List */}
        <div style={{
          border: '1px solid #ddd',
          borderRadius: '8px',
          padding: '20px',
          backgroundColor: '#fff',
        }}>
          <h3 style={{ marginTop: 0, marginBottom: '16px' }}>Users</h3>
          
          {users.length === 0 ? (
            <div style={{ textAlign: 'center', color: '#999', padding: '20px' }}>
              No users found
            </div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
              {users.map((user) => (
                <div
                  key={user.userId}
                  onClick={() => handleUserClick(user.userId)}
                  style={{
                    padding: '12px',
                    border: '1px solid #ddd',
                    borderRadius: '4px',
                    cursor: 'pointer',
                    backgroundColor: selectedUser?.userId === user.userId ? '#e3f2fd' : '#fafafa',
                    transition: 'background-color 0.2s',
                  }}
                  onMouseEnter={(e) => {
                    if (selectedUser?.userId !== user.userId) {
                      e.currentTarget.style.backgroundColor = '#f5f5f5';
                    }
                  }}
                  onMouseLeave={(e) => {
                    if (selectedUser?.userId !== user.userId) {
                      e.currentTarget.style.backgroundColor = '#fafafa';
                    }
                  }}
                >
                  <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                    {user.avatarUrl && (
                      <img
                        src={user.avatarUrl}
                        alt={user.username}
                        style={{
                          width: '40px',
                          height: '40px',
                          borderRadius: '50%',
                        }}
                      />
                    )}
                    <div>
                      <div style={{ fontWeight: 'bold' }}>{user.username}</div>
                      <div style={{ fontSize: '12px', color: '#666' }}>{user.userId}</div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Groups List */}
        <div style={{
          border: '1px solid #ddd',
          borderRadius: '8px',
          padding: '20px',
          backgroundColor: '#fff',
        }}>
          <h3 style={{ marginTop: 0, marginBottom: '16px' }}>Groups</h3>
          
          {groups.length === 0 ? (
            <div style={{ textAlign: 'center', color: '#999', padding: '20px' }}>
              No groups found
            </div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
              {groups.map((group) => (
                <div
                  key={group.groupId}
                  onClick={() => handleGroupClick(group.groupId)}
                  style={{
                    padding: '12px',
                    border: '1px solid #ddd',
                    borderRadius: '4px',
                    cursor: 'pointer',
                    backgroundColor: selectedGroup === group.groupId ? '#e3f2fd' : '#fafafa',
                    transition: 'background-color 0.2s',
                  }}
                  onMouseEnter={(e) => {
                    if (selectedGroup !== group.groupId) {
                      e.currentTarget.style.backgroundColor = '#f5f5f5';
                    }
                  }}
                  onMouseLeave={(e) => {
                    if (selectedGroup !== group.groupId) {
                      e.currentTarget.style.backgroundColor = '#fafafa';
                    }
                  }}
                >
                  <div style={{ fontWeight: 'bold' }}>{group.groupId}</div>
                  <div style={{ fontSize: '12px', color: '#666' }}>
                    {group.memberCount} member{group.memberCount !== 1 ? 's' : ''}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* User Details */}
      {selectedUser && (
        <div style={{
          marginTop: '24px',
          border: '1px solid #ddd',
          borderRadius: '8px',
          padding: '20px',
          backgroundColor: '#fff',
        }}>
          <h3 style={{ marginTop: 0, marginBottom: '16px' }}>User Details</h3>
          
          <div style={{ display: 'flex', gap: '24px', alignItems: 'flex-start' }}>
            {selectedUser.avatarUrl && (
              <img
                src={selectedUser.avatarUrl}
                alt={selectedUser.username}
                style={{
                  width: '80px',
                  height: '80px',
                  borderRadius: '50%',
                }}
              />
            )}
            
            <div style={{ flex: 1 }}>
              <div style={{ marginBottom: '12px' }}>
                <strong>Username:</strong> {selectedUser.username}
              </div>
              <div style={{ marginBottom: '12px' }}>
                <strong>User ID:</strong> {selectedUser.userId}
              </div>
              <div style={{ marginBottom: '12px' }}>
                <strong>Email:</strong> {selectedUser.email}
              </div>
              <div>
                <strong>Created:</strong>{' '}
                {new Date(selectedUser.createdAt).toLocaleDateString('en-US', {
                  year: 'numeric',
                  month: 'long',
                  day: 'numeric',
                })}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Group Members */}
      {selectedGroup && groupMembers.length > 0 && (
        <div style={{
          marginTop: '24px',
          border: '1px solid #ddd',
          borderRadius: '8px',
          padding: '20px',
          backgroundColor: '#fff',
        }}>
          <h3 style={{ marginTop: 0, marginBottom: '16px' }}>
            Group Members: {selectedGroup}
          </h3>
          
          <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
            {groupMembers.map((member) => (
              <div
                key={member.userId}
                style={{
                  padding: '12px',
                  border: '1px solid #ddd',
                  borderRadius: '4px',
                  backgroundColor: '#fafafa',
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'center',
                }}
              >
                <span>{member.userId}</span>
                <span style={{
                  padding: '4px 8px',
                  backgroundColor: member.role === 'owner' ? '#4CAF50' : member.role === 'admin' ? '#2196F3' : '#999',
                  color: '#fff',
                  borderRadius: '4px',
                  fontSize: '12px',
                  fontWeight: 'bold',
                }}>
                  {member.role.toUpperCase()}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
