package com.pingxin403.cuckoo.flashsale;

import static io.grpc.MethodDescriptor.generateFullMethodName;

/**
 * <pre>
 * Flash sale service
 * </pre>
 */
@io.grpc.stub.annotations.GrpcGenerated
public final class UflashUsaleUserviceServiceGrpc {

  private UflashUsaleUserviceServiceGrpc() {}

  public static final java.lang.String SERVICE_NAME = "flash_sale_servicepb.UflashUsaleUserviceService";

  // Static method descriptors that strictly reflect the proto.
  private static volatile io.grpc.MethodDescriptor<com.pingxin403.cuckoo.flashsale.HealthCheckRequest,
      com.pingxin403.cuckoo.flashsale.HealthCheckResponse> getHealthCheckMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "HealthCheck",
      requestType = com.pingxin403.cuckoo.flashsale.HealthCheckRequest.class,
      responseType = com.pingxin403.cuckoo.flashsale.HealthCheckResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<com.pingxin403.cuckoo.flashsale.HealthCheckRequest,
      com.pingxin403.cuckoo.flashsale.HealthCheckResponse> getHealthCheckMethod() {
    io.grpc.MethodDescriptor<com.pingxin403.cuckoo.flashsale.HealthCheckRequest, com.pingxin403.cuckoo.flashsale.HealthCheckResponse> getHealthCheckMethod;
    if ((getHealthCheckMethod = UflashUsaleUserviceServiceGrpc.getHealthCheckMethod) == null) {
      synchronized (UflashUsaleUserviceServiceGrpc.class) {
        if ((getHealthCheckMethod = UflashUsaleUserviceServiceGrpc.getHealthCheckMethod) == null) {
          UflashUsaleUserviceServiceGrpc.getHealthCheckMethod = getHealthCheckMethod =
              io.grpc.MethodDescriptor.<com.pingxin403.cuckoo.flashsale.HealthCheckRequest, com.pingxin403.cuckoo.flashsale.HealthCheckResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "HealthCheck"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  com.pingxin403.cuckoo.flashsale.HealthCheckRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  com.pingxin403.cuckoo.flashsale.HealthCheckResponse.getDefaultInstance()))
              .setSchemaDescriptor(new UflashUsaleUserviceServiceMethodDescriptorSupplier("HealthCheck"))
              .build();
        }
      }
    }
    return getHealthCheckMethod;
  }

  /**
   * Creates a new async stub that supports all call types for the service
   */
  public static UflashUsaleUserviceServiceStub newStub(io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<UflashUsaleUserviceServiceStub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<UflashUsaleUserviceServiceStub>() {
        @java.lang.Override
        public UflashUsaleUserviceServiceStub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new UflashUsaleUserviceServiceStub(channel, callOptions);
        }
      };
    return UflashUsaleUserviceServiceStub.newStub(factory, channel);
  }

  /**
   * Creates a new blocking-style stub that supports all types of calls on the service
   */
  public static UflashUsaleUserviceServiceBlockingV2Stub newBlockingV2Stub(
      io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<UflashUsaleUserviceServiceBlockingV2Stub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<UflashUsaleUserviceServiceBlockingV2Stub>() {
        @java.lang.Override
        public UflashUsaleUserviceServiceBlockingV2Stub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new UflashUsaleUserviceServiceBlockingV2Stub(channel, callOptions);
        }
      };
    return UflashUsaleUserviceServiceBlockingV2Stub.newStub(factory, channel);
  }

  /**
   * Creates a new blocking-style stub that supports unary and streaming output calls on the service
   */
  public static UflashUsaleUserviceServiceBlockingStub newBlockingStub(
      io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<UflashUsaleUserviceServiceBlockingStub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<UflashUsaleUserviceServiceBlockingStub>() {
        @java.lang.Override
        public UflashUsaleUserviceServiceBlockingStub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new UflashUsaleUserviceServiceBlockingStub(channel, callOptions);
        }
      };
    return UflashUsaleUserviceServiceBlockingStub.newStub(factory, channel);
  }

  /**
   * Creates a new ListenableFuture-style stub that supports unary calls on the service
   */
  public static UflashUsaleUserviceServiceFutureStub newFutureStub(
      io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<UflashUsaleUserviceServiceFutureStub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<UflashUsaleUserviceServiceFutureStub>() {
        @java.lang.Override
        public UflashUsaleUserviceServiceFutureStub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new UflashUsaleUserviceServiceFutureStub(channel, callOptions);
        }
      };
    return UflashUsaleUserviceServiceFutureStub.newStub(factory, channel);
  }

  /**
   * <pre>
   * Flash sale service
   * </pre>
   */
  public interface AsyncService {

    /**
     * <pre>
     * Add your RPC methods here
     * </pre>
     */
    default void healthCheck(com.pingxin403.cuckoo.flashsale.HealthCheckRequest request,
        io.grpc.stub.StreamObserver<com.pingxin403.cuckoo.flashsale.HealthCheckResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getHealthCheckMethod(), responseObserver);
    }
  }

  /**
   * Base class for the server implementation of the service UflashUsaleUserviceService.
   * <pre>
   * Flash sale service
   * </pre>
   */
  public static abstract class UflashUsaleUserviceServiceImplBase
      implements io.grpc.BindableService, AsyncService {

    @java.lang.Override public final io.grpc.ServerServiceDefinition bindService() {
      return UflashUsaleUserviceServiceGrpc.bindService(this);
    }
  }

  /**
   * A stub to allow clients to do asynchronous rpc calls to service UflashUsaleUserviceService.
   * <pre>
   * Flash sale service
   * </pre>
   */
  public static final class UflashUsaleUserviceServiceStub
      extends io.grpc.stub.AbstractAsyncStub<UflashUsaleUserviceServiceStub> {
    private UflashUsaleUserviceServiceStub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected UflashUsaleUserviceServiceStub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new UflashUsaleUserviceServiceStub(channel, callOptions);
    }

    /**
     * <pre>
     * Add your RPC methods here
     * </pre>
     */
    public void healthCheck(com.pingxin403.cuckoo.flashsale.HealthCheckRequest request,
        io.grpc.stub.StreamObserver<com.pingxin403.cuckoo.flashsale.HealthCheckResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getHealthCheckMethod(), getCallOptions()), request, responseObserver);
    }
  }

  /**
   * A stub to allow clients to do synchronous rpc calls to service UflashUsaleUserviceService.
   * <pre>
   * Flash sale service
   * </pre>
   */
  public static final class UflashUsaleUserviceServiceBlockingV2Stub
      extends io.grpc.stub.AbstractBlockingStub<UflashUsaleUserviceServiceBlockingV2Stub> {
    private UflashUsaleUserviceServiceBlockingV2Stub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected UflashUsaleUserviceServiceBlockingV2Stub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new UflashUsaleUserviceServiceBlockingV2Stub(channel, callOptions);
    }

    /**
     * <pre>
     * Add your RPC methods here
     * </pre>
     */
    public com.pingxin403.cuckoo.flashsale.HealthCheckResponse healthCheck(com.pingxin403.cuckoo.flashsale.HealthCheckRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getHealthCheckMethod(), getCallOptions(), request);
    }
  }

  /**
   * A stub to allow clients to do limited synchronous rpc calls to service UflashUsaleUserviceService.
   * <pre>
   * Flash sale service
   * </pre>
   */
  public static final class UflashUsaleUserviceServiceBlockingStub
      extends io.grpc.stub.AbstractBlockingStub<UflashUsaleUserviceServiceBlockingStub> {
    private UflashUsaleUserviceServiceBlockingStub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected UflashUsaleUserviceServiceBlockingStub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new UflashUsaleUserviceServiceBlockingStub(channel, callOptions);
    }

    /**
     * <pre>
     * Add your RPC methods here
     * </pre>
     */
    public com.pingxin403.cuckoo.flashsale.HealthCheckResponse healthCheck(com.pingxin403.cuckoo.flashsale.HealthCheckRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getHealthCheckMethod(), getCallOptions(), request);
    }
  }

  /**
   * A stub to allow clients to do ListenableFuture-style rpc calls to service UflashUsaleUserviceService.
   * <pre>
   * Flash sale service
   * </pre>
   */
  public static final class UflashUsaleUserviceServiceFutureStub
      extends io.grpc.stub.AbstractFutureStub<UflashUsaleUserviceServiceFutureStub> {
    private UflashUsaleUserviceServiceFutureStub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected UflashUsaleUserviceServiceFutureStub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new UflashUsaleUserviceServiceFutureStub(channel, callOptions);
    }

    /**
     * <pre>
     * Add your RPC methods here
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<com.pingxin403.cuckoo.flashsale.HealthCheckResponse> healthCheck(
        com.pingxin403.cuckoo.flashsale.HealthCheckRequest request) {
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
          serviceImpl.healthCheck((com.pingxin403.cuckoo.flashsale.HealthCheckRequest) request,
              (io.grpc.stub.StreamObserver<com.pingxin403.cuckoo.flashsale.HealthCheckResponse>) responseObserver);
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
              com.pingxin403.cuckoo.flashsale.HealthCheckRequest,
              com.pingxin403.cuckoo.flashsale.HealthCheckResponse>(
                service, METHODID_HEALTH_CHECK)))
        .build();
  }

  private static abstract class UflashUsaleUserviceServiceBaseDescriptorSupplier
      implements io.grpc.protobuf.ProtoFileDescriptorSupplier, io.grpc.protobuf.ProtoServiceDescriptorSupplier {
    UflashUsaleUserviceServiceBaseDescriptorSupplier() {}

    @java.lang.Override
    public com.google.protobuf.Descriptors.FileDescriptor getFileDescriptor() {
      return com.pingxin403.cuckoo.flashsale.FlashSaleService.getDescriptor();
    }

    @java.lang.Override
    public com.google.protobuf.Descriptors.ServiceDescriptor getServiceDescriptor() {
      return getFileDescriptor().findServiceByName("UflashUsaleUserviceService");
    }
  }

  private static final class UflashUsaleUserviceServiceFileDescriptorSupplier
      extends UflashUsaleUserviceServiceBaseDescriptorSupplier {
    UflashUsaleUserviceServiceFileDescriptorSupplier() {}
  }

  private static final class UflashUsaleUserviceServiceMethodDescriptorSupplier
      extends UflashUsaleUserviceServiceBaseDescriptorSupplier
      implements io.grpc.protobuf.ProtoMethodDescriptorSupplier {
    private final java.lang.String methodName;

    UflashUsaleUserviceServiceMethodDescriptorSupplier(java.lang.String methodName) {
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
      synchronized (UflashUsaleUserviceServiceGrpc.class) {
        result = serviceDescriptor;
        if (result == null) {
          serviceDescriptor = result = io.grpc.ServiceDescriptor.newBuilder(SERVICE_NAME)
              .setSchemaDescriptor(new UflashUsaleUserviceServiceFileDescriptorSupplier())
              .addMethod(getHealthCheckMethod())
              .build();
        }
      }
    }
    return result;
  }
}
