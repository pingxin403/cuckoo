package com.pingxin403.api.v1;

import static io.grpc.MethodDescriptor.generateFullMethodName;

/**
 * <pre>
 * ShortenerService provides URL shortening functionality
 * This service transforms long URLs into short, memorable links and provides
 * fast redirection. It supports custom short codes, expiration times, and
 * basic analytics.
 * </pre>
 */
@io.grpc.stub.annotations.GrpcGenerated
public final class ShortenerServiceGrpc {

  private ShortenerServiceGrpc() {}

  public static final java.lang.String SERVICE_NAME = "api.v1.ShortenerService";

  // Static method descriptors that strictly reflect the proto.
  private static volatile io.grpc.MethodDescriptor<com.pingxin403.api.v1.CreateShortLinkRequest,
      com.pingxin403.api.v1.CreateShortLinkResponse> getCreateShortLinkMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "CreateShortLink",
      requestType = com.pingxin403.api.v1.CreateShortLinkRequest.class,
      responseType = com.pingxin403.api.v1.CreateShortLinkResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<com.pingxin403.api.v1.CreateShortLinkRequest,
      com.pingxin403.api.v1.CreateShortLinkResponse> getCreateShortLinkMethod() {
    io.grpc.MethodDescriptor<com.pingxin403.api.v1.CreateShortLinkRequest, com.pingxin403.api.v1.CreateShortLinkResponse> getCreateShortLinkMethod;
    if ((getCreateShortLinkMethod = ShortenerServiceGrpc.getCreateShortLinkMethod) == null) {
      synchronized (ShortenerServiceGrpc.class) {
        if ((getCreateShortLinkMethod = ShortenerServiceGrpc.getCreateShortLinkMethod) == null) {
          ShortenerServiceGrpc.getCreateShortLinkMethod = getCreateShortLinkMethod =
              io.grpc.MethodDescriptor.<com.pingxin403.api.v1.CreateShortLinkRequest, com.pingxin403.api.v1.CreateShortLinkResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "CreateShortLink"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  com.pingxin403.api.v1.CreateShortLinkRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  com.pingxin403.api.v1.CreateShortLinkResponse.getDefaultInstance()))
              .setSchemaDescriptor(new ShortenerServiceMethodDescriptorSupplier("CreateShortLink"))
              .build();
        }
      }
    }
    return getCreateShortLinkMethod;
  }

  private static volatile io.grpc.MethodDescriptor<com.pingxin403.api.v1.GetLinkInfoRequest,
      com.pingxin403.api.v1.GetLinkInfoResponse> getGetLinkInfoMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "GetLinkInfo",
      requestType = com.pingxin403.api.v1.GetLinkInfoRequest.class,
      responseType = com.pingxin403.api.v1.GetLinkInfoResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<com.pingxin403.api.v1.GetLinkInfoRequest,
      com.pingxin403.api.v1.GetLinkInfoResponse> getGetLinkInfoMethod() {
    io.grpc.MethodDescriptor<com.pingxin403.api.v1.GetLinkInfoRequest, com.pingxin403.api.v1.GetLinkInfoResponse> getGetLinkInfoMethod;
    if ((getGetLinkInfoMethod = ShortenerServiceGrpc.getGetLinkInfoMethod) == null) {
      synchronized (ShortenerServiceGrpc.class) {
        if ((getGetLinkInfoMethod = ShortenerServiceGrpc.getGetLinkInfoMethod) == null) {
          ShortenerServiceGrpc.getGetLinkInfoMethod = getGetLinkInfoMethod =
              io.grpc.MethodDescriptor.<com.pingxin403.api.v1.GetLinkInfoRequest, com.pingxin403.api.v1.GetLinkInfoResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "GetLinkInfo"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  com.pingxin403.api.v1.GetLinkInfoRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  com.pingxin403.api.v1.GetLinkInfoResponse.getDefaultInstance()))
              .setSchemaDescriptor(new ShortenerServiceMethodDescriptorSupplier("GetLinkInfo"))
              .build();
        }
      }
    }
    return getGetLinkInfoMethod;
  }

  private static volatile io.grpc.MethodDescriptor<com.pingxin403.api.v1.DeleteShortLinkRequest,
      com.pingxin403.api.v1.DeleteShortLinkResponse> getDeleteShortLinkMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "DeleteShortLink",
      requestType = com.pingxin403.api.v1.DeleteShortLinkRequest.class,
      responseType = com.pingxin403.api.v1.DeleteShortLinkResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<com.pingxin403.api.v1.DeleteShortLinkRequest,
      com.pingxin403.api.v1.DeleteShortLinkResponse> getDeleteShortLinkMethod() {
    io.grpc.MethodDescriptor<com.pingxin403.api.v1.DeleteShortLinkRequest, com.pingxin403.api.v1.DeleteShortLinkResponse> getDeleteShortLinkMethod;
    if ((getDeleteShortLinkMethod = ShortenerServiceGrpc.getDeleteShortLinkMethod) == null) {
      synchronized (ShortenerServiceGrpc.class) {
        if ((getDeleteShortLinkMethod = ShortenerServiceGrpc.getDeleteShortLinkMethod) == null) {
          ShortenerServiceGrpc.getDeleteShortLinkMethod = getDeleteShortLinkMethod =
              io.grpc.MethodDescriptor.<com.pingxin403.api.v1.DeleteShortLinkRequest, com.pingxin403.api.v1.DeleteShortLinkResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "DeleteShortLink"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  com.pingxin403.api.v1.DeleteShortLinkRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  com.pingxin403.api.v1.DeleteShortLinkResponse.getDefaultInstance()))
              .setSchemaDescriptor(new ShortenerServiceMethodDescriptorSupplier("DeleteShortLink"))
              .build();
        }
      }
    }
    return getDeleteShortLinkMethod;
  }

  /**
   * Creates a new async stub that supports all call types for the service
   */
  public static ShortenerServiceStub newStub(io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<ShortenerServiceStub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<ShortenerServiceStub>() {
        @java.lang.Override
        public ShortenerServiceStub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new ShortenerServiceStub(channel, callOptions);
        }
      };
    return ShortenerServiceStub.newStub(factory, channel);
  }

  /**
   * Creates a new blocking-style stub that supports all types of calls on the service
   */
  public static ShortenerServiceBlockingV2Stub newBlockingV2Stub(
      io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<ShortenerServiceBlockingV2Stub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<ShortenerServiceBlockingV2Stub>() {
        @java.lang.Override
        public ShortenerServiceBlockingV2Stub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new ShortenerServiceBlockingV2Stub(channel, callOptions);
        }
      };
    return ShortenerServiceBlockingV2Stub.newStub(factory, channel);
  }

  /**
   * Creates a new blocking-style stub that supports unary and streaming output calls on the service
   */
  public static ShortenerServiceBlockingStub newBlockingStub(
      io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<ShortenerServiceBlockingStub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<ShortenerServiceBlockingStub>() {
        @java.lang.Override
        public ShortenerServiceBlockingStub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new ShortenerServiceBlockingStub(channel, callOptions);
        }
      };
    return ShortenerServiceBlockingStub.newStub(factory, channel);
  }

  /**
   * Creates a new ListenableFuture-style stub that supports unary calls on the service
   */
  public static ShortenerServiceFutureStub newFutureStub(
      io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<ShortenerServiceFutureStub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<ShortenerServiceFutureStub>() {
        @java.lang.Override
        public ShortenerServiceFutureStub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new ShortenerServiceFutureStub(channel, callOptions);
        }
      };
    return ShortenerServiceFutureStub.newStub(factory, channel);
  }

  /**
   * <pre>
   * ShortenerService provides URL shortening functionality
   * This service transforms long URLs into short, memorable links and provides
   * fast redirection. It supports custom short codes, expiration times, and
   * basic analytics.
   * </pre>
   */
  public interface AsyncService {

    /**
     * <pre>
     * CreateShortLink creates a new short link from a long URL
     * Generates a unique 7-character short code (or uses a custom code if provided)
     * and stores the mapping. The service validates the URL format and checks for
     * malicious patterns before creating the link.
     * Example:
     *   Request: { long_url: "https://example.com/very/long/path" }
     *   Response: { short_url: "https://ex.co/abc123x", short_code: "abc123x", ... }
     *   Request: { long_url: "https://example.com", custom_code: "promo2024" }
     *   Response: { short_url: "https://ex.co/promo2024", short_code: "promo2024", ... }
     * Errors:
     *   - INVALID_ARGUMENT: Invalid URL format, URL too long (&gt;2048 chars), or malicious pattern detected
     *   - ALREADY_EXISTS: Custom code already in use (HTTP 409)
     *   - RESOURCE_EXHAUSTED: Rate limit exceeded (HTTP 429)
     *   - INTERNAL: Failed to generate unique code after retries
     * </pre>
     */
    default void createShortLink(com.pingxin403.api.v1.CreateShortLinkRequest request,
        io.grpc.stub.StreamObserver<com.pingxin403.api.v1.CreateShortLinkResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getCreateShortLinkMethod(), responseObserver);
    }

    /**
     * <pre>
     * GetLinkInfo retrieves metadata for a short link
     * Returns information about a short link including the original URL,
     * creation time, expiration time (if set), and click statistics.
     * Example:
     *   Request: { short_code: "abc123x" }
     *   Response: { short_code: "abc123x", long_url: "https://example.com", click_count: 42, ... }
     * Errors:
     *   - INVALID_ARGUMENT: Empty or invalid short code
     *   - NOT_FOUND: Short code does not exist
     * </pre>
     */
    default void getLinkInfo(com.pingxin403.api.v1.GetLinkInfoRequest request,
        io.grpc.stub.StreamObserver<com.pingxin403.api.v1.GetLinkInfoResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getGetLinkInfoMethod(), responseObserver);
    }

    /**
     * <pre>
     * DeleteShortLink removes a short link
     * Performs a soft delete of the short link, making it inaccessible for future
     * redirects. The operation is idempotent - deleting an already deleted link
     * returns success.
     * Example:
     *   Request: { short_code: "abc123x" }
     *   Response: { success: true }
     * Errors:
     *   - INVALID_ARGUMENT: Empty or invalid short code
     *   - NOT_FOUND: Short code does not exist
     * </pre>
     */
    default void deleteShortLink(com.pingxin403.api.v1.DeleteShortLinkRequest request,
        io.grpc.stub.StreamObserver<com.pingxin403.api.v1.DeleteShortLinkResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getDeleteShortLinkMethod(), responseObserver);
    }
  }

  /**
   * Base class for the server implementation of the service ShortenerService.
   * <pre>
   * ShortenerService provides URL shortening functionality
   * This service transforms long URLs into short, memorable links and provides
   * fast redirection. It supports custom short codes, expiration times, and
   * basic analytics.
   * </pre>
   */
  public static abstract class ShortenerServiceImplBase
      implements io.grpc.BindableService, AsyncService {

    @java.lang.Override public final io.grpc.ServerServiceDefinition bindService() {
      return ShortenerServiceGrpc.bindService(this);
    }
  }

  /**
   * A stub to allow clients to do asynchronous rpc calls to service ShortenerService.
   * <pre>
   * ShortenerService provides URL shortening functionality
   * This service transforms long URLs into short, memorable links and provides
   * fast redirection. It supports custom short codes, expiration times, and
   * basic analytics.
   * </pre>
   */
  public static final class ShortenerServiceStub
      extends io.grpc.stub.AbstractAsyncStub<ShortenerServiceStub> {
    private ShortenerServiceStub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected ShortenerServiceStub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new ShortenerServiceStub(channel, callOptions);
    }

    /**
     * <pre>
     * CreateShortLink creates a new short link from a long URL
     * Generates a unique 7-character short code (or uses a custom code if provided)
     * and stores the mapping. The service validates the URL format and checks for
     * malicious patterns before creating the link.
     * Example:
     *   Request: { long_url: "https://example.com/very/long/path" }
     *   Response: { short_url: "https://ex.co/abc123x", short_code: "abc123x", ... }
     *   Request: { long_url: "https://example.com", custom_code: "promo2024" }
     *   Response: { short_url: "https://ex.co/promo2024", short_code: "promo2024", ... }
     * Errors:
     *   - INVALID_ARGUMENT: Invalid URL format, URL too long (&gt;2048 chars), or malicious pattern detected
     *   - ALREADY_EXISTS: Custom code already in use (HTTP 409)
     *   - RESOURCE_EXHAUSTED: Rate limit exceeded (HTTP 429)
     *   - INTERNAL: Failed to generate unique code after retries
     * </pre>
     */
    public void createShortLink(com.pingxin403.api.v1.CreateShortLinkRequest request,
        io.grpc.stub.StreamObserver<com.pingxin403.api.v1.CreateShortLinkResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getCreateShortLinkMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     * <pre>
     * GetLinkInfo retrieves metadata for a short link
     * Returns information about a short link including the original URL,
     * creation time, expiration time (if set), and click statistics.
     * Example:
     *   Request: { short_code: "abc123x" }
     *   Response: { short_code: "abc123x", long_url: "https://example.com", click_count: 42, ... }
     * Errors:
     *   - INVALID_ARGUMENT: Empty or invalid short code
     *   - NOT_FOUND: Short code does not exist
     * </pre>
     */
    public void getLinkInfo(com.pingxin403.api.v1.GetLinkInfoRequest request,
        io.grpc.stub.StreamObserver<com.pingxin403.api.v1.GetLinkInfoResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getGetLinkInfoMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     * <pre>
     * DeleteShortLink removes a short link
     * Performs a soft delete of the short link, making it inaccessible for future
     * redirects. The operation is idempotent - deleting an already deleted link
     * returns success.
     * Example:
     *   Request: { short_code: "abc123x" }
     *   Response: { success: true }
     * Errors:
     *   - INVALID_ARGUMENT: Empty or invalid short code
     *   - NOT_FOUND: Short code does not exist
     * </pre>
     */
    public void deleteShortLink(com.pingxin403.api.v1.DeleteShortLinkRequest request,
        io.grpc.stub.StreamObserver<com.pingxin403.api.v1.DeleteShortLinkResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getDeleteShortLinkMethod(), getCallOptions()), request, responseObserver);
    }
  }

  /**
   * A stub to allow clients to do synchronous rpc calls to service ShortenerService.
   * <pre>
   * ShortenerService provides URL shortening functionality
   * This service transforms long URLs into short, memorable links and provides
   * fast redirection. It supports custom short codes, expiration times, and
   * basic analytics.
   * </pre>
   */
  public static final class ShortenerServiceBlockingV2Stub
      extends io.grpc.stub.AbstractBlockingStub<ShortenerServiceBlockingV2Stub> {
    private ShortenerServiceBlockingV2Stub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected ShortenerServiceBlockingV2Stub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new ShortenerServiceBlockingV2Stub(channel, callOptions);
    }

    /**
     * <pre>
     * CreateShortLink creates a new short link from a long URL
     * Generates a unique 7-character short code (or uses a custom code if provided)
     * and stores the mapping. The service validates the URL format and checks for
     * malicious patterns before creating the link.
     * Example:
     *   Request: { long_url: "https://example.com/very/long/path" }
     *   Response: { short_url: "https://ex.co/abc123x", short_code: "abc123x", ... }
     *   Request: { long_url: "https://example.com", custom_code: "promo2024" }
     *   Response: { short_url: "https://ex.co/promo2024", short_code: "promo2024", ... }
     * Errors:
     *   - INVALID_ARGUMENT: Invalid URL format, URL too long (&gt;2048 chars), or malicious pattern detected
     *   - ALREADY_EXISTS: Custom code already in use (HTTP 409)
     *   - RESOURCE_EXHAUSTED: Rate limit exceeded (HTTP 429)
     *   - INTERNAL: Failed to generate unique code after retries
     * </pre>
     */
    public com.pingxin403.api.v1.CreateShortLinkResponse createShortLink(com.pingxin403.api.v1.CreateShortLinkRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getCreateShortLinkMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * GetLinkInfo retrieves metadata for a short link
     * Returns information about a short link including the original URL,
     * creation time, expiration time (if set), and click statistics.
     * Example:
     *   Request: { short_code: "abc123x" }
     *   Response: { short_code: "abc123x", long_url: "https://example.com", click_count: 42, ... }
     * Errors:
     *   - INVALID_ARGUMENT: Empty or invalid short code
     *   - NOT_FOUND: Short code does not exist
     * </pre>
     */
    public com.pingxin403.api.v1.GetLinkInfoResponse getLinkInfo(com.pingxin403.api.v1.GetLinkInfoRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getGetLinkInfoMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * DeleteShortLink removes a short link
     * Performs a soft delete of the short link, making it inaccessible for future
     * redirects. The operation is idempotent - deleting an already deleted link
     * returns success.
     * Example:
     *   Request: { short_code: "abc123x" }
     *   Response: { success: true }
     * Errors:
     *   - INVALID_ARGUMENT: Empty or invalid short code
     *   - NOT_FOUND: Short code does not exist
     * </pre>
     */
    public com.pingxin403.api.v1.DeleteShortLinkResponse deleteShortLink(com.pingxin403.api.v1.DeleteShortLinkRequest request) throws io.grpc.StatusException {
      return io.grpc.stub.ClientCalls.blockingV2UnaryCall(
          getChannel(), getDeleteShortLinkMethod(), getCallOptions(), request);
    }
  }

  /**
   * A stub to allow clients to do limited synchronous rpc calls to service ShortenerService.
   * <pre>
   * ShortenerService provides URL shortening functionality
   * This service transforms long URLs into short, memorable links and provides
   * fast redirection. It supports custom short codes, expiration times, and
   * basic analytics.
   * </pre>
   */
  public static final class ShortenerServiceBlockingStub
      extends io.grpc.stub.AbstractBlockingStub<ShortenerServiceBlockingStub> {
    private ShortenerServiceBlockingStub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected ShortenerServiceBlockingStub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new ShortenerServiceBlockingStub(channel, callOptions);
    }

    /**
     * <pre>
     * CreateShortLink creates a new short link from a long URL
     * Generates a unique 7-character short code (or uses a custom code if provided)
     * and stores the mapping. The service validates the URL format and checks for
     * malicious patterns before creating the link.
     * Example:
     *   Request: { long_url: "https://example.com/very/long/path" }
     *   Response: { short_url: "https://ex.co/abc123x", short_code: "abc123x", ... }
     *   Request: { long_url: "https://example.com", custom_code: "promo2024" }
     *   Response: { short_url: "https://ex.co/promo2024", short_code: "promo2024", ... }
     * Errors:
     *   - INVALID_ARGUMENT: Invalid URL format, URL too long (&gt;2048 chars), or malicious pattern detected
     *   - ALREADY_EXISTS: Custom code already in use (HTTP 409)
     *   - RESOURCE_EXHAUSTED: Rate limit exceeded (HTTP 429)
     *   - INTERNAL: Failed to generate unique code after retries
     * </pre>
     */
    public com.pingxin403.api.v1.CreateShortLinkResponse createShortLink(com.pingxin403.api.v1.CreateShortLinkRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getCreateShortLinkMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * GetLinkInfo retrieves metadata for a short link
     * Returns information about a short link including the original URL,
     * creation time, expiration time (if set), and click statistics.
     * Example:
     *   Request: { short_code: "abc123x" }
     *   Response: { short_code: "abc123x", long_url: "https://example.com", click_count: 42, ... }
     * Errors:
     *   - INVALID_ARGUMENT: Empty or invalid short code
     *   - NOT_FOUND: Short code does not exist
     * </pre>
     */
    public com.pingxin403.api.v1.GetLinkInfoResponse getLinkInfo(com.pingxin403.api.v1.GetLinkInfoRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getGetLinkInfoMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * DeleteShortLink removes a short link
     * Performs a soft delete of the short link, making it inaccessible for future
     * redirects. The operation is idempotent - deleting an already deleted link
     * returns success.
     * Example:
     *   Request: { short_code: "abc123x" }
     *   Response: { success: true }
     * Errors:
     *   - INVALID_ARGUMENT: Empty or invalid short code
     *   - NOT_FOUND: Short code does not exist
     * </pre>
     */
    public com.pingxin403.api.v1.DeleteShortLinkResponse deleteShortLink(com.pingxin403.api.v1.DeleteShortLinkRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getDeleteShortLinkMethod(), getCallOptions(), request);
    }
  }

  /**
   * A stub to allow clients to do ListenableFuture-style rpc calls to service ShortenerService.
   * <pre>
   * ShortenerService provides URL shortening functionality
   * This service transforms long URLs into short, memorable links and provides
   * fast redirection. It supports custom short codes, expiration times, and
   * basic analytics.
   * </pre>
   */
  public static final class ShortenerServiceFutureStub
      extends io.grpc.stub.AbstractFutureStub<ShortenerServiceFutureStub> {
    private ShortenerServiceFutureStub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected ShortenerServiceFutureStub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new ShortenerServiceFutureStub(channel, callOptions);
    }

    /**
     * <pre>
     * CreateShortLink creates a new short link from a long URL
     * Generates a unique 7-character short code (or uses a custom code if provided)
     * and stores the mapping. The service validates the URL format and checks for
     * malicious patterns before creating the link.
     * Example:
     *   Request: { long_url: "https://example.com/very/long/path" }
     *   Response: { short_url: "https://ex.co/abc123x", short_code: "abc123x", ... }
     *   Request: { long_url: "https://example.com", custom_code: "promo2024" }
     *   Response: { short_url: "https://ex.co/promo2024", short_code: "promo2024", ... }
     * Errors:
     *   - INVALID_ARGUMENT: Invalid URL format, URL too long (&gt;2048 chars), or malicious pattern detected
     *   - ALREADY_EXISTS: Custom code already in use (HTTP 409)
     *   - RESOURCE_EXHAUSTED: Rate limit exceeded (HTTP 429)
     *   - INTERNAL: Failed to generate unique code after retries
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<com.pingxin403.api.v1.CreateShortLinkResponse> createShortLink(
        com.pingxin403.api.v1.CreateShortLinkRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getCreateShortLinkMethod(), getCallOptions()), request);
    }

    /**
     * <pre>
     * GetLinkInfo retrieves metadata for a short link
     * Returns information about a short link including the original URL,
     * creation time, expiration time (if set), and click statistics.
     * Example:
     *   Request: { short_code: "abc123x" }
     *   Response: { short_code: "abc123x", long_url: "https://example.com", click_count: 42, ... }
     * Errors:
     *   - INVALID_ARGUMENT: Empty or invalid short code
     *   - NOT_FOUND: Short code does not exist
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<com.pingxin403.api.v1.GetLinkInfoResponse> getLinkInfo(
        com.pingxin403.api.v1.GetLinkInfoRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getGetLinkInfoMethod(), getCallOptions()), request);
    }

    /**
     * <pre>
     * DeleteShortLink removes a short link
     * Performs a soft delete of the short link, making it inaccessible for future
     * redirects. The operation is idempotent - deleting an already deleted link
     * returns success.
     * Example:
     *   Request: { short_code: "abc123x" }
     *   Response: { success: true }
     * Errors:
     *   - INVALID_ARGUMENT: Empty or invalid short code
     *   - NOT_FOUND: Short code does not exist
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<com.pingxin403.api.v1.DeleteShortLinkResponse> deleteShortLink(
        com.pingxin403.api.v1.DeleteShortLinkRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getDeleteShortLinkMethod(), getCallOptions()), request);
    }
  }

  private static final int METHODID_CREATE_SHORT_LINK = 0;
  private static final int METHODID_GET_LINK_INFO = 1;
  private static final int METHODID_DELETE_SHORT_LINK = 2;

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
        case METHODID_CREATE_SHORT_LINK:
          serviceImpl.createShortLink((com.pingxin403.api.v1.CreateShortLinkRequest) request,
              (io.grpc.stub.StreamObserver<com.pingxin403.api.v1.CreateShortLinkResponse>) responseObserver);
          break;
        case METHODID_GET_LINK_INFO:
          serviceImpl.getLinkInfo((com.pingxin403.api.v1.GetLinkInfoRequest) request,
              (io.grpc.stub.StreamObserver<com.pingxin403.api.v1.GetLinkInfoResponse>) responseObserver);
          break;
        case METHODID_DELETE_SHORT_LINK:
          serviceImpl.deleteShortLink((com.pingxin403.api.v1.DeleteShortLinkRequest) request,
              (io.grpc.stub.StreamObserver<com.pingxin403.api.v1.DeleteShortLinkResponse>) responseObserver);
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
          getCreateShortLinkMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              com.pingxin403.api.v1.CreateShortLinkRequest,
              com.pingxin403.api.v1.CreateShortLinkResponse>(
                service, METHODID_CREATE_SHORT_LINK)))
        .addMethod(
          getGetLinkInfoMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              com.pingxin403.api.v1.GetLinkInfoRequest,
              com.pingxin403.api.v1.GetLinkInfoResponse>(
                service, METHODID_GET_LINK_INFO)))
        .addMethod(
          getDeleteShortLinkMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              com.pingxin403.api.v1.DeleteShortLinkRequest,
              com.pingxin403.api.v1.DeleteShortLinkResponse>(
                service, METHODID_DELETE_SHORT_LINK)))
        .build();
  }

  private static abstract class ShortenerServiceBaseDescriptorSupplier
      implements io.grpc.protobuf.ProtoFileDescriptorSupplier, io.grpc.protobuf.ProtoServiceDescriptorSupplier {
    ShortenerServiceBaseDescriptorSupplier() {}

    @java.lang.Override
    public com.google.protobuf.Descriptors.FileDescriptor getFileDescriptor() {
      return com.pingxin403.api.v1.ShortenerServiceProto.getDescriptor();
    }

    @java.lang.Override
    public com.google.protobuf.Descriptors.ServiceDescriptor getServiceDescriptor() {
      return getFileDescriptor().findServiceByName("ShortenerService");
    }
  }

  private static final class ShortenerServiceFileDescriptorSupplier
      extends ShortenerServiceBaseDescriptorSupplier {
    ShortenerServiceFileDescriptorSupplier() {}
  }

  private static final class ShortenerServiceMethodDescriptorSupplier
      extends ShortenerServiceBaseDescriptorSupplier
      implements io.grpc.protobuf.ProtoMethodDescriptorSupplier {
    private final java.lang.String methodName;

    ShortenerServiceMethodDescriptorSupplier(java.lang.String methodName) {
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
      synchronized (ShortenerServiceGrpc.class) {
        result = serviceDescriptor;
        if (result == null) {
          serviceDescriptor = result = io.grpc.ServiceDescriptor.newBuilder(SERVICE_NAME)
              .setSchemaDescriptor(new ShortenerServiceFileDescriptorSupplier())
              .addMethod(getCreateShortLinkMethod())
              .addMethod(getGetLinkInfoMethod())
              .addMethod(getDeleteShortLinkMethod())
              .build();
        }
      }
    }
    return result;
  }
}
