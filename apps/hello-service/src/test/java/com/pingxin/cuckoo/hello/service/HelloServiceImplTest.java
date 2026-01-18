package com.pingxin.cuckoo.hello.service;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.Mockito.verify;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.ArgumentCaptor;
import org.mockito.Captor;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import com.myorg.api.v1.HelloRequest;
import com.myorg.api.v1.HelloResponse;

import io.grpc.stub.StreamObserver;

/**
 * Unit tests for HelloServiceImpl
 *
 * <p>Tests the core business logic of the Hello service including: - Personalized greetings with
 * valid names - Default greeting for empty/null names - Whitespace handling - Response format
 * validation
 */
@ExtendWith(MockitoExtension.class)
@DisplayName("HelloService Unit Tests")
class HelloServiceImplTest {

  private HelloServiceImpl helloService;

  @Mock private StreamObserver<HelloResponse> responseObserver;

  @Captor private ArgumentCaptor<HelloResponse> responseCaptor;

  @BeforeEach
  void setUp() {
    helloService = new HelloServiceImpl();
  }

  @Test
  @DisplayName("Should return personalized greeting for valid name")
  void shouldReturnPersonalizedGreetingForValidName() {
    // Given
    HelloRequest request = HelloRequest.newBuilder().setName("Alice").build();

    // When
    helloService.sayHello(request, responseObserver);

    // Then
    verify(responseObserver).onNext(responseCaptor.capture());
    verify(responseObserver).onCompleted();

    HelloResponse response = responseCaptor.getValue();
    assertThat(response.getMessage()).isEqualTo("Hello, Alice!").contains("Alice");
  }

  @Test
  @DisplayName("Should return default greeting for empty name")
  void shouldReturnDefaultGreetingForEmptyName() {
    // Given
    HelloRequest request = HelloRequest.newBuilder().setName("").build();

    // When
    helloService.sayHello(request, responseObserver);

    // Then
    verify(responseObserver).onNext(responseCaptor.capture());
    verify(responseObserver).onCompleted();

    HelloResponse response = responseCaptor.getValue();
    assertThat(response.getMessage()).isEqualTo("Hello, World!");
  }

  @Test
  @DisplayName("Should return default greeting for null name")
  void shouldReturnDefaultGreetingForNullName() {
    // Given
    HelloRequest request = HelloRequest.newBuilder().build(); // name is null by default

    // When
    helloService.sayHello(request, responseObserver);

    // Then
    verify(responseObserver).onNext(responseCaptor.capture());
    verify(responseObserver).onCompleted();

    HelloResponse response = responseCaptor.getValue();
    assertThat(response.getMessage()).isEqualTo("Hello, World!");
  }

  @Test
  @DisplayName("Should return default greeting for whitespace-only name")
  void shouldReturnDefaultGreetingForWhitespaceOnlyName() {
    // Given
    HelloRequest request = HelloRequest.newBuilder().setName("   ").build();

    // When
    helloService.sayHello(request, responseObserver);

    // Then
    verify(responseObserver).onNext(responseCaptor.capture());
    verify(responseObserver).onCompleted();

    HelloResponse response = responseCaptor.getValue();
    assertThat(response.getMessage()).isEqualTo("Hello, World!");
  }

  @Test
  @DisplayName("Should trim whitespace from name")
  void shouldTrimWhitespaceFromName() {
    // Given
    HelloRequest request = HelloRequest.newBuilder().setName("  Bob  ").build();

    // When
    helloService.sayHello(request, responseObserver);

    // Then
    verify(responseObserver).onNext(responseCaptor.capture());
    verify(responseObserver).onCompleted();

    HelloResponse response = responseCaptor.getValue();
    assertThat(response.getMessage()).isEqualTo("Hello, Bob!").doesNotContain("  ");
  }

  @Test
  @DisplayName("Should handle names with special characters")
  void shouldHandleNamesWithSpecialCharacters() {
    // Given
    HelloRequest request = HelloRequest.newBuilder().setName("José García").build();

    // When
    helloService.sayHello(request, responseObserver);

    // Then
    verify(responseObserver).onNext(responseCaptor.capture());
    verify(responseObserver).onCompleted();

    HelloResponse response = responseCaptor.getValue();
    assertThat(response.getMessage()).isEqualTo("Hello, José García!").contains("José García");
  }

  @Test
  @DisplayName("Should handle very long names")
  void shouldHandleVeryLongNames() {
    // Given
    String longName = "A".repeat(1000);
    HelloRequest request = HelloRequest.newBuilder().setName(longName).build();

    // When
    helloService.sayHello(request, responseObserver);

    // Then
    verify(responseObserver).onNext(responseCaptor.capture());
    verify(responseObserver).onCompleted();

    HelloResponse response = responseCaptor.getValue();
    assertThat(response.getMessage()).startsWith("Hello, ").endsWith("!").contains(longName);
  }

  @Test
  @DisplayName("Should handle names with numbers")
  void shouldHandleNamesWithNumbers() {
    // Given
    HelloRequest request = HelloRequest.newBuilder().setName("User123").build();

    // When
    helloService.sayHello(request, responseObserver);

    // Then
    verify(responseObserver).onNext(responseCaptor.capture());
    verify(responseObserver).onCompleted();

    HelloResponse response = responseCaptor.getValue();
    assertThat(response.getMessage()).isEqualTo("Hello, User123!").contains("User123");
  }

  @Test
  @DisplayName("Should handle single character name")
  void shouldHandleSingleCharacterName() {
    // Given
    HelloRequest request = HelloRequest.newBuilder().setName("X").build();

    // When
    helloService.sayHello(request, responseObserver);

    // Then
    verify(responseObserver).onNext(responseCaptor.capture());
    verify(responseObserver).onCompleted();

    HelloResponse response = responseCaptor.getValue();
    assertThat(response.getMessage()).isEqualTo("Hello, X!");
  }

  @Test
  @DisplayName("Should always call onCompleted after onNext")
  void shouldAlwaysCallOnCompletedAfterOnNext() {
    // Given
    HelloRequest request = HelloRequest.newBuilder().setName("Test").build();

    // When
    helloService.sayHello(request, responseObserver);

    // Then
    verify(responseObserver).onNext(responseCaptor.capture());
    verify(responseObserver).onCompleted();
  }
}
