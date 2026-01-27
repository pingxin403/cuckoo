package com.pingxin403.cuckoo.im.gateway;

import static io.grpc.MethodDescriptor.generateFullMethodName;

/**
 * <pre>
 * im-gateway-service service
 * </pre>
 */
@io.grpc.stub.annotations.GrpcGenerated
public final class UimUgatewayUserviceServiceGrpc {

  private UimUgatewayUserviceServiceGrpc() {}

  public static final java.lang.String SERVICE_NAME = "im_gateway_servicepb.UimUgatewayUserviceService";

  // Static method descriptors that strictly reflect the proto.
  private static volatile io.grpc.MethodDescriptor<com.pingxin403.cuckoo.im.gateway.HealthCheckRequest,
      com.pingxin403.cuckoo.im.gateway.HealthCheckResponse> getHealthCheckMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "HealthCheck",
      requestType = com.pingxin403.cuckoo.im.gateway.HealthCheckRequest.class,
      responseType = com.pingxin403.cuckoo.im.gateway.HealthCheckResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<com.pingxin403.cuckoo.im.gateway.HealthCheckRequest,
      com.pingxin403.cuckoo.im.gateway.HealthCheckResponse> getHealthCheckMethod() {
    io.grpc.MethodDescriptor<com.pingxin403.cuckoo.im.gateway.HealthCheckRequest, com.pingxin403.cuckoo.im.gateway.HealthCheckResponse> getHealthCheckMethod;
    if ((getHealthCheckMethod = UimUgatewayUserviceServiceGrpc.getHealthCheckMethod) == null) {
      synchronized (UimUgatewayUserviceServiceGrpc.class) {
        if ((getHealthCheckMethod = UimUgatewayUserviceServiceGrpc.getHealthCheckMethod) == null) {
          UimUgatewayUserviceServiceGrpc.getHealthCheckMethod = getHealthCheckMethod =
              io.grpc.MethodDescriptor.<com.pingxin403.cuckoo.im.gateway.HealthCheckRequest, com.pingxin403.cuckoo.im.gateway.HealthCheckResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "HealthCheck"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  com.pingxin403.cuckoo.im.gateway.HealthCheckRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  com.pingxin403.cuckoo.im.gateway.HealthCheckResponse.getDefaultInstance()))
              .setSchemaDescriptor(new UimUgatewayUserviceServiceMethodDescriptorSupplier("HealthCheck"))
              .build();
        }
      }
    }
    return getHealthCheckMethod;
  }

  /**
   * Creates a new async stub that supports all call types for the service
   */
  public static UimUgatewayUserviceServiceStub newStub(io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<UimUgatewayUserviceServiceStub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<UimUgatewayUserviceServiceStub>() {
        @java.lang.Override
        public UimUgatewayUserviceServiceStub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new UimUgatewayUserviceServiceStub(channel, callOptions);
        }
      };
    return UimUgatewayUserviceServiceStub.newStub(factory, channel);
  }

  /**
   * Creates a new blocking-style stub that supports all types of calls on the service
   */
  public static UimUgatewayUserviceServiceBlockingV2Stub newBlockingV2Stub(
      io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<UimUgatewayUserviceServiceBlockingV2Stub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<UimUgatewayUserviceServiceBlockingV2Stub>() {
        @java.lang.Override
        public UimUgatewayUserviceServiceBlockingV2Stub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new UimUgatewayUserviceServiceBlockingV2Stub(channel, callOptions);
        }
      };
    return UimUgatewayUserviceServiceBlockingV2Stub.newStub(factory, channel);
  }

  /**
   * Creates a new blocking-style stub that supports unary and streaming output calls on the service
   */
  public static UimUgatewayUserviceServiceBlockingStub newBlockingStub(
      io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<UimUgatewayUserviceServiceBlockingStub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<UimUgatewayUserviceServiceBlockingStub>() {
        @java.lang.Override
        public UimUgatewayUserviceServiceBlockingStub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new UimUgatewayUserviceServiceBlockingStub(channel, callOptions);
        }
      };
    return UimUgatewayUserviceServiceBlockingStub.newStub(factory, channel);
  }

  /**
   * Creates a new ListenableFuture-style stub that supports unary calls on the service
   */
  public static UimUgatewayUserviceServiceFutureStub newFutureStub(
      io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<UimUgatewayUserviceServiceFutureStub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<UimUgatewayUserviceServiceFutureStub>() {
        @java.lang.Override
        public UimUgatewayUserviceServiceFutureStub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new UimUgatewayUserviceServiceFutureStub(channel, callOptions);
        }
      };
    return UimUgatewayUserviceServiceFutureStub.newStub(factory, channel);
  }

  /**
   * <pre>
   * im-gateway-service service
   * </pre>
   */
  public interface AsyncService {

    /**
     * <pre>
     * Add your RPC methods here
     * </pre>
     */
    default void healthCheck(com.pingxin403.cuckoo.im.gateway.HealthCheckRequest request,
        io.grpc.stub.StreamObserver<com.pingxin403.cuckoo.im.gateway.HealthCheckResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getHealthCheckMethod(), responseObserver);
    }
  }

  /**
   * Base class for the server implementation of the service UimUgatewayUserviceService.
   * <pre>
   * im-gateway-service service
   * </pre>
   */
  public static abstract class UimUgatewayUserviceServiceImplBase
      implements io.grpc.BindableService, AsyncService {

    @java.lang.Override public final io.grpc.ServerServiceDefinition bindService() {
      return UimUgatewayUserviceServiceGrpc.bindService(this);
    }
  }

  /**
   * A stub to allow clients to do asynchronous rpc calls to service UimUgatewayUserviceService.
   * <pre>
   * im-gateway-service service
   * </pre>
   */
  public static final class UimUgatewayUserviceServiceStub
      extends io.grpc.stub.AbstractAsyncStub<UimUgatewayUserviceServiceStub> {
    private UimUgatewayUserviceServiceStub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected UimUgatewayUserviceServiceStub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new UimUgatewayUserviceServiceStub(channel, callOptions);
    }

    /**
     * <pre>
     * Add your RPC methods here
     * </pre>
     */
    public void healthCheck(com.pingxin403.cuckoo.im.gateway.HealthCheckRequest request,
        io.grpc.stub.StreamObserver<com.pingxin403.cuckoo.im.gateway.HealthCheckResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getHealthCheckMethod(), getCallOptions()), request, responseObserver);
    }
  }

  /**
   * A stub to allow clients to do synchronous rpc calls to service UimUgatewayUserviceService.
   * <pre>
   * im-gateway-service service
   * </pre>
   */
  public static final class UimUgatewayUserviceServiceBlockingV2Stub
      extends io.grpc.stub.AbstractBlockingStub<UimUgatewayUserviceServiceBlockingV2Stub> {
    private UimUgatewayUserviceServiceBlockingV2Stub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected UimUgatewayUserviceServiceBlockingV2Stub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new UimUgatewayUserviceServiceBlockingV2Stub(channel, callOptions);
    }

    /**
     * <pre>
     * Add your RPC methods here
     * </pre>
     */
    public com.pingxin403.cuckoo.im.gateway.HealthCheckResponse healthCheck(com.pingxin403.cuckoo.im.gateway.HealthCheckRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getHealthCheckMethod(), getCallOptions(), request);
    }
  }

  /**
   * A stub to allow clients to do limited synchronous rpc calls to service UimUgatewayUserviceService.
   * <pre>
   * im-gateway-service service
   * </pre>
   */
  public static final class UimUgatewayUserviceServiceBlockingStub
      extends io.grpc.stub.AbstractBlockingStub<UimUgatewayUserviceServiceBlockingStub> {
    private UimUgatewayUserviceServiceBlockingStub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected UimUgatewayUserviceServiceBlockingStub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new UimUgatewayUserviceServiceBlockingStub(channel, callOptions);
    }

    /**
     * <pre>
     * Add your RPC methods here
     * </pre>
     */
    public com.pingxin403.cuckoo.im.gateway.HealthCheckResponse healthCheck(com.pingxin403.cuckoo.im.gateway.HealthCheckRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getHealthCheckMethod(), getCallOptions(), request);
    }
  }

  /**
   * A stub to allow clients to do ListenableFuture-style rpc calls to service UimUgatewayUserviceService.
   * <pre>
   * im-gateway-service service
   * </pre>
   */
  public static final class UimUgatewayUserviceServiceFutureStub
      extends io.grpc.stub.AbstractFutureStub<UimUgatewayUserviceServiceFutureStub> {
    private UimUgatewayUserviceServiceFutureStub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected UimUgatewayUserviceServiceFutureStub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new UimUgatewayUserviceServiceFutureStub(channel, callOptions);
    }

    /**
     * <pre>
     * Add your RPC methods here
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<com.pingxin403.cuckoo.im.gateway.HealthCheckResponse> healthCheck(
        com.pingxin403.cuckoo.im.gateway.HealthCheckRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getHealthCheckMethod(), getCallOptions()), request);
    }
  }

  private static final int METHODID_HEALTH_CHECK = 0;

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
        case METHODID_HEALTH_CHECK:
          serviceImpl.healthCheck((com.pingxin403.cuckoo.im.gateway.HealthCheckRequest) request,
              (io.grpc.stub.StreamObserver<com.pingxin403.cuckoo.im.gateway.HealthCheckResponse>) responseObserver);
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
          getHealthCheckMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              com.pingxin403.cuckoo.im.gateway.HealthCheckRequest,
              com.pingxin403.cuckoo.im.gateway.HealthCheckResponse>(
                service, METHODID_HEALTH_CHECK)))
        .build();
  }

  private static abstract class UimUgatewayUserviceServiceBaseDescriptorSupplier
      implements io.grpc.protobuf.ProtoFileDescriptorSupplier, io.grpc.protobuf.ProtoServiceDescriptorSupplier {
    UimUgatewayUserviceServiceBaseDescriptorSupplier() {}

    @java.lang.Override
    public com.google.protobuf.Descriptors.FileDescriptor getFileDescriptor() {
      return com.pingxin403.cuckoo.im.gateway.ImGatewayServiceProto.getDescriptor();
    }

    @java.lang.Override
    public com.google.protobuf.Descriptors.ServiceDescriptor getServiceDescriptor() {
      return getFileDescriptor().findServiceByName("UimUgatewayUserviceService");
    }
  }

  private static final class UimUgatewayUserviceServiceFileDescriptorSupplier
      extends UimUgatewayUserviceServiceBaseDescriptorSupplier {
    UimUgatewayUserviceServiceFileDescriptorSupplier() {}
  }

  private static final class UimUgatewayUserviceServiceMethodDescriptorSupplier
      extends UimUgatewayUserviceServiceBaseDescriptorSupplier
      implements io.grpc.protobuf.ProtoMethodDescriptorSupplier {
    private final java.lang.String methodName;

    UimUgatewayUserviceServiceMethodDescriptorSupplier(java.lang.String methodName) {
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
      synchronized (UimUgatewayUserviceServiceGrpc.class) {
        result = serviceDescriptor;
        if (result == null) {
          serviceDescriptor = result = io.grpc.ServiceDescriptor.newBuilder(SERVICE_NAME)
              .setSchemaDescriptor(new UimUgatewayUserviceServiceFileDescriptorSupplier())
              .addMethod(getHealthCheckMethod())
              .build();
        }
      }
    }
    return result;
  }
}
