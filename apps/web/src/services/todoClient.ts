import {
  CreateTodoRequest,
  CreateTodoResponse,
  ListTodosRequest,
  ListTodosResponse,
  UpdateTodoRequest,
  UpdateTodoResponse,
  DeleteTodoRequest,
  DeleteTodoResponse,
} from '../gen/todo';

// Simple wrapper for TODO Service using fetch
export class TodoServiceClient {
  private baseUrl: string;

  constructor(baseUrl: string = '/api/todo') {
    this.baseUrl = baseUrl;
  }

  private async call<TRequest, TResponse>(
    method: string,
    request: TRequest,
    requestEncoder: { encode: (req: TRequest) => { finish: () => Uint8Array } },
    responseDecoder: { decode: (bytes: Uint8Array) => TResponse },
  ): Promise<TResponse> {
    const url = `${this.baseUrl}/api.v1.TodoService/${method}`;

    // Encode the request
    const requestBytes = requestEncoder.encode(request).finish();

    try {
      const response = await fetch(url, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/grpc-web+proto',
          'X-Grpc-Web': '1',
        },
        body: requestBytes as BodyInit,
      });

      if (!response.ok) {
        throw new Error(`gRPC request failed: ${response.statusText}`);
      }

      const responseBytes = new Uint8Array(await response.arrayBuffer());

      // Skip the gRPC-Web frame header (5 bytes: 1 byte flags + 4 bytes length)
      const messageBytes = responseBytes.slice(5);

      return responseDecoder.decode(messageBytes);
    } catch (error) {
      console.error(`TODO service ${method} error:`, error);
      throw error;
    }
  }

  async createTodo(request: CreateTodoRequest): Promise<CreateTodoResponse> {
    return this.call('CreateTodo', request, CreateTodoRequest, CreateTodoResponse);
  }

  async listTodos(request: ListTodosRequest): Promise<ListTodosResponse> {
    return this.call('ListTodos', request, ListTodosRequest, ListTodosResponse);
  }

  async updateTodo(request: UpdateTodoRequest): Promise<UpdateTodoResponse> {
    return this.call('UpdateTodo', request, UpdateTodoRequest, UpdateTodoResponse);
  }

  async deleteTodo(request: DeleteTodoRequest): Promise<DeleteTodoResponse> {
    return this.call('DeleteTodo', request, DeleteTodoRequest, DeleteTodoResponse);
  }
}

export const todoClient = new TodoServiceClient();
