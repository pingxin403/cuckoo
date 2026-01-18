package com.pingxin.cuckoo.hello.service;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.Mockito.*;

import org.mockito.ArgumentCaptor;

import com.myorg.api.v1.HelloRequest;
import com.myorg.api.v1.HelloResponse;

import io.grpc.stub.StreamObserver;
import net.jqwik.api.*;

/**
 * Property-based tests for HelloServiceImpl.
 *
 * <p>These tests verify properties that should hold for all inputs, using jqwik to generate random
 * test data.
 *
 * <p>Property 1: Hello Service Name Inclusion - For any non-empty name, the response message should
 * contain that name
 */
class HelloServicePropertyTest {

  /**
   * Property 1: Hello Service Name Inclusion
   *
   * <p>For any non-empty name provided in the request, the response message should contain that
   * name.
   *
   * <p>This property ensures that the service correctly incorporates the user's name into the
   * greeting message.
   */
  @Property
  @Label("Response message should contain the provided name")
  void responseContainsProvidedName(@ForAll("nonEmptyNames") String name) {
    // Arrange
    HelloServiceImpl helloService = new HelloServiceImpl();
    HelloRequest request = HelloRequest.newBuilder().setName(name).build();

    @SuppressWarnings("unchecked")
    StreamObserver<HelloResponse> responseObserver = mock(StreamObserver.class);

    // Act
    helloService.sayHello(request, responseObserver);

    // Assert
    ArgumentCaptor<HelloResponse> captor = ArgumentCaptor.forClass(HelloResponse.class);
    verify(responseObserver).onNext(captor.capture());
    verify(responseObserver).onCompleted();
    verify(responseObserver, never()).onError(any());

    HelloResponse response = captor.getValue();
    assertThat(response.getMessage())
        .as("Response message should contain the name '%s'", name)
        .contains(name);
  }

  /**
   * Property: Response is always non-empty
   *
   * <p>For any input (including empty strings), the service should always return a non-empty
   * response message.
   */
  @Property
  @Label("Response message should never be empty")
  void responseIsNeverEmpty(@ForAll String name) {
    // Arrange
    HelloServiceImpl helloService = new HelloServiceImpl();
    HelloRequest request = HelloRequest.newBuilder().setName(name).build();

    @SuppressWarnings("unchecked")
    StreamObserver<HelloResponse> responseObserver = mock(StreamObserver.class);

    // Act
    helloService.sayHello(request, responseObserver);

    // Assert
    ArgumentCaptor<HelloResponse> captor = ArgumentCaptor.forClass(HelloResponse.class);
    verify(responseObserver).onNext(captor.capture());
    verify(responseObserver).onCompleted();

    HelloResponse response = captor.getValue();
    assertThat(response.getMessage()).as("Response message should never be empty").isNotEmpty();
  }

  /**
   * Property: Service always completes successfully
   *
   * <p>For any input, the service should always call onCompleted() and never call onError().
   */
  @Property
  @Label("Service should always complete successfully")
  void serviceAlwaysCompletesSuccessfully(@ForAll String name) {
    // Arrange
    HelloServiceImpl helloService = new HelloServiceImpl();
    HelloRequest request = HelloRequest.newBuilder().setName(name).build();

    @SuppressWarnings("unchecked")
    StreamObserver<HelloResponse> responseObserver = mock(StreamObserver.class);

    // Act
    helloService.sayHello(request, responseObserver);

    // Assert
    verify(responseObserver).onNext(any(HelloResponse.class));
    verify(responseObserver).onCompleted();
    verify(responseObserver, never()).onError(any());
  }

  /**
   * Property: Response length is reasonable
   *
   * <p>The response message should not be excessively long, even for very long input names.
   */
  @Property
  @Label("Response length should be reasonable")
  void responseLengthIsReasonable(@ForAll("longNames") String name) {
    // Arrange
    HelloServiceImpl helloService = new HelloServiceImpl();
    HelloRequest request = HelloRequest.newBuilder().setName(name).build();

    @SuppressWarnings("unchecked")
    StreamObserver<HelloResponse> responseObserver = mock(StreamObserver.class);

    // Act
    helloService.sayHello(request, responseObserver);

    // Assert
    ArgumentCaptor<HelloResponse> captor = ArgumentCaptor.forClass(HelloResponse.class);
    verify(responseObserver).onNext(captor.capture());

    HelloResponse response = captor.getValue();
    // Response should not be more than 10x the input length
    assertThat(response.getMessage().length())
        .as("Response should not be excessively long")
        .isLessThan(name.length() * 10 + 100);
  }

  /**
   * Property: Special characters are handled safely
   *
   * <p>The service should handle names with special characters without errors or unexpected
   * behavior.
   */
  @Property
  @Label("Service should handle special characters safely")
  void handlesSpecialCharactersSafely(@ForAll("namesWithSpecialChars") String name) {
    // Arrange
    HelloServiceImpl helloService = new HelloServiceImpl();
    HelloRequest request = HelloRequest.newBuilder().setName(name).build();

    @SuppressWarnings("unchecked")
    StreamObserver<HelloResponse> responseObserver = mock(StreamObserver.class);

    // Act
    helloService.sayHello(request, responseObserver);

    // Assert
    ArgumentCaptor<HelloResponse> captor = ArgumentCaptor.forClass(HelloResponse.class);
    verify(responseObserver).onNext(captor.capture());
    verify(responseObserver).onCompleted();
    verify(responseObserver, never()).onError(any());

    HelloResponse response = captor.getValue();
    assertThat(response.getMessage()).isNotEmpty();
  }

  // Arbitraries (data generators)

  @Provide
  Arbitrary<String> nonEmptyNames() {
    return Arbitraries.strings().alpha().ofMinLength(1).ofMaxLength(50);
  }

  @Provide
  Arbitrary<String> longNames() {
    return Arbitraries.strings().alpha().ofMinLength(100).ofMaxLength(1000);
  }

  @Provide
  Arbitrary<String> namesWithSpecialChars() {
    return Arbitraries.strings()
        .withCharRange('!', '~') // All printable ASCII characters
        .ofMinLength(1)
        .ofMaxLength(50);
  }
}
