package com.pingxin403.cuckoo.user.v1;

import static io.grpc.MethodDescriptor.generateFullMethodName;

/**
 * <pre>
 * UserService provides user profile and group membership management.
 * </pre>
 */
@io.grpc.stub.annotations.GrpcGenerated
public final class UserServiceGrpc {

  private UserServiceGrpc() {}

  public static final java.lang.String SERVICE_NAME = "user.v1.UserService";

  // Static method descriptors that strictly reflect the proto.
  private static volatile io.grpc.MethodDescriptor<com.pingxin403.cuckoo.user.v1.GetUserRequest,
      com.pingxin403.cuckoo.user.v1.GetUserResponse> getGetUserMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "GetUser",
      requestType = com.pingxin403.cuckoo.user.v1.GetUserRequest.class,
      responseType = com.pingxin403.cuckoo.user.v1.GetUserResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<com.pingxin403.cuckoo.user.v1.GetUserRequest,
      com.pingxin403.cuckoo.user.v1.GetUserResponse> getGetUserMethod() {
    io.grpc.MethodDescriptor<com.pingxin403.cuckoo.user.v1.GetUserRequest, com.pingxin403.cuckoo.user.v1.GetUserResponse> getGetUserMethod;
    if ((getGetUserMethod = UserServiceGrpc.getGetUserMethod) == null) {
      synchronized (UserServiceGrpc.class) {
        if ((getGetUserMethod = UserServiceGrpc.getGetUserMethod) == null) {
          UserServiceGrpc.getGetUserMethod = getGetUserMethod =
              io.grpc.MethodDescriptor.<com.pingxin403.cuckoo.user.v1.GetUserRequest, com.pingxin403.cuckoo.user.v1.GetUserResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "GetUser"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  com.pingxin403.cuckoo.user.v1.GetUserRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  com.pingxin403.cuckoo.user.v1.GetUserResponse.getDefaultInstance()))
              .setSchemaDescriptor(new UserServiceMethodDescriptorSupplier("GetUser"))
              .build();
        }
      }
    }
    return getGetUserMethod;
  }

  private static volatile io.grpc.MethodDescriptor<com.pingxin403.cuckoo.user.v1.BatchGetUsersRequest,
      com.pingxin403.cuckoo.user.v1.BatchGetUsersResponse> getBatchGetUsersMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "BatchGetUsers",
      requestType = com.pingxin403.cuckoo.user.v1.BatchGetUsersRequest.class,
      responseType = com.pingxin403.cuckoo.user.v1.BatchGetUsersResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<com.pingxin403.cuckoo.user.v1.BatchGetUsersRequest,
      com.pingxin403.cuckoo.user.v1.BatchGetUsersResponse> getBatchGetUsersMethod() {
    io.grpc.MethodDescriptor<com.pingxin403.cuckoo.user.v1.BatchGetUsersRequest, com.pingxin403.cuckoo.user.v1.BatchGetUsersResponse> getBatchGetUsersMethod;
    if ((getBatchGetUsersMethod = UserServiceGrpc.getBatchGetUsersMethod) == null) {
      synchronized (UserServiceGrpc.class) {
        if ((getBatchGetUsersMethod = UserServiceGrpc.getBatchGetUsersMethod) == null) {
          UserServiceGrpc.getBatchGetUsersMethod = getBatchGetUsersMethod =
              io.grpc.MethodDescriptor.<com.pingxin403.cuckoo.user.v1.BatchGetUsersRequest, com.pingxin403.cuckoo.user.v1.BatchGetUsersResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "BatchGetUsers"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  com.pingxin403.cuckoo.user.v1.BatchGetUsersRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  com.pingxin403.cuckoo.user.v1.BatchGetUsersResponse.getDefaultInstance()))
              .setSchemaDescriptor(new UserServiceMethodDescriptorSupplier("BatchGetUsers"))
              .build();
        }
      }
    }
    return getBatchGetUsersMethod;
  }

  private static volatile io.grpc.MethodDescriptor<com.pingxin403.cuckoo.user.v1.GetGroupMembersRequest,
      com.pingxin403.cuckoo.user.v1.GetGroupMembersResponse> getGetGroupMembersMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "GetGroupMembers",
      requestType = com.pingxin403.cuckoo.user.v1.GetGroupMembersRequest.class,
      responseType = com.pingxin403.cuckoo.user.v1.GetGroupMembersResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<com.pingxin403.cuckoo.user.v1.GetGroupMembersRequest,
      com.pingxin403.cuckoo.user.v1.GetGroupMembersResponse> getGetGroupMembersMethod() {
    io.grpc.MethodDescriptor<com.pingxin403.cuckoo.user.v1.GetGroupMembersRequest, com.pingxin403.cuckoo.user.v1.GetGroupMembersResponse> getGetGroupMembersMethod;
    if ((getGetGroupMembersMethod = UserServiceGrpc.getGetGroupMembersMethod) == null) {
      synchronized (UserServiceGrpc.class) {
        if ((getGetGroupMembersMethod = UserServiceGrpc.getGetGroupMembersMethod) == null) {
          UserServiceGrpc.getGetGroupMembersMethod = getGetGroupMembersMethod =
              io.grpc.MethodDescriptor.<com.pingxin403.cuckoo.user.v1.GetGroupMembersRequest, com.pingxin403.cuckoo.user.v1.GetGroupMembersResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "GetGroupMembers"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  com.pingxin403.cuckoo.user.v1.GetGroupMembersRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  com.pingxin403.cuckoo.user.v1.GetGroupMembersResponse.getDefaultInstance()))
              .setSchemaDescriptor(new UserServiceMethodDescriptorSupplier("GetGroupMembers"))
              .build();
        }
      }
    }
    return getGetGroupMembersMethod;
  }

  private static volatile io.grpc.MethodDescriptor<com.pingxin403.cuckoo.user.v1.ValidateGroupMembershipRequest,
      com.pingxin403.cuckoo.user.v1.ValidateGroupMembershipResponse> getValidateGroupMembershipMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "ValidateGroupMembership",
      requestType = com.pingxin403.cuckoo.user.v1.ValidateGroupMembershipRequest.class,
      responseType = com.pingxin403.cuckoo.user.v1.ValidateGroupMembershipResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<com.pingxin403.cuckoo.user.v1.ValidateGroupMembershipRequest,
      com.pingxin403.cuckoo.user.v1.ValidateGroupMembershipResponse> getValidateGroupMembershipMethod() {
    io.grpc.MethodDescriptor<com.pingxin403.cuckoo.user.v1.ValidateGroupMembershipRequest, com.pingxin403.cuckoo.user.v1.ValidateGroupMembershipResponse> getValidateGroupMembershipMethod;
    if ((getValidateGroupMembershipMethod = UserServiceGrpc.getValidateGroupMembershipMethod) == null) {
      synchronized (UserServiceGrpc.class) {
        if ((getValidateGroupMembershipMethod = UserServiceGrpc.getValidateGroupMembershipMethod) == null) {
          UserServiceGrpc.getValidateGroupMembershipMethod = getValidateGroupMembershipMethod =
              io.grpc.MethodDescriptor.<com.pingxin403.cuckoo.user.v1.ValidateGroupMembershipRequest, com.pingxin403.cuckoo.user.v1.ValidateGroupMembershipResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "ValidateGroupMembership"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  com.pingxin403.cuckoo.user.v1.ValidateGroupMembershipRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  com.pingxin403.cuckoo.user.v1.ValidateGroupMembershipResponse.getDefaultInstance()))
              .setSchemaDescriptor(new UserServiceMethodDescriptorSupplier("ValidateGroupMembership"))
              .build();
        }
      }
    }
    return getValidateGroupMembershipMethod;
  }

  /**
   * Creates a new async stub that supports all call types for the service
   */
  public static UserServiceStub newStub(io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<UserServiceStub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<UserServiceStub>() {
        @java.lang.Override
        public UserServiceStub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new UserServiceStub(channel, callOptions);
        }
      };
    return UserServiceStub.newStub(factory, channel);
  }

  /**
   * Creates a new blocking-style stub that supports all types of calls on the service
   */
  public static UserServiceBlockingV2Stub newBlockingV2Stub(
      io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<UserServiceBlockingV2Stub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<UserServiceBlockingV2Stub>() {
        @java.lang.Override
        public UserServiceBlockingV2Stub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new UserServiceBlockingV2Stub(channel, callOptions);
        }
      };
    return UserServiceBlockingV2Stub.newStub(factory, channel);
  }

  /**
   * Creates a new blocking-style stub that supports unary and streaming output calls on the service
   */
  public static UserServiceBlockingStub newBlockingStub(
      io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<UserServiceBlockingStub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<UserServiceBlockingStub>() {
        @java.lang.Override
        public UserServiceBlockingStub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new UserServiceBlockingStub(channel, callOptions);
        }
      };
    return UserServiceBlockingStub.newStub(factory, channel);
  }

  /**
   * Creates a new ListenableFuture-style stub that supports unary calls on the service
   */
  public static UserServiceFutureStub newFutureStub(
      io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<UserServiceFutureStub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<UserServiceFutureStub>() {
        @java.lang.Override
        public UserServiceFutureStub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new UserServiceFutureStub(channel, callOptions);
        }
      };
    return UserServiceFutureStub.newStub(factory, channel);
  }

  /**
   * <pre>
   * UserService provides user profile and group membership management.
   * </pre>
   */
  public interface AsyncService {

    /**
     * <pre>
     * GetUser retrieves a single user's profile information.
     * </pre>
     */
    default void getUser(com.pingxin403.cuckoo.user.v1.GetUserRequest request,
        io.grpc.stub.StreamObserver<com.pingxin403.cuckoo.user.v1.GetUserResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getGetUserMethod(), responseObserver);
    }

    /**
     * <pre>
     * BatchGetUsers retrieves multiple users' profiles in a single request.
     * </pre>
     */
    default void batchGetUsers(com.pingxin403.cuckoo.user.v1.BatchGetUsersRequest request,
        io.grpc.stub.StreamObserver<com.pingxin403.cuckoo.user.v1.BatchGetUsersResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getBatchGetUsersMethod(), responseObserver);
    }

    /**
     * <pre>
     * GetGroupMembers retrieves all members of a group with pagination support.
     * </pre>
     */
    default void getGroupMembers(com.pingxin403.cuckoo.user.v1.GetGroupMembersRequest request,
        io.grpc.stub.StreamObserver<com.pingxin403.cuckoo.user.v1.GetGroupMembersResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getGetGroupMembersMethod(), responseObserver);
    }

    /**
     * <pre>
     * ValidateGroupMembership checks if a user is a member of a specific group.
     * </pre>
     */
    default void validateGroupMembership(com.pingxin403.cuckoo.user.v1.ValidateGroupMembershipRequest request,
        io.grpc.stub.StreamObserver<com.pingxin403.cuckoo.user.v1.ValidateGroupMembershipResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getValidateGroupMembershipMethod(), responseObserver);
    }
  }

  /**
   * Base class for the server implementation of the service UserService.
   * <pre>
   * UserService provides user profile and group membership management.
   * </pre>
   */
  public static abstract class UserServiceImplBase
      implements io.grpc.BindableService, AsyncService {

    @java.lang.Override public final io.grpc.ServerServiceDefinition bindService() {
      return UserServiceGrpc.bindService(this);
    }
  }

  /**
   * A stub to allow clients to do asynchronous rpc calls to service UserService.
   * <pre>
   * UserService provides user profile and group membership management.
   * </pre>
   */
  public static final class UserServiceStub
      extends io.grpc.stub.AbstractAsyncStub<UserServiceStub> {
    private UserServiceStub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected UserServiceStub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new UserServiceStub(channel, callOptions);
    }

    /**
     * <pre>
     * GetUser retrieves a single user's profile information.
     * </pre>
     */
    public void getUser(com.pingxin403.cuckoo.user.v1.GetUserRequest request,
        io.grpc.stub.StreamObserver<com.pingxin403.cuckoo.user.v1.GetUserResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getGetUserMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     * <pre>
     * BatchGetUsers retrieves multiple users' profiles in a single request.
     * </pre>
     */
    public void batchGetUsers(com.pingxin403.cuckoo.user.v1.BatchGetUsersRequest request,
        io.grpc.stub.StreamObserver<com.pingxin403.cuckoo.user.v1.BatchGetUsersResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getBatchGetUsersMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     * <pre>
     * GetGroupMembers retrieves all members of a group with pagination support.
     * </pre>
     */
    public void getGroupMembers(com.pingxin403.cuckoo.user.v1.GetGroupMembersRequest request,
        io.grpc.stub.StreamObserver<com.pingxin403.cuckoo.user.v1.GetGroupMembersResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getGetGroupMembersMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     * <pre>
     * ValidateGroupMembership checks if a user is a member of a specific group.
     * </pre>
     */
    public void validateGroupMembership(com.pingxin403.cuckoo.user.v1.ValidateGroupMembershipRequest request,
        io.grpc.stub.StreamObserver<com.pingxin403.cuckoo.user.v1.ValidateGroupMembershipResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getValidateGroupMembershipMethod(), getCallOptions()), request, responseObserver);
    }
  }

  /**
   * A stub to allow clients to do synchronous rpc calls to service UserService.
   * <pre>
   * UserService provides user profile and group membership management.
   * </pre>
   */
  public static final class UserServiceBlockingV2Stub
      extends io.grpc.stub.AbstractBlockingStub<UserServiceBlockingV2Stub> {
    private UserServiceBlockingV2Stub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected UserServiceBlockingV2Stub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new UserServiceBlockingV2Stub(channel, callOptions);
    }

    /**
     * <pre>
     * GetUser retrieves a single user's profile information.
     * </pre>
     */
    public com.pingxin403.cuckoo.user.v1.GetUserResponse getUser(com.pingxin403.cuckoo.user.v1.GetUserRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getGetUserMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * BatchGetUsers retrieves multiple users' profiles in a single request.
     * </pre>
     */
    public com.pingxin403.cuckoo.user.v1.BatchGetUsersResponse batchGetUsers(com.pingxin403.cuckoo.user.v1.BatchGetUsersRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getBatchGetUsersMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * GetGroupMembers retrieves all members of a group with pagination support.
     * </pre>
     */
    public com.pingxin403.cuckoo.user.v1.GetGroupMembersResponse getGroupMembers(com.pingxin403.cuckoo.user.v1.GetGroupMembersRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getGetGroupMembersMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * ValidateGroupMembership checks if a user is a member of a specific group.
     * </pre>
     */
    public com.pingxin403.cuckoo.user.v1.ValidateGroupMembershipResponse validateGroupMembership(com.pingxin403.cuckoo.user.v1.ValidateGroupMembershipRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getValidateGroupMembershipMethod(), getCallOptions(), request);
    }
  }

  /**
   * A stub to allow clients to do limited synchronous rpc calls to service UserService.
   * <pre>
   * UserService provides user profile and group membership management.
   * </pre>
   */
  public static final class UserServiceBlockingStub
      extends io.grpc.stub.AbstractBlockingStub<UserServiceBlockingStub> {
    private UserServiceBlockingStub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected UserServiceBlockingStub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new UserServiceBlockingStub(channel, callOptions);
    }

    /**
     * <pre>
     * GetUser retrieves a single user's profile information.
     * </pre>
     */
    public com.pingxin403.cuckoo.user.v1.GetUserResponse getUser(com.pingxin403.cuckoo.user.v1.GetUserRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getGetUserMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * BatchGetUsers retrieves multiple users' profiles in a single request.
     * </pre>
     */
    public com.pingxin403.cuckoo.user.v1.BatchGetUsersResponse batchGetUsers(com.pingxin403.cuckoo.user.v1.BatchGetUsersRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getBatchGetUsersMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * GetGroupMembers retrieves all members of a group with pagination support.
     * </pre>
     */
    public com.pingxin403.cuckoo.user.v1.GetGroupMembersResponse getGroupMembers(com.pingxin403.cuckoo.user.v1.GetGroupMembersRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getGetGroupMembersMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * ValidateGroupMembership checks if a user is a member of a specific group.
     * </pre>
     */
    public com.pingxin403.cuckoo.user.v1.ValidateGroupMembershipResponse validateGroupMembership(com.pingxin403.cuckoo.user.v1.ValidateGroupMembershipRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getValidateGroupMembershipMethod(), getCallOptions(), request);
    }
  }

  /**
   * A stub to allow clients to do ListenableFuture-style rpc calls to service UserService.
   * <pre>
   * UserService provides user profile and group membership management.
   * </pre>
   */
  public static final class UserServiceFutureStub
      extends io.grpc.stub.AbstractFutureStub<UserServiceFutureStub> {
    private UserServiceFutureStub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected UserServiceFutureStub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new UserServiceFutureStub(channel, callOptions);
    }

    /**
     * <pre>
     * GetUser retrieves a single user's profile information.
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<com.pingxin403.cuckoo.user.v1.GetUserResponse> getUser(
        com.pingxin403.cuckoo.user.v1.GetUserRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getGetUserMethod(), getCallOptions()), request);
    }

    /**
     * <pre>
     * BatchGetUsers retrieves multiple users' profiles in a single request.
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<com.pingxin403.cuckoo.user.v1.BatchGetUsersResponse> batchGetUsers(
        com.pingxin403.cuckoo.user.v1.BatchGetUsersRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getBatchGetUsersMethod(), getCallOptions()), request);
    }

    /**
     * <pre>
     * GetGroupMembers retrieves all members of a group with pagination support.
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<com.pingxin403.cuckoo.user.v1.GetGroupMembersResponse> getGroupMembers(
        com.pingxin403.cuckoo.user.v1.GetGroupMembersRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getGetGroupMembersMethod(), getCallOptions()), request);
    }

    /**
     * <pre>
     * ValidateGroupMembership checks if a user is a member of a specific group.
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<com.pingxin403.cuckoo.user.v1.ValidateGroupMembershipResponse> validateGroupMembership(
        com.pingxin403.cuckoo.user.v1.ValidateGroupMembershipRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getValidateGroupMembershipMethod(), getCallOptions()), request);
    }
  }

  private static final int METHODID_GET_USER = 0;
  private static final int METHODID_BATCH_GET_USERS = 1;
  private static final int METHODID_GET_GROUP_MEMBERS = 2;
  private static final int METHODID_VALIDATE_GROUP_MEMBERSHIP = 3;

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
        case METHODID_GET_USER:
          serviceImpl.getUser((com.pingxin403.cuckoo.user.v1.GetUserRequest) request,
              (io.grpc.stub.StreamObserver<com.pingxin403.cuckoo.user.v1.GetUserResponse>) responseObserver);
          break;
        case METHODID_BATCH_GET_USERS:
          serviceImpl.batchGetUsers((com.pingxin403.cuckoo.user.v1.BatchGetUsersRequest) request,
              (io.grpc.stub.StreamObserver<com.pingxin403.cuckoo.user.v1.BatchGetUsersResponse>) responseObserver);
          break;
        case METHODID_GET_GROUP_MEMBERS:
          serviceImpl.getGroupMembers((com.pingxin403.cuckoo.user.v1.GetGroupMembersRequest) request,
              (io.grpc.stub.StreamObserver<com.pingxin403.cuckoo.user.v1.GetGroupMembersResponse>) responseObserver);
          break;
        case METHODID_VALIDATE_GROUP_MEMBERSHIP:
          serviceImpl.validateGroupMembership((com.pingxin403.cuckoo.user.v1.ValidateGroupMembershipRequest) request,
              (io.grpc.stub.StreamObserver<com.pingxin403.cuckoo.user.v1.ValidateGroupMembershipResponse>) responseObserver);
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
          getGetUserMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              com.pingxin403.cuckoo.user.v1.GetUserRequest,
              com.pingxin403.cuckoo.user.v1.GetUserResponse>(
                service, METHODID_GET_USER)))
        .addMethod(
          getBatchGetUsersMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              com.pingxin403.cuckoo.user.v1.BatchGetUsersRequest,
              com.pingxin403.cuckoo.user.v1.BatchGetUsersResponse>(
                service, METHODID_BATCH_GET_USERS)))
        .addMethod(
          getGetGroupMembersMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              com.pingxin403.cuckoo.user.v1.GetGroupMembersRequest,
              com.pingxin403.cuckoo.user.v1.GetGroupMembersResponse>(
                service, METHODID_GET_GROUP_MEMBERS)))
        .addMethod(
          getValidateGroupMembershipMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              com.pingxin403.cuckoo.user.v1.ValidateGroupMembershipRequest,
              com.pingxin403.cuckoo.user.v1.ValidateGroupMembershipResponse>(
                service, METHODID_VALIDATE_GROUP_MEMBERSHIP)))
        .build();
  }

  private static abstract class UserServiceBaseDescriptorSupplier
      implements io.grpc.protobuf.ProtoFileDescriptorSupplier, io.grpc.protobuf.ProtoServiceDescriptorSupplier {
    UserServiceBaseDescriptorSupplier() {}

    @java.lang.Override
    public com.google.protobuf.Descriptors.FileDescriptor getFileDescriptor() {
      return com.pingxin403.cuckoo.user.v1.User.getDescriptor();
    }

    @java.lang.Override
    public com.google.protobuf.Descriptors.ServiceDescriptor getServiceDescriptor() {
      return getFileDescriptor().findServiceByName("UserService");
    }
  }

  private static final class UserServiceFileDescriptorSupplier
      extends UserServiceBaseDescriptorSupplier {
    UserServiceFileDescriptorSupplier() {}
  }

  private static final class UserServiceMethodDescriptorSupplier
      extends UserServiceBaseDescriptorSupplier
      implements io.grpc.protobuf.ProtoMethodDescriptorSupplier {
    private final java.lang.String methodName;

    UserServiceMethodDescriptorSupplier(java.lang.String methodName) {
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
      synchronized (UserServiceGrpc.class) {
        result = serviceDescriptor;
        if (result == null) {
          serviceDescriptor = result = io.grpc.ServiceDescriptor.newBuilder(SERVICE_NAME)
              .setSchemaDescriptor(new UserServiceFileDescriptorSupplier())
              .addMethod(getGetUserMethod())
              .addMethod(getBatchGetUsersMethod())
              .addMethod(getGetGroupMembersMethod())
              .addMethod(getValidateGroupMembershipMethod())
              .build();
        }
      }
    }
    return result;
  }
}
