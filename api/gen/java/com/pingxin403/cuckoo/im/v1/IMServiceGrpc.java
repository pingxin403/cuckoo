package com.pingxin403.cuckoo.im.v1;

import static io.grpc.MethodDescriptor.generateFullMethodName;

/**
 * <pre>
 * IMService provides message routing and delivery for the IM system.
 * It handles both private and group message routing with sequence number assignment.
 * </pre>
 */
@io.grpc.stub.annotations.GrpcGenerated
public final class IMServiceGrpc {

  private IMServiceGrpc() {}

  public static final java.lang.String SERVICE_NAME = "im.v1.IMService";

  // Static method descriptors that strictly reflect the proto.
  private static volatile io.grpc.MethodDescriptor<com.pingxin403.cuckoo.im.v1.RoutePrivateMessageRequest,
      com.pingxin403.cuckoo.im.v1.RoutePrivateMessageResponse> getRoutePrivateMessageMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "RoutePrivateMessage",
      requestType = com.pingxin403.cuckoo.im.v1.RoutePrivateMessageRequest.class,
      responseType = com.pingxin403.cuckoo.im.v1.RoutePrivateMessageResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<com.pingxin403.cuckoo.im.v1.RoutePrivateMessageRequest,
      com.pingxin403.cuckoo.im.v1.RoutePrivateMessageResponse> getRoutePrivateMessageMethod() {
    io.grpc.MethodDescriptor<com.pingxin403.cuckoo.im.v1.RoutePrivateMessageRequest, com.pingxin403.cuckoo.im.v1.RoutePrivateMessageResponse> getRoutePrivateMessageMethod;
    if ((getRoutePrivateMessageMethod = IMServiceGrpc.getRoutePrivateMessageMethod) == null) {
      synchronized (IMServiceGrpc.class) {
        if ((getRoutePrivateMessageMethod = IMServiceGrpc.getRoutePrivateMessageMethod) == null) {
          IMServiceGrpc.getRoutePrivateMessageMethod = getRoutePrivateMessageMethod =
              io.grpc.MethodDescriptor.<com.pingxin403.cuckoo.im.v1.RoutePrivateMessageRequest, com.pingxin403.cuckoo.im.v1.RoutePrivateMessageResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "RoutePrivateMessage"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  com.pingxin403.cuckoo.im.v1.RoutePrivateMessageRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  com.pingxin403.cuckoo.im.v1.RoutePrivateMessageResponse.getDefaultInstance()))
              .setSchemaDescriptor(new IMServiceMethodDescriptorSupplier("RoutePrivateMessage"))
              .build();
        }
      }
    }
    return getRoutePrivateMessageMethod;
  }

  private static volatile io.grpc.MethodDescriptor<com.pingxin403.cuckoo.im.v1.RouteGroupMessageRequest,
      com.pingxin403.cuckoo.im.v1.RouteGroupMessageResponse> getRouteGroupMessageMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "RouteGroupMessage",
      requestType = com.pingxin403.cuckoo.im.v1.RouteGroupMessageRequest.class,
      responseType = com.pingxin403.cuckoo.im.v1.RouteGroupMessageResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<com.pingxin403.cuckoo.im.v1.RouteGroupMessageRequest,
      com.pingxin403.cuckoo.im.v1.RouteGroupMessageResponse> getRouteGroupMessageMethod() {
    io.grpc.MethodDescriptor<com.pingxin403.cuckoo.im.v1.RouteGroupMessageRequest, com.pingxin403.cuckoo.im.v1.RouteGroupMessageResponse> getRouteGroupMessageMethod;
    if ((getRouteGroupMessageMethod = IMServiceGrpc.getRouteGroupMessageMethod) == null) {
      synchronized (IMServiceGrpc.class) {
        if ((getRouteGroupMessageMethod = IMServiceGrpc.getRouteGroupMessageMethod) == null) {
          IMServiceGrpc.getRouteGroupMessageMethod = getRouteGroupMessageMethod =
              io.grpc.MethodDescriptor.<com.pingxin403.cuckoo.im.v1.RouteGroupMessageRequest, com.pingxin403.cuckoo.im.v1.RouteGroupMessageResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "RouteGroupMessage"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  com.pingxin403.cuckoo.im.v1.RouteGroupMessageRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  com.pingxin403.cuckoo.im.v1.RouteGroupMessageResponse.getDefaultInstance()))
              .setSchemaDescriptor(new IMServiceMethodDescriptorSupplier("RouteGroupMessage"))
              .build();
        }
      }
    }
    return getRouteGroupMessageMethod;
  }

  private static volatile io.grpc.MethodDescriptor<com.pingxin403.cuckoo.im.v1.GetMessageStatusRequest,
      com.pingxin403.cuckoo.im.v1.GetMessageStatusResponse> getGetMessageStatusMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "GetMessageStatus",
      requestType = com.pingxin403.cuckoo.im.v1.GetMessageStatusRequest.class,
      responseType = com.pingxin403.cuckoo.im.v1.GetMessageStatusResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<com.pingxin403.cuckoo.im.v1.GetMessageStatusRequest,
      com.pingxin403.cuckoo.im.v1.GetMessageStatusResponse> getGetMessageStatusMethod() {
    io.grpc.MethodDescriptor<com.pingxin403.cuckoo.im.v1.GetMessageStatusRequest, com.pingxin403.cuckoo.im.v1.GetMessageStatusResponse> getGetMessageStatusMethod;
    if ((getGetMessageStatusMethod = IMServiceGrpc.getGetMessageStatusMethod) == null) {
      synchronized (IMServiceGrpc.class) {
        if ((getGetMessageStatusMethod = IMServiceGrpc.getGetMessageStatusMethod) == null) {
          IMServiceGrpc.getGetMessageStatusMethod = getGetMessageStatusMethod =
              io.grpc.MethodDescriptor.<com.pingxin403.cuckoo.im.v1.GetMessageStatusRequest, com.pingxin403.cuckoo.im.v1.GetMessageStatusResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "GetMessageStatus"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  com.pingxin403.cuckoo.im.v1.GetMessageStatusRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  com.pingxin403.cuckoo.im.v1.GetMessageStatusResponse.getDefaultInstance()))
              .setSchemaDescriptor(new IMServiceMethodDescriptorSupplier("GetMessageStatus"))
              .build();
        }
      }
    }
    return getGetMessageStatusMethod;
  }

  /**
   * Creates a new async stub that supports all call types for the service
   */
  public static IMServiceStub newStub(io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<IMServiceStub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<IMServiceStub>() {
        @java.lang.Override
        public IMServiceStub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new IMServiceStub(channel, callOptions);
        }
      };
    return IMServiceStub.newStub(factory, channel);
  }

  /**
   * Creates a new blocking-style stub that supports all types of calls on the service
   */
  public static IMServiceBlockingV2Stub newBlockingV2Stub(
      io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<IMServiceBlockingV2Stub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<IMServiceBlockingV2Stub>() {
        @java.lang.Override
        public IMServiceBlockingV2Stub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new IMServiceBlockingV2Stub(channel, callOptions);
        }
      };
    return IMServiceBlockingV2Stub.newStub(factory, channel);
  }

  /**
   * Creates a new blocking-style stub that supports unary and streaming output calls on the service
   */
  public static IMServiceBlockingStub newBlockingStub(
      io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<IMServiceBlockingStub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<IMServiceBlockingStub>() {
        @java.lang.Override
        public IMServiceBlockingStub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new IMServiceBlockingStub(channel, callOptions);
        }
      };
    return IMServiceBlockingStub.newStub(factory, channel);
  }

  /**
   * Creates a new ListenableFuture-style stub that supports unary calls on the service
   */
  public static IMServiceFutureStub newFutureStub(
      io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<IMServiceFutureStub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<IMServiceFutureStub>() {
        @java.lang.Override
        public IMServiceFutureStub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new IMServiceFutureStub(channel, callOptions);
        }
      };
    return IMServiceFutureStub.newStub(factory, channel);
  }

  /**
   * <pre>
   * IMService provides message routing and delivery for the IM system.
   * It handles both private and group message routing with sequence number assignment.
   * </pre>
   */
  public interface AsyncService {

    /**
     * <pre>
     * RoutePrivateMessage routes a private message to a specific user.
     * Assigns sequence number and routes to Fast Path (online) or Slow Path (offline).
     * </pre>
     */
    default void routePrivateMessage(com.pingxin403.cuckoo.im.v1.RoutePrivateMessageRequest request,
        io.grpc.stub.StreamObserver<com.pingxin403.cuckoo.im.v1.RoutePrivateMessageResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getRoutePrivateMessageMethod(), responseObserver);
    }

    /**
     * <pre>
     * RouteGroupMessage routes a message to all members of a group.
     * Assigns sequence number and publishes to Kafka for broadcast.
     * </pre>
     */
    default void routeGroupMessage(com.pingxin403.cuckoo.im.v1.RouteGroupMessageRequest request,
        io.grpc.stub.StreamObserver<com.pingxin403.cuckoo.im.v1.RouteGroupMessageResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getRouteGroupMessageMethod(), responseObserver);
    }

    /**
     * <pre>
     * GetMessageStatus retrieves the delivery status of a message.
     * </pre>
     */
    default void getMessageStatus(com.pingxin403.cuckoo.im.v1.GetMessageStatusRequest request,
        io.grpc.stub.StreamObserver<com.pingxin403.cuckoo.im.v1.GetMessageStatusResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getGetMessageStatusMethod(), responseObserver);
    }
  }

  /**
   * Base class for the server implementation of the service IMService.
   * <pre>
   * IMService provides message routing and delivery for the IM system.
   * It handles both private and group message routing with sequence number assignment.
   * </pre>
   */
  public static abstract class IMServiceImplBase
      implements io.grpc.BindableService, AsyncService {

    @java.lang.Override public final io.grpc.ServerServiceDefinition bindService() {
      return IMServiceGrpc.bindService(this);
    }
  }

  /**
   * A stub to allow clients to do asynchronous rpc calls to service IMService.
   * <pre>
   * IMService provides message routing and delivery for the IM system.
   * It handles both private and group message routing with sequence number assignment.
   * </pre>
   */
  public static final class IMServiceStub
      extends io.grpc.stub.AbstractAsyncStub<IMServiceStub> {
    private IMServiceStub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected IMServiceStub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new IMServiceStub(channel, callOptions);
    }

    /**
     * <pre>
     * RoutePrivateMessage routes a private message to a specific user.
     * Assigns sequence number and routes to Fast Path (online) or Slow Path (offline).
     * </pre>
     */
    public void routePrivateMessage(com.pingxin403.cuckoo.im.v1.RoutePrivateMessageRequest request,
        io.grpc.stub.StreamObserver<com.pingxin403.cuckoo.im.v1.RoutePrivateMessageResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getRoutePrivateMessageMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     * <pre>
     * RouteGroupMessage routes a message to all members of a group.
     * Assigns sequence number and publishes to Kafka for broadcast.
     * </pre>
     */
    public void routeGroupMessage(com.pingxin403.cuckoo.im.v1.RouteGroupMessageRequest request,
        io.grpc.stub.StreamObserver<com.pingxin403.cuckoo.im.v1.RouteGroupMessageResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getRouteGroupMessageMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     * <pre>
     * GetMessageStatus retrieves the delivery status of a message.
     * </pre>
     */
    public void getMessageStatus(com.pingxin403.cuckoo.im.v1.GetMessageStatusRequest request,
        io.grpc.stub.StreamObserver<com.pingxin403.cuckoo.im.v1.GetMessageStatusResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getGetMessageStatusMethod(), getCallOptions()), request, responseObserver);
    }
  }

  /**
   * A stub to allow clients to do synchronous rpc calls to service IMService.
   * <pre>
   * IMService provides message routing and delivery for the IM system.
   * It handles both private and group message routing with sequence number assignment.
   * </pre>
   */
  public static final class IMServiceBlockingV2Stub
      extends io.grpc.stub.AbstractBlockingStub<IMServiceBlockingV2Stub> {
    private IMServiceBlockingV2Stub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected IMServiceBlockingV2Stub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new IMServiceBlockingV2Stub(channel, callOptions);
    }

    /**
     * <pre>
     * RoutePrivateMessage routes a private message to a specific user.
     * Assigns sequence number and routes to Fast Path (online) or Slow Path (offline).
     * </pre>
     */
    public com.pingxin403.cuckoo.im.v1.RoutePrivateMessageResponse routePrivateMessage(com.pingxin403.cuckoo.im.v1.RoutePrivateMessageRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getRoutePrivateMessageMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * RouteGroupMessage routes a message to all members of a group.
     * Assigns sequence number and publishes to Kafka for broadcast.
     * </pre>
     */
    public com.pingxin403.cuckoo.im.v1.RouteGroupMessageResponse routeGroupMessage(com.pingxin403.cuckoo.im.v1.RouteGroupMessageRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getRouteGroupMessageMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * GetMessageStatus retrieves the delivery status of a message.
     * </pre>
     */
    public com.pingxin403.cuckoo.im.v1.GetMessageStatusResponse getMessageStatus(com.pingxin403.cuckoo.im.v1.GetMessageStatusRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getGetMessageStatusMethod(), getCallOptions(), request);
    }
  }

  /**
   * A stub to allow clients to do limited synchronous rpc calls to service IMService.
   * <pre>
   * IMService provides message routing and delivery for the IM system.
   * It handles both private and group message routing with sequence number assignment.
   * </pre>
   */
  public static final class IMServiceBlockingStub
      extends io.grpc.stub.AbstractBlockingStub<IMServiceBlockingStub> {
    private IMServiceBlockingStub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected IMServiceBlockingStub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new IMServiceBlockingStub(channel, callOptions);
    }

    /**
     * <pre>
     * RoutePrivateMessage routes a private message to a specific user.
     * Assigns sequence number and routes to Fast Path (online) or Slow Path (offline).
     * </pre>
     */
    public com.pingxin403.cuckoo.im.v1.RoutePrivateMessageResponse routePrivateMessage(com.pingxin403.cuckoo.im.v1.RoutePrivateMessageRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getRoutePrivateMessageMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * RouteGroupMessage routes a message to all members of a group.
     * Assigns sequence number and publishes to Kafka for broadcast.
     * </pre>
     */
    public com.pingxin403.cuckoo.im.v1.RouteGroupMessageResponse routeGroupMessage(com.pingxin403.cuckoo.im.v1.RouteGroupMessageRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getRouteGroupMessageMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * GetMessageStatus retrieves the delivery status of a message.
     * </pre>
     */
    public com.pingxin403.cuckoo.im.v1.GetMessageStatusResponse getMessageStatus(com.pingxin403.cuckoo.im.v1.GetMessageStatusRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getGetMessageStatusMethod(), getCallOptions(), request);
    }
  }

  /**
   * A stub to allow clients to do ListenableFuture-style rpc calls to service IMService.
   * <pre>
   * IMService provides message routing and delivery for the IM system.
   * It handles both private and group message routing with sequence number assignment.
   * </pre>
   */
  public static final class IMServiceFutureStub
      extends io.grpc.stub.AbstractFutureStub<IMServiceFutureStub> {
    private IMServiceFutureStub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected IMServiceFutureStub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new IMServiceFutureStub(channel, callOptions);
    }

    /**
     * <pre>
     * RoutePrivateMessage routes a private message to a specific user.
     * Assigns sequence number and routes to Fast Path (online) or Slow Path (offline).
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<com.pingxin403.cuckoo.im.v1.RoutePrivateMessageResponse> routePrivateMessage(
        com.pingxin403.cuckoo.im.v1.RoutePrivateMessageRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getRoutePrivateMessageMethod(), getCallOptions()), request);
    }

    /**
     * <pre>
     * RouteGroupMessage routes a message to all members of a group.
     * Assigns sequence number and publishes to Kafka for broadcast.
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<com.pingxin403.cuckoo.im.v1.RouteGroupMessageResponse> routeGroupMessage(
        com.pingxin403.cuckoo.im.v1.RouteGroupMessageRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getRouteGroupMessageMethod(), getCallOptions()), request);
    }

    /**
     * <pre>
     * GetMessageStatus retrieves the delivery status of a message.
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<com.pingxin403.cuckoo.im.v1.GetMessageStatusResponse> getMessageStatus(
        com.pingxin403.cuckoo.im.v1.GetMessageStatusRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getGetMessageStatusMethod(), getCallOptions()), request);
    }
  }

  private static final int METHODID_ROUTE_PRIVATE_MESSAGE = 0;
  private static final int METHODID_ROUTE_GROUP_MESSAGE = 1;
  private static final int METHODID_GET_MESSAGE_STATUS = 2;

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
        case METHODID_ROUTE_PRIVATE_MESSAGE:
          serviceImpl.routePrivateMessage((com.pingxin403.cuckoo.im.v1.RoutePrivateMessageRequest) request,
              (io.grpc.stub.StreamObserver<com.pingxin403.cuckoo.im.v1.RoutePrivateMessageResponse>) responseObserver);
          break;
        case METHODID_ROUTE_GROUP_MESSAGE:
          serviceImpl.routeGroupMessage((com.pingxin403.cuckoo.im.v1.RouteGroupMessageRequest) request,
              (io.grpc.stub.StreamObserver<com.pingxin403.cuckoo.im.v1.RouteGroupMessageResponse>) responseObserver);
          break;
        case METHODID_GET_MESSAGE_STATUS:
          serviceImpl.getMessageStatus((com.pingxin403.cuckoo.im.v1.GetMessageStatusRequest) request,
              (io.grpc.stub.StreamObserver<com.pingxin403.cuckoo.im.v1.GetMessageStatusResponse>) responseObserver);
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
          getRoutePrivateMessageMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              com.pingxin403.cuckoo.im.v1.RoutePrivateMessageRequest,
              com.pingxin403.cuckoo.im.v1.RoutePrivateMessageResponse>(
                service, METHODID_ROUTE_PRIVATE_MESSAGE)))
        .addMethod(
          getRouteGroupMessageMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              com.pingxin403.cuckoo.im.v1.RouteGroupMessageRequest,
              com.pingxin403.cuckoo.im.v1.RouteGroupMessageResponse>(
                service, METHODID_ROUTE_GROUP_MESSAGE)))
        .addMethod(
          getGetMessageStatusMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              com.pingxin403.cuckoo.im.v1.GetMessageStatusRequest,
              com.pingxin403.cuckoo.im.v1.GetMessageStatusResponse>(
                service, METHODID_GET_MESSAGE_STATUS)))
        .build();
  }

  private static abstract class IMServiceBaseDescriptorSupplier
      implements io.grpc.protobuf.ProtoFileDescriptorSupplier, io.grpc.protobuf.ProtoServiceDescriptorSupplier {
    IMServiceBaseDescriptorSupplier() {}

    @java.lang.Override
    public com.google.protobuf.Descriptors.FileDescriptor getFileDescriptor() {
      return com.pingxin403.cuckoo.im.v1.Im.getDescriptor();
    }

    @java.lang.Override
    public com.google.protobuf.Descriptors.ServiceDescriptor getServiceDescriptor() {
      return getFileDescriptor().findServiceByName("IMService");
    }
  }

  private static final class IMServiceFileDescriptorSupplier
      extends IMServiceBaseDescriptorSupplier {
    IMServiceFileDescriptorSupplier() {}
  }

  private static final class IMServiceMethodDescriptorSupplier
      extends IMServiceBaseDescriptorSupplier
      implements io.grpc.protobuf.ProtoMethodDescriptorSupplier {
    private final java.lang.String methodName;

    IMServiceMethodDescriptorSupplier(java.lang.String methodName) {
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
      synchronized (IMServiceGrpc.class) {
        result = serviceDescriptor;
        if (result == null) {
          serviceDescriptor = result = io.grpc.ServiceDescriptor.newBuilder(SERVICE_NAME)
              .setSchemaDescriptor(new IMServiceFileDescriptorSupplier())
              .addMethod(getRoutePrivateMessageMethod())
              .addMethod(getRouteGroupMessageMethod())
              .addMethod(getGetMessageStatusMethod())
              .build();
        }
      }
    }
    return result;
  }
}
