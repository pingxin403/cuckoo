package com.pingxin403.cuckoo.hello;

import org.junit.jupiter.api.Test;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.context.TestPropertySource;

@SpringBootTest(webEnvironment = SpringBootTest.WebEnvironment.NONE)
@TestPropertySource(
    properties = {
      "grpc.server.port=0" // Use random port to avoid conflicts
    })
class HelloServiceApplicationTests {

  @Test
  void contextLoads() {}
}
