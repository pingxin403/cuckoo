import { HelloRequest, HelloResponse } from "../gen/hello";

// Simple wrapper for Hello Service using fetch
export class HelloServiceClient {
  private baseUrl: string;

  constructor(baseUrl: string = "/api/hello") {
    this.baseUrl = baseUrl;
  }

  async sayHello(request: HelloRequest): Promise<HelloResponse> {
    const url = `${this.baseUrl}/api.v1.HelloService/SayHello`;
    
    // Encode the request
    const requestBytes = HelloRequest.encode(request).finish();
    
    try {
      const response = await fetch(url, {
        method: "POST",
        headers: {
          "Content-Type": "application/grpc-web+proto",
          "X-Grpc-Web": "1",
        },
        body: requestBytes,
      });

      if (!response.ok) {
        throw new Error(`gRPC request failed: ${response.statusText}`);
      }

      const responseBytes = new Uint8Array(await response.arrayBuffer());
      
      // Skip the gRPC-Web frame header (5 bytes: 1 byte flags + 4 bytes length)
      const messageBytes = responseBytes.slice(5);
      
      return HelloResponse.decode(messageBytes);
    } catch (error) {
      console.error("Hello service error:", error);
      throw error;
    }
  }
}

export const helloClient = new HelloServiceClient();

