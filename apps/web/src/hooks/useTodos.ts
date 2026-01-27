import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { todoClient } from '../services/todoClient';
import {
  ListTodosRequest,
  CreateTodoRequest,
  UpdateTodoRequest,
  DeleteTodoRequest,
} from '@cuckoo/api-gen/todopb/todo';

export function useTodos() {
  const queryClient = useQueryClient();

  // Query to fetch all todos
  const {
    data: todosResponse,
    isLoading,
    error,
  } = useQuery({
    queryKey: ['todos'],
    queryFn: async () => {
      const request: ListTodosRequest = {};
      return todoClient.listTodos(request);
    },
  });

  // Mutation to create a new todo
  const createMutation = useMutation({
    mutationFn: async (data: { title: string; description: string }) => {
      const request: CreateTodoRequest = {
        title: data.title,
        description: data.description,
      };
      return todoClient.createTodo(request);
    },
    onSuccess: () => {
      // Invalidate and refetch todos after creating
      queryClient.invalidateQueries({ queryKey: ['todos'] });
    },
  });

  // Mutation to update a todo
  const updateMutation = useMutation({
    mutationFn: async (data: {
      id: string;
      title: string;
      description: string;
      completed: boolean;
    }) => {
      const request: UpdateTodoRequest = {
        id: data.id,
        title: data.title,
        description: data.description,
        completed: data.completed,
      };
      return todoClient.updateTodo(request);
    },
    onSuccess: () => {
      // Invalidate and refetch todos after updating
      queryClient.invalidateQueries({ queryKey: ['todos'] });
    },
  });

  // Mutation to delete a todo
  const deleteMutation = useMutation({
    mutationFn: async (id: string) => {
      const request: DeleteTodoRequest = { id };
      return todoClient.deleteTodo(request);
    },
    onSuccess: () => {
      // Invalidate and refetch todos after deleting
      queryClient.invalidateQueries({ queryKey: ['todos'] });
    },
  });

  return {
    todos: todosResponse?.todos || [],
    isLoading,
    error,
    createTodo: createMutation.mutate,
    updateTodo: updateMutation.mutate,
    deleteTodo: deleteMutation.mutate,
    isCreating: createMutation.isPending,
    isUpdating: updateMutation.isPending,
    isDeleting: deleteMutation.isPending,
  };
}
