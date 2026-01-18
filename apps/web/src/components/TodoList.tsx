import { useState } from 'react';
import { useTodos } from '../hooks/useTodos';
import { Todo } from '../gen/todo';

export function TodoList() {
  const { todos, isLoading, error, updateTodo, deleteTodo, isUpdating, isDeleting } = useTodos();
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editTitle, setEditTitle] = useState('');
  const [editDescription, setEditDescription] = useState('');

  if (isLoading) {
    return <div style={{ padding: '20px' }}>Loading todos...</div>;
  }

  if (error) {
    return (
      <div
        style={{
          padding: '20px',
          backgroundColor: '#f8d7da',
          color: '#721c24',
          border: '1px solid #f5c6cb',
          borderRadius: '4px',
        }}
      >
        Error loading todos: {error instanceof Error ? error.message : 'Unknown error'}
      </div>
    );
  }

  const handleEdit = (todo: Todo) => {
    setEditingId(todo.id);
    setEditTitle(todo.title);
    setEditDescription(todo.description);
  };

  const handleSave = (todo: Todo) => {
    updateTodo({
      id: todo.id,
      title: editTitle,
      description: editDescription,
      completed: todo.completed,
    });
    setEditingId(null);
  };

  const handleCancel = () => {
    setEditingId(null);
    setEditTitle('');
    setEditDescription('');
  };

  const handleToggleComplete = (todo: Todo) => {
    updateTodo({
      id: todo.id,
      title: todo.title,
      description: todo.description,
      completed: !todo.completed,
    });
  };

  const handleDelete = (id: string) => {
    if (window.confirm('Are you sure you want to delete this todo?')) {
      deleteTodo(id);
    }
  };

  return (
    <div style={{ padding: '20px' }}>
      <h2>TODO List</h2>
      {todos.length === 0 ? (
        <p style={{ color: '#666' }}>No todos yet. Create one to get started!</p>
      ) : (
        <div>
          {todos.map((todo) => (
            <div
              key={todo.id}
              style={{
                marginBottom: '15px',
                padding: '15px',
                border: '1px solid #ddd',
                borderRadius: '4px',
                backgroundColor: todo.completed ? '#f0f0f0' : 'white',
              }}
            >
              {editingId === todo.id ? (
                // Edit mode
                <div>
                  <input
                    type="text"
                    value={editTitle}
                    onChange={(e) => setEditTitle(e.target.value)}
                    style={{
                      width: '100%',
                      padding: '8px',
                      marginBottom: '8px',
                      fontSize: '16px',
                      border: '1px solid #ccc',
                      borderRadius: '4px',
                    }}
                  />
                  <textarea
                    value={editDescription}
                    onChange={(e) => setEditDescription(e.target.value)}
                    rows={3}
                    style={{
                      width: '100%',
                      padding: '8px',
                      marginBottom: '8px',
                      fontSize: '14px',
                      border: '1px solid #ccc',
                      borderRadius: '4px',
                    }}
                  />
                  <div>
                    <button
                      onClick={() => handleSave(todo)}
                      disabled={isUpdating}
                      style={{
                        padding: '6px 12px',
                        marginRight: '8px',
                        backgroundColor: '#28a745',
                        color: 'white',
                        border: 'none',
                        borderRadius: '4px',
                        cursor: isUpdating ? 'not-allowed' : 'pointer',
                      }}
                    >
                      Save
                    </button>
                    <button
                      onClick={handleCancel}
                      disabled={isUpdating}
                      style={{
                        padding: '6px 12px',
                        backgroundColor: '#6c757d',
                        color: 'white',
                        border: 'none',
                        borderRadius: '4px',
                        cursor: isUpdating ? 'not-allowed' : 'pointer',
                      }}
                    >
                      Cancel
                    </button>
                  </div>
                </div>
              ) : (
                // View mode
                <div>
                  <div style={{ display: 'flex', alignItems: 'center', marginBottom: '8px' }}>
                    <input
                      type="checkbox"
                      checked={todo.completed}
                      onChange={() => handleToggleComplete(todo)}
                      style={{ marginRight: '10px', cursor: 'pointer' }}
                    />
                    <h3
                      style={{
                        margin: 0,
                        textDecoration: todo.completed ? 'line-through' : 'none',
                        color: todo.completed ? '#666' : '#000',
                      }}
                    >
                      {todo.title}
                    </h3>
                  </div>
                  {todo.description && (
                    <p
                      style={{
                        margin: '8px 0',
                        color: todo.completed ? '#666' : '#333',
                      }}
                    >
                      {todo.description}
                    </p>
                  )}
                  <div style={{ marginTop: '10px' }}>
                    <button
                      onClick={() => handleEdit(todo)}
                      disabled={isUpdating || isDeleting}
                      style={{
                        padding: '6px 12px',
                        marginRight: '8px',
                        backgroundColor: '#007bff',
                        color: 'white',
                        border: 'none',
                        borderRadius: '4px',
                        cursor: isUpdating || isDeleting ? 'not-allowed' : 'pointer',
                      }}
                    >
                      Edit
                    </button>
                    <button
                      onClick={() => handleDelete(todo.id)}
                      disabled={isUpdating || isDeleting}
                      style={{
                        padding: '6px 12px',
                        backgroundColor: '#dc3545',
                        color: 'white',
                        border: 'none',
                        borderRadius: '4px',
                        cursor: isUpdating || isDeleting ? 'not-allowed' : 'pointer',
                      }}
                    >
                      Delete
                    </button>
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
