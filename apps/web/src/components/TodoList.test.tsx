import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { fireEvent } from '@testing-library/react';
import { TodoList } from './TodoList';

vi.mock('../hooks/useTodos', () => ({
  useTodos: vi.fn(),
}));

const mockUseTodos = await import('../hooks/useTodos');
const { useTodos } = mockUseTodos;

describe('TodoList', () => {
  const mockTodos = [
    { id: '1', title: 'Test Todo 1', description: 'Description 1', completed: false },
    { id: '2', title: 'Test Todo 2', description: 'Description 2', completed: true },
  ];

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders loading state', () => {
    vi.mocked(useTodos).mockReturnValue({
      todos: [],
      isLoading: true,
      error: null,
      updateTodo: vi.fn(),
      deleteTodo: vi.fn(),
      isUpdating: false,
      isDeleting: false,
    });

    render(<TodoList />);
    expect(screen.getByText(/loading todos/i)).toBeDefined();
  });

  it('renders error state', () => {
    vi.mocked(useTodos).mockReturnValue({
      todos: [],
      isLoading: false,
      error: new Error('Failed to load'),
      updateTodo: vi.fn(),
      deleteTodo: vi.fn(),
      isUpdating: false,
      isDeleting: false,
    });

    render(<TodoList />);
    expect(screen.getByText(/error loading todos/i)).toBeDefined();
  });

  it('renders empty state when no todos', () => {
    vi.mocked(useTodos).mockReturnValue({
      todos: [],
      isLoading: false,
      error: null,
      updateTodo: vi.fn(),
      deleteTodo: vi.fn(),
      isUpdating: false,
      isDeleting: false,
    });

    render(<TodoList />);
    expect(screen.getByText(/no todos yet/i)).toBeDefined();
  });

  it('renders todo items', () => {
    vi.mocked(useTodos).mockReturnValue({
      todos: mockTodos,
      isLoading: false,
      error: null,
      updateTodo: vi.fn(),
      deleteTodo: vi.fn(),
      isUpdating: false,
      isDeleting: false,
    });

    render(<TodoList />);
    expect(screen.getByText('Test Todo 1')).toBeDefined();
    expect(screen.getByText('Test Todo 2')).toBeDefined();
  });

  it('calls updateTodo when toggling completion', async () => {
    const mockUpdateTodo = vi.fn();
    
    vi.mocked(useTodos).mockReturnValue({
      todos: [mockTodos[0]],
      isLoading: false,
      error: null,
      updateTodo: mockUpdateTodo,
      deleteTodo: vi.fn(),
      isUpdating: false,
      isDeleting: false,
    });

    render(<TodoList />);
    
    const checkbox = screen.getByRole('checkbox');
    fireEvent.click(checkbox);

    expect(mockUpdateTodo).toHaveBeenCalledWith(
      expect.objectContaining({ id: '1', completed: true })
    );
  });

  it('calls deleteTodo when delete button clicked', async () => {
    const mockDeleteTodo = vi.fn();
    vi.stubGlobal('confirm', vi.fn().mockReturnValue(true));
    
    vi.mocked(useTodos).mockReturnValue({
      todos: [mockTodos[0]],
      isLoading: false,
      error: null,
      updateTodo: vi.fn(),
      deleteTodo: mockDeleteTodo,
      isUpdating: false,
      isDeleting: false,
    });

    render(<TodoList />);
    
    const deleteButton = screen.getByRole('button', { name: /delete/i });
    fireEvent.click(deleteButton);

    expect(confirm).toHaveBeenCalled();
    expect(mockDeleteTodo).toHaveBeenCalledWith('1');
  });

  it('enters edit mode when edit button clicked', () => {
    vi.mocked(useTodos).mockReturnValue({
      todos: [mockTodos[0]],
      isLoading: false,
      error: null,
      updateTodo: vi.fn(),
      deleteTodo: vi.fn(),
      isUpdating: false,
      isDeleting: false,
    });

    render(<TodoList />);
    
    const editButton = screen.getByRole('button', { name: /edit/i });
    fireEvent.click(editButton);

    expect(screen.getByRole('button', { name: /save/i })).toBeDefined();
    expect(screen.getByRole('button', { name: /cancel/i })).toBeDefined();
  });
});