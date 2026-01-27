package com.pingxin403.cuckoo.todo.v1;

import static io.grpc.MethodDescriptor.generateFullMethodName;

/**
 * <pre>
 * TodoService 提供 TODO 任务管理功能
 * 该服务支持完整的 CRUD 操作，允许用户创建、查询、更新和删除 TODO 任务。
 * 所有任务都有唯一的 ID，并记录创建和更新时间。
 * </pre>
 */
@io.grpc.stub.annotations.GrpcGenerated
public final class TodoServiceGrpc {

  private TodoServiceGrpc() {}

  public static final java.lang.String SERVICE_NAME = "api.v1.TodoService";

  // Static method descriptors that strictly reflect the proto.
  private static volatile io.grpc.MethodDescriptor<com.pingxin403.cuckoo.todo.v1.CreateTodoRequest,
      com.pingxin403.cuckoo.todo.v1.CreateTodoResponse> getCreateTodoMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "CreateTodo",
      requestType = com.pingxin403.cuckoo.todo.v1.CreateTodoRequest.class,
      responseType = com.pingxin403.cuckoo.todo.v1.CreateTodoResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<com.pingxin403.cuckoo.todo.v1.CreateTodoRequest,
      com.pingxin403.cuckoo.todo.v1.CreateTodoResponse> getCreateTodoMethod() {
    io.grpc.MethodDescriptor<com.pingxin403.cuckoo.todo.v1.CreateTodoRequest, com.pingxin403.cuckoo.todo.v1.CreateTodoResponse> getCreateTodoMethod;
    if ((getCreateTodoMethod = TodoServiceGrpc.getCreateTodoMethod) == null) {
      synchronized (TodoServiceGrpc.class) {
        if ((getCreateTodoMethod = TodoServiceGrpc.getCreateTodoMethod) == null) {
          TodoServiceGrpc.getCreateTodoMethod = getCreateTodoMethod =
              io.grpc.MethodDescriptor.<com.pingxin403.cuckoo.todo.v1.CreateTodoRequest, com.pingxin403.cuckoo.todo.v1.CreateTodoResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "CreateTodo"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  com.pingxin403.cuckoo.todo.v1.CreateTodoRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  com.pingxin403.cuckoo.todo.v1.CreateTodoResponse.getDefaultInstance()))
              .setSchemaDescriptor(new TodoServiceMethodDescriptorSupplier("CreateTodo"))
              .build();
        }
      }
    }
    return getCreateTodoMethod;
  }

  private static volatile io.grpc.MethodDescriptor<com.pingxin403.cuckoo.todo.v1.ListTodosRequest,
      com.pingxin403.cuckoo.todo.v1.ListTodosResponse> getListTodosMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "ListTodos",
      requestType = com.pingxin403.cuckoo.todo.v1.ListTodosRequest.class,
      responseType = com.pingxin403.cuckoo.todo.v1.ListTodosResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<com.pingxin403.cuckoo.todo.v1.ListTodosRequest,
      com.pingxin403.cuckoo.todo.v1.ListTodosResponse> getListTodosMethod() {
    io.grpc.MethodDescriptor<com.pingxin403.cuckoo.todo.v1.ListTodosRequest, com.pingxin403.cuckoo.todo.v1.ListTodosResponse> getListTodosMethod;
    if ((getListTodosMethod = TodoServiceGrpc.getListTodosMethod) == null) {
      synchronized (TodoServiceGrpc.class) {
        if ((getListTodosMethod = TodoServiceGrpc.getListTodosMethod) == null) {
          TodoServiceGrpc.getListTodosMethod = getListTodosMethod =
              io.grpc.MethodDescriptor.<com.pingxin403.cuckoo.todo.v1.ListTodosRequest, com.pingxin403.cuckoo.todo.v1.ListTodosResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "ListTodos"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  com.pingxin403.cuckoo.todo.v1.ListTodosRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  com.pingxin403.cuckoo.todo.v1.ListTodosResponse.getDefaultInstance()))
              .setSchemaDescriptor(new TodoServiceMethodDescriptorSupplier("ListTodos"))
              .build();
        }
      }
    }
    return getListTodosMethod;
  }

  private static volatile io.grpc.MethodDescriptor<com.pingxin403.cuckoo.todo.v1.UpdateTodoRequest,
      com.pingxin403.cuckoo.todo.v1.UpdateTodoResponse> getUpdateTodoMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "UpdateTodo",
      requestType = com.pingxin403.cuckoo.todo.v1.UpdateTodoRequest.class,
      responseType = com.pingxin403.cuckoo.todo.v1.UpdateTodoResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<com.pingxin403.cuckoo.todo.v1.UpdateTodoRequest,
      com.pingxin403.cuckoo.todo.v1.UpdateTodoResponse> getUpdateTodoMethod() {
    io.grpc.MethodDescriptor<com.pingxin403.cuckoo.todo.v1.UpdateTodoRequest, com.pingxin403.cuckoo.todo.v1.UpdateTodoResponse> getUpdateTodoMethod;
    if ((getUpdateTodoMethod = TodoServiceGrpc.getUpdateTodoMethod) == null) {
      synchronized (TodoServiceGrpc.class) {
        if ((getUpdateTodoMethod = TodoServiceGrpc.getUpdateTodoMethod) == null) {
          TodoServiceGrpc.getUpdateTodoMethod = getUpdateTodoMethod =
              io.grpc.MethodDescriptor.<com.pingxin403.cuckoo.todo.v1.UpdateTodoRequest, com.pingxin403.cuckoo.todo.v1.UpdateTodoResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "UpdateTodo"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  com.pingxin403.cuckoo.todo.v1.UpdateTodoRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  com.pingxin403.cuckoo.todo.v1.UpdateTodoResponse.getDefaultInstance()))
              .setSchemaDescriptor(new TodoServiceMethodDescriptorSupplier("UpdateTodo"))
              .build();
        }
      }
    }
    return getUpdateTodoMethod;
  }

  private static volatile io.grpc.MethodDescriptor<com.pingxin403.cuckoo.todo.v1.DeleteTodoRequest,
      com.pingxin403.cuckoo.todo.v1.DeleteTodoResponse> getDeleteTodoMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "DeleteTodo",
      requestType = com.pingxin403.cuckoo.todo.v1.DeleteTodoRequest.class,
      responseType = com.pingxin403.cuckoo.todo.v1.DeleteTodoResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<com.pingxin403.cuckoo.todo.v1.DeleteTodoRequest,
      com.pingxin403.cuckoo.todo.v1.DeleteTodoResponse> getDeleteTodoMethod() {
    io.grpc.MethodDescriptor<com.pingxin403.cuckoo.todo.v1.DeleteTodoRequest, com.pingxin403.cuckoo.todo.v1.DeleteTodoResponse> getDeleteTodoMethod;
    if ((getDeleteTodoMethod = TodoServiceGrpc.getDeleteTodoMethod) == null) {
      synchronized (TodoServiceGrpc.class) {
        if ((getDeleteTodoMethod = TodoServiceGrpc.getDeleteTodoMethod) == null) {
          TodoServiceGrpc.getDeleteTodoMethod = getDeleteTodoMethod =
              io.grpc.MethodDescriptor.<com.pingxin403.cuckoo.todo.v1.DeleteTodoRequest, com.pingxin403.cuckoo.todo.v1.DeleteTodoResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "DeleteTodo"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  com.pingxin403.cuckoo.todo.v1.DeleteTodoRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  com.pingxin403.cuckoo.todo.v1.DeleteTodoResponse.getDefaultInstance()))
              .setSchemaDescriptor(new TodoServiceMethodDescriptorSupplier("DeleteTodo"))
              .build();
        }
      }
    }
    return getDeleteTodoMethod;
  }

  /**
   * Creates a new async stub that supports all call types for the service
   */
  public static TodoServiceStub newStub(io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<TodoServiceStub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<TodoServiceStub>() {
        @java.lang.Override
        public TodoServiceStub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new TodoServiceStub(channel, callOptions);
        }
      };
    return TodoServiceStub.newStub(factory, channel);
  }

  /**
   * Creates a new blocking-style stub that supports all types of calls on the service
   */
  public static TodoServiceBlockingV2Stub newBlockingV2Stub(
      io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<TodoServiceBlockingV2Stub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<TodoServiceBlockingV2Stub>() {
        @java.lang.Override
        public TodoServiceBlockingV2Stub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new TodoServiceBlockingV2Stub(channel, callOptions);
        }
      };
    return TodoServiceBlockingV2Stub.newStub(factory, channel);
  }

  /**
   * Creates a new blocking-style stub that supports unary and streaming output calls on the service
   */
  public static TodoServiceBlockingStub newBlockingStub(
      io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<TodoServiceBlockingStub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<TodoServiceBlockingStub>() {
        @java.lang.Override
        public TodoServiceBlockingStub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new TodoServiceBlockingStub(channel, callOptions);
        }
      };
    return TodoServiceBlockingStub.newStub(factory, channel);
  }

  /**
   * Creates a new ListenableFuture-style stub that supports unary calls on the service
   */
  public static TodoServiceFutureStub newFutureStub(
      io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<TodoServiceFutureStub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<TodoServiceFutureStub>() {
        @java.lang.Override
        public TodoServiceFutureStub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new TodoServiceFutureStub(channel, callOptions);
        }
      };
    return TodoServiceFutureStub.newStub(factory, channel);
  }

  /**
   * <pre>
   * TodoService 提供 TODO 任务管理功能
   * 该服务支持完整的 CRUD 操作，允许用户创建、查询、更新和删除 TODO 任务。
   * 所有任务都有唯一的 ID，并记录创建和更新时间。
   * </pre>
   */
  public interface AsyncService {

    /**
     * <pre>
     * CreateTodo 创建新的 TODO 任务
     * 创建一个新的 TODO 项并返回包含生成的唯一 ID 和时间戳的完整对象。
     * 标题字段是必需的，不能为空。
     * Example:
     *   Request: { title: "Buy groceries", description: "Milk, eggs, bread" }
     *   Response: { todo: { id: "uuid", title: "Buy groceries", ... } }
     * Errors:
     *   - INVALID_ARGUMENT: 如果标题为空或仅包含空格
     * </pre>
     */
    default void createTodo(com.pingxin403.cuckoo.todo.v1.CreateTodoRequest request,
        io.grpc.stub.StreamObserver<com.pingxin403.cuckoo.todo.v1.CreateTodoResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getCreateTodoMethod(), responseObserver);
    }

    /**
     * <pre>
     * ListTodos 获取所有 TODO 任务
     * 返回系统中所有 TODO 任务的列表。任务按创建时间排序（最新的在前）。
     * Example:
     *   Request: {}
     *   Response: { todos: [{ id: "1", title: "Task 1" }, ...] }
     * </pre>
     */
    default void listTodos(com.pingxin403.cuckoo.todo.v1.ListTodosRequest request,
        io.grpc.stub.StreamObserver<com.pingxin403.cuckoo.todo.v1.ListTodosResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getListTodosMethod(), responseObserver);
    }

    /**
     * <pre>
     * UpdateTodo 更新现有的 TODO 任务
     * 更新指定 ID 的 TODO 任务。可以更新标题、描述和完成状态。
     * 更新操作会自动更新 updated_at 时间戳。
     * Example:
     *   Request: { id: "uuid", title: "Updated title", completed: true }
     *   Response: { todo: { id: "uuid", title: "Updated title", completed: true, ... } }
     * Errors:
     *   - NOT_FOUND: 如果指定的 TODO 不存在
     *   - INVALID_ARGUMENT: 如果 ID 为空或标题为空
     * </pre>
     */
    default void updateTodo(com.pingxin403.cuckoo.todo.v1.UpdateTodoRequest request,
        io.grpc.stub.StreamObserver<com.pingxin403.cuckoo.todo.v1.UpdateTodoResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getUpdateTodoMethod(), responseObserver);
    }

    /**
     * <pre>
     * DeleteTodo 删除指定的 TODO 任务
     * 根据 ID 删除 TODO 任务。删除操作是幂等的，重复删除同一个 ID 不会报错。
     * Example:
     *   Request: { id: "uuid" }
     *   Response: { success: true }
     * Errors:
     *   - NOT_FOUND: 如果指定的 TODO 不存在
     *   - INVALID_ARGUMENT: 如果 ID 为空
     * </pre>
     */
    default void deleteTodo(com.pingxin403.cuckoo.todo.v1.DeleteTodoRequest request,
        io.grpc.stub.StreamObserver<com.pingxin403.cuckoo.todo.v1.DeleteTodoResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getDeleteTodoMethod(), responseObserver);
    }
  }

  /**
   * Base class for the server implementation of the service TodoService.
   * <pre>
   * TodoService 提供 TODO 任务管理功能
   * 该服务支持完整的 CRUD 操作，允许用户创建、查询、更新和删除 TODO 任务。
   * 所有任务都有唯一的 ID，并记录创建和更新时间。
   * </pre>
   */
  public static abstract class TodoServiceImplBase
      implements io.grpc.BindableService, AsyncService {

    @java.lang.Override public final io.grpc.ServerServiceDefinition bindService() {
      return TodoServiceGrpc.bindService(this);
    }
  }

  /**
   * A stub to allow clients to do asynchronous rpc calls to service TodoService.
   * <pre>
   * TodoService 提供 TODO 任务管理功能
   * 该服务支持完整的 CRUD 操作，允许用户创建、查询、更新和删除 TODO 任务。
   * 所有任务都有唯一的 ID，并记录创建和更新时间。
   * </pre>
   */
  public static final class TodoServiceStub
      extends io.grpc.stub.AbstractAsyncStub<TodoServiceStub> {
    private TodoServiceStub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected TodoServiceStub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new TodoServiceStub(channel, callOptions);
    }

    /**
     * <pre>
     * CreateTodo 创建新的 TODO 任务
     * 创建一个新的 TODO 项并返回包含生成的唯一 ID 和时间戳的完整对象。
     * 标题字段是必需的，不能为空。
     * Example:
     *   Request: { title: "Buy groceries", description: "Milk, eggs, bread" }
     *   Response: { todo: { id: "uuid", title: "Buy groceries", ... } }
     * Errors:
     *   - INVALID_ARGUMENT: 如果标题为空或仅包含空格
     * </pre>
     */
    public void createTodo(com.pingxin403.cuckoo.todo.v1.CreateTodoRequest request,
        io.grpc.stub.StreamObserver<com.pingxin403.cuckoo.todo.v1.CreateTodoResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getCreateTodoMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     * <pre>
     * ListTodos 获取所有 TODO 任务
     * 返回系统中所有 TODO 任务的列表。任务按创建时间排序（最新的在前）。
     * Example:
     *   Request: {}
     *   Response: { todos: [{ id: "1", title: "Task 1" }, ...] }
     * </pre>
     */
    public void listTodos(com.pingxin403.cuckoo.todo.v1.ListTodosRequest request,
        io.grpc.stub.StreamObserver<com.pingxin403.cuckoo.todo.v1.ListTodosResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getListTodosMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     * <pre>
     * UpdateTodo 更新现有的 TODO 任务
     * 更新指定 ID 的 TODO 任务。可以更新标题、描述和完成状态。
     * 更新操作会自动更新 updated_at 时间戳。
     * Example:
     *   Request: { id: "uuid", title: "Updated title", completed: true }
     *   Response: { todo: { id: "uuid", title: "Updated title", completed: true, ... } }
     * Errors:
     *   - NOT_FOUND: 如果指定的 TODO 不存在
     *   - INVALID_ARGUMENT: 如果 ID 为空或标题为空
     * </pre>
     */
    public void updateTodo(com.pingxin403.cuckoo.todo.v1.UpdateTodoRequest request,
        io.grpc.stub.StreamObserver<com.pingxin403.cuckoo.todo.v1.UpdateTodoResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getUpdateTodoMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     * <pre>
     * DeleteTodo 删除指定的 TODO 任务
     * 根据 ID 删除 TODO 任务。删除操作是幂等的，重复删除同一个 ID 不会报错。
     * Example:
     *   Request: { id: "uuid" }
     *   Response: { success: true }
     * Errors:
     *   - NOT_FOUND: 如果指定的 TODO 不存在
     *   - INVALID_ARGUMENT: 如果 ID 为空
     * </pre>
     */
    public void deleteTodo(com.pingxin403.cuckoo.todo.v1.DeleteTodoRequest request,
        io.grpc.stub.StreamObserver<com.pingxin403.cuckoo.todo.v1.DeleteTodoResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getDeleteTodoMethod(), getCallOptions()), request, responseObserver);
    }
  }

  /**
   * A stub to allow clients to do synchronous rpc calls to service TodoService.
   * <pre>
   * TodoService 提供 TODO 任务管理功能
   * 该服务支持完整的 CRUD 操作，允许用户创建、查询、更新和删除 TODO 任务。
   * 所有任务都有唯一的 ID，并记录创建和更新时间。
   * </pre>
   */
  public static final class TodoServiceBlockingV2Stub
      extends io.grpc.stub.AbstractBlockingStub<TodoServiceBlockingV2Stub> {
    private TodoServiceBlockingV2Stub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected TodoServiceBlockingV2Stub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new TodoServiceBlockingV2Stub(channel, callOptions);
    }

    /**
     * <pre>
     * CreateTodo 创建新的 TODO 任务
     * 创建一个新的 TODO 项并返回包含生成的唯一 ID 和时间戳的完整对象。
     * 标题字段是必需的，不能为空。
     * Example:
     *   Request: { title: "Buy groceries", description: "Milk, eggs, bread" }
     *   Response: { todo: { id: "uuid", title: "Buy groceries", ... } }
     * Errors:
     *   - INVALID_ARGUMENT: 如果标题为空或仅包含空格
     * </pre>
     */
    public com.pingxin403.cuckoo.todo.v1.CreateTodoResponse createTodo(com.pingxin403.cuckoo.todo.v1.CreateTodoRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getCreateTodoMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * ListTodos 获取所有 TODO 任务
     * 返回系统中所有 TODO 任务的列表。任务按创建时间排序（最新的在前）。
     * Example:
     *   Request: {}
     *   Response: { todos: [{ id: "1", title: "Task 1" }, ...] }
     * </pre>
     */
    public com.pingxin403.cuckoo.todo.v1.ListTodosResponse listTodos(com.pingxin403.cuckoo.todo.v1.ListTodosRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getListTodosMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * UpdateTodo 更新现有的 TODO 任务
     * 更新指定 ID 的 TODO 任务。可以更新标题、描述和完成状态。
     * 更新操作会自动更新 updated_at 时间戳。
     * Example:
     *   Request: { id: "uuid", title: "Updated title", completed: true }
     *   Response: { todo: { id: "uuid", title: "Updated title", completed: true, ... } }
     * Errors:
     *   - NOT_FOUND: 如果指定的 TODO 不存在
     *   - INVALID_ARGUMENT: 如果 ID 为空或标题为空
     * </pre>
     */
    public com.pingxin403.cuckoo.todo.v1.UpdateTodoResponse updateTodo(com.pingxin403.cuckoo.todo.v1.UpdateTodoRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getUpdateTodoMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * DeleteTodo 删除指定的 TODO 任务
     * 根据 ID 删除 TODO 任务。删除操作是幂等的，重复删除同一个 ID 不会报错。
     * Example:
     *   Request: { id: "uuid" }
     *   Response: { success: true }
     * Errors:
     *   - NOT_FOUND: 如果指定的 TODO 不存在
     *   - INVALID_ARGUMENT: 如果 ID 为空
     * </pre>
     */
    public com.pingxin403.cuckoo.todo.v1.DeleteTodoResponse deleteTodo(com.pingxin403.cuckoo.todo.v1.DeleteTodoRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getDeleteTodoMethod(), getCallOptions(), request);
    }
  }

  /**
   * A stub to allow clients to do limited synchronous rpc calls to service TodoService.
   * <pre>
   * TodoService 提供 TODO 任务管理功能
   * 该服务支持完整的 CRUD 操作，允许用户创建、查询、更新和删除 TODO 任务。
   * 所有任务都有唯一的 ID，并记录创建和更新时间。
   * </pre>
   */
  public static final class TodoServiceBlockingStub
      extends io.grpc.stub.AbstractBlockingStub<TodoServiceBlockingStub> {
    private TodoServiceBlockingStub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected TodoServiceBlockingStub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new TodoServiceBlockingStub(channel, callOptions);
    }

    /**
     * <pre>
     * CreateTodo 创建新的 TODO 任务
     * 创建一个新的 TODO 项并返回包含生成的唯一 ID 和时间戳的完整对象。
     * 标题字段是必需的，不能为空。
     * Example:
     *   Request: { title: "Buy groceries", description: "Milk, eggs, bread" }
     *   Response: { todo: { id: "uuid", title: "Buy groceries", ... } }
     * Errors:
     *   - INVALID_ARGUMENT: 如果标题为空或仅包含空格
     * </pre>
     */
    public com.pingxin403.cuckoo.todo.v1.CreateTodoResponse createTodo(com.pingxin403.cuckoo.todo.v1.CreateTodoRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getCreateTodoMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * ListTodos 获取所有 TODO 任务
     * 返回系统中所有 TODO 任务的列表。任务按创建时间排序（最新的在前）。
     * Example:
     *   Request: {}
     *   Response: { todos: [{ id: "1", title: "Task 1" }, ...] }
     * </pre>
     */
    public com.pingxin403.cuckoo.todo.v1.ListTodosResponse listTodos(com.pingxin403.cuckoo.todo.v1.ListTodosRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getListTodosMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * UpdateTodo 更新现有的 TODO 任务
     * 更新指定 ID 的 TODO 任务。可以更新标题、描述和完成状态。
     * 更新操作会自动更新 updated_at 时间戳。
     * Example:
     *   Request: { id: "uuid", title: "Updated title", completed: true }
     *   Response: { todo: { id: "uuid", title: "Updated title", completed: true, ... } }
     * Errors:
     *   - NOT_FOUND: 如果指定的 TODO 不存在
     *   - INVALID_ARGUMENT: 如果 ID 为空或标题为空
     * </pre>
     */
    public com.pingxin403.cuckoo.todo.v1.UpdateTodoResponse updateTodo(com.pingxin403.cuckoo.todo.v1.UpdateTodoRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getUpdateTodoMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * DeleteTodo 删除指定的 TODO 任务
     * 根据 ID 删除 TODO 任务。删除操作是幂等的，重复删除同一个 ID 不会报错。
     * Example:
     *   Request: { id: "uuid" }
     *   Response: { success: true }
     * Errors:
     *   - NOT_FOUND: 如果指定的 TODO 不存在
     *   - INVALID_ARGUMENT: 如果 ID 为空
     * </pre>
     */
    public com.pingxin403.cuckoo.todo.v1.DeleteTodoResponse deleteTodo(com.pingxin403.cuckoo.todo.v1.DeleteTodoRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getDeleteTodoMethod(), getCallOptions(), request);
    }
  }

  /**
   * A stub to allow clients to do ListenableFuture-style rpc calls to service TodoService.
   * <pre>
   * TodoService 提供 TODO 任务管理功能
   * 该服务支持完整的 CRUD 操作，允许用户创建、查询、更新和删除 TODO 任务。
   * 所有任务都有唯一的 ID，并记录创建和更新时间。
   * </pre>
   */
  public static final class TodoServiceFutureStub
      extends io.grpc.stub.AbstractFutureStub<TodoServiceFutureStub> {
    private TodoServiceFutureStub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected TodoServiceFutureStub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new TodoServiceFutureStub(channel, callOptions);
    }

    /**
     * <pre>
     * CreateTodo 创建新的 TODO 任务
     * 创建一个新的 TODO 项并返回包含生成的唯一 ID 和时间戳的完整对象。
     * 标题字段是必需的，不能为空。
     * Example:
     *   Request: { title: "Buy groceries", description: "Milk, eggs, bread" }
     *   Response: { todo: { id: "uuid", title: "Buy groceries", ... } }
     * Errors:
     *   - INVALID_ARGUMENT: 如果标题为空或仅包含空格
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<com.pingxin403.cuckoo.todo.v1.CreateTodoResponse> createTodo(
        com.pingxin403.cuckoo.todo.v1.CreateTodoRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getCreateTodoMethod(), getCallOptions()), request);
    }

    /**
     * <pre>
     * ListTodos 获取所有 TODO 任务
     * 返回系统中所有 TODO 任务的列表。任务按创建时间排序（最新的在前）。
     * Example:
     *   Request: {}
     *   Response: { todos: [{ id: "1", title: "Task 1" }, ...] }
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<com.pingxin403.cuckoo.todo.v1.ListTodosResponse> listTodos(
        com.pingxin403.cuckoo.todo.v1.ListTodosRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getListTodosMethod(), getCallOptions()), request);
    }

    /**
     * <pre>
     * UpdateTodo 更新现有的 TODO 任务
     * 更新指定 ID 的 TODO 任务。可以更新标题、描述和完成状态。
     * 更新操作会自动更新 updated_at 时间戳。
     * Example:
     *   Request: { id: "uuid", title: "Updated title", completed: true }
     *   Response: { todo: { id: "uuid", title: "Updated title", completed: true, ... } }
     * Errors:
     *   - NOT_FOUND: 如果指定的 TODO 不存在
     *   - INVALID_ARGUMENT: 如果 ID 为空或标题为空
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<com.pingxin403.cuckoo.todo.v1.UpdateTodoResponse> updateTodo(
        com.pingxin403.cuckoo.todo.v1.UpdateTodoRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getUpdateTodoMethod(), getCallOptions()), request);
    }

    /**
     * <pre>
     * DeleteTodo 删除指定的 TODO 任务
     * 根据 ID 删除 TODO 任务。删除操作是幂等的，重复删除同一个 ID 不会报错。
     * Example:
     *   Request: { id: "uuid" }
     *   Response: { success: true }
     * Errors:
     *   - NOT_FOUND: 如果指定的 TODO 不存在
     *   - INVALID_ARGUMENT: 如果 ID 为空
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<com.pingxin403.cuckoo.todo.v1.DeleteTodoResponse> deleteTodo(
        com.pingxin403.cuckoo.todo.v1.DeleteTodoRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getDeleteTodoMethod(), getCallOptions()), request);
    }
  }

  private static final int METHODID_CREATE_TODO = 0;
  private static final int METHODID_LIST_TODOS = 1;
  private static final int METHODID_UPDATE_TODO = 2;
  private static final int METHODID_DELETE_TODO = 3;

  private static final class MethodHandlers<Req, Resp> implements
      io.grpc.stub.ServerCalls.UnaryMethod<Req, Resp>,
      io.grpc.stub.ServerCalls.ServerStreamingMethod<Req, Resp>,
      io.grpc.stub.ServerCalls.ClientStreamingMethod<Req, Resp>,
      io.grpc.stub.ServerCalls.BidiStreamingMethod<Req, Resp> {
    private final AsyncService serviceImpl;
    private final int methodId;

    MethodHandlers(AsyncService serviceImpl, int methodId) {
      this.serviceImpl = serviceImpl;
      this.methodId = methodId;
    }

    @java.lang.Override
    @java.lang.SuppressWarnings("unchecked")
    public void invoke(Req request, io.grpc.stub.StreamObserver<Resp> responseObserver) {
      switch (methodId) {
        case METHODID_CREATE_TODO:
          serviceImpl.createTodo((com.pingxin403.cuckoo.todo.v1.CreateTodoRequest) request,
              (io.grpc.stub.StreamObserver<com.pingxin403.cuckoo.todo.v1.CreateTodoResponse>) responseObserver);
          break;
        case METHODID_LIST_TODOS:
          serviceImpl.listTodos((com.pingxin403.cuckoo.todo.v1.ListTodosRequest) request,
              (io.grpc.stub.StreamObserver<com.pingxin403.cuckoo.todo.v1.ListTodosResponse>) responseObserver);
          break;
        case METHODID_UPDATE_TODO:
          serviceImpl.updateTodo((com.pingxin403.cuckoo.todo.v1.UpdateTodoRequest) request,
              (io.grpc.stub.StreamObserver<com.pingxin403.cuckoo.todo.v1.UpdateTodoResponse>) responseObserver);
          break;
        case METHODID_DELETE_TODO:
          serviceImpl.deleteTodo((com.pingxin403.cuckoo.todo.v1.DeleteTodoRequest) request,
              (io.grpc.stub.StreamObserver<com.pingxin403.cuckoo.todo.v1.DeleteTodoResponse>) responseObserver);
          break;
        default:
          throw new AssertionError();
      }
    }

    @java.lang.Override
    @java.lang.SuppressWarnings("unchecked")
    public io.grpc.stub.StreamObserver<Req> invoke(
        io.grpc.stub.StreamObserver<Resp> responseObserver) {
      switch (methodId) {
        default:
          throw new AssertionError();
      }
    }
  }

  public static final io.grpc.ServerServiceDefinition bindService(AsyncService service) {
    return io.grpc.ServerServiceDefinition.builder(getServiceDescriptor())
        .addMethod(
          getCreateTodoMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              com.pingxin403.cuckoo.todo.v1.CreateTodoRequest,
              com.pingxin403.cuckoo.todo.v1.CreateTodoResponse>(
                service, METHODID_CREATE_TODO)))
        .addMethod(
          getListTodosMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              com.pingxin403.cuckoo.todo.v1.ListTodosRequest,
              com.pingxin403.cuckoo.todo.v1.ListTodosResponse>(
                service, METHODID_LIST_TODOS)))
        .addMethod(
          getUpdateTodoMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              com.pingxin403.cuckoo.todo.v1.UpdateTodoRequest,
              com.pingxin403.cuckoo.todo.v1.UpdateTodoResponse>(
                service, METHODID_UPDATE_TODO)))
        .addMethod(
          getDeleteTodoMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              com.pingxin403.cuckoo.todo.v1.DeleteTodoRequest,
              com.pingxin403.cuckoo.todo.v1.DeleteTodoResponse>(
                service, METHODID_DELETE_TODO)))
        .build();
  }

  private static abstract class TodoServiceBaseDescriptorSupplier
      implements io.grpc.protobuf.ProtoFileDescriptorSupplier, io.grpc.protobuf.ProtoServiceDescriptorSupplier {
    TodoServiceBaseDescriptorSupplier() {}

    @java.lang.Override
    public com.google.protobuf.Descriptors.FileDescriptor getFileDescriptor() {
      return com.pingxin403.cuckoo.todo.v1.TodoProto.getDescriptor();
    }

    @java.lang.Override
    public com.google.protobuf.Descriptors.ServiceDescriptor getServiceDescriptor() {
      return getFileDescriptor().findServiceByName("TodoService");
    }
  }

  private static final class TodoServiceFileDescriptorSupplier
      extends TodoServiceBaseDescriptorSupplier {
    TodoServiceFileDescriptorSupplier() {}
  }

  private static final class TodoServiceMethodDescriptorSupplier
      extends TodoServiceBaseDescriptorSupplier
      implements io.grpc.protobuf.ProtoMethodDescriptorSupplier {
    private final java.lang.String methodName;

    TodoServiceMethodDescriptorSupplier(java.lang.String methodName) {
      this.methodName = methodName;
    }

    @java.lang.Override
    public com.google.protobuf.Descriptors.MethodDescriptor getMethodDescriptor() {
      return getServiceDescriptor().findMethodByName(methodName);
    }
  }

  private static volatile io.grpc.ServiceDescriptor serviceDescriptor;

  public static io.grpc.ServiceDescriptor getServiceDescriptor() {
    io.grpc.ServiceDescriptor result = serviceDescriptor;
    if (result == null) {
      synchronized (TodoServiceGrpc.class) {
        result = serviceDescriptor;
        if (result == null) {
          serviceDescriptor = result = io.grpc.ServiceDescriptor.newBuilder(SERVICE_NAME)
              .setSchemaDescriptor(new TodoServiceFileDescriptorSupplier())
              .addMethod(getCreateTodoMethod())
              .addMethod(getListTodosMethod())
              .addMethod(getUpdateTodoMethod())
              .addMethod(getDeleteTodoMethod())
              .build();
        }
      }
    }
    return result;
  }
}
