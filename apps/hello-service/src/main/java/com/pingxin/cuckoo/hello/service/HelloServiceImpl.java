package com.pingxin.cuckoo.hello.service;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import com.myorg.api.v1.HelloRequest;
import com.myorg.api.v1.HelloResponse;
import com.myorg.api.v1.HelloServiceGrpc;

import io.grpc.stub.StreamObserver;
import net.devh.boot.grpc.server.service.GrpcService;

/**
 * HelloServiceImpl 实现 gRPC Hello 服务
 *
 * <p>该服务接收用户姓名并返回个性化的问候消息。 如果未提供姓名或姓名为空，则返回默认问候消息。
 */
@GrpcService
public class HelloServiceImpl extends HelloServiceGrpc.HelloServiceImplBase {

  private static final Logger logger = LoggerFactory.getLogger(HelloServiceImpl.class);
  private static final String DEFAULT_GREETING = "Hello, World!";

  /**
   * 实现 SayHello RPC 方法
   *
   * @param request 包含用户姓名的请求
   * @param responseObserver 用于发送响应的观察者
   */
  @Override
  public void sayHello(HelloRequest request, StreamObserver<HelloResponse> responseObserver) {
    String name = request.getName();
    String message;

    // 如果名字为空或仅包含空格，返回默认问候
    if (name == null || name.trim().isEmpty()) {
      message = DEFAULT_GREETING;
      logger.debug("Received empty name, returning default greeting");
    } else {
      message = "Hello, " + name.trim() + "!";
      logger.debug("Received name: {}, returning personalized greeting", name);
    }

    HelloResponse response = HelloResponse.newBuilder().setMessage(message).build();

    responseObserver.onNext(response);
    responseObserver.onCompleted();

    logger.info("Processed hello request - name: '{}', message: '{}'", name, message);
  }
}
