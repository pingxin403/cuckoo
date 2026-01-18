package client

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/pingxin403/cuckoo/apps/todo-service/gen/hellopb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// HelloClient wraps the gRPC client for Hello service
type HelloClient struct {
	client hellopb.HelloServiceClient
	conn   *grpc.ClientConn
}

// NewHelloClient creates a new Hello service client
// It reads the service address from the HELLO_SERVICE_ADDR environment variable
// If not set, it defaults to "localhost:9090"
func NewHelloClient() (*HelloClient, error) {
	// Get service address from environment variable
	addr := os.Getenv("HELLO_SERVICE_ADDR")
	if addr == "" {
		addr = "localhost:9090" // Default for local development
	}

	// Configure gRPC connection with retry policy
	conn, err := grpc.Dial(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{
			"methodConfig": [{
				"name": [{"service": "api.v1.HelloService"}],
				"retryPolicy": {
					"maxAttempts": 3,
					"initialBackoff": "0.1s",
					"maxBackoff": "1s",
					"backoffMultiplier": 2,
					"retryableStatusCodes": ["UNAVAILABLE", "DEADLINE_EXCEEDED"]
				}
			}]
		}`),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Hello service at %s: %w", addr, err)
	}

	client := hellopb.NewHelloServiceClient(conn)
	return &HelloClient{
		client: client,
		conn:   conn,
	}, nil
}

// SayHello calls the Hello service with the given name
// It includes timeout and error handling
func (c *HelloClient) SayHello(ctx context.Context, name string) (string, error) {
	// Set timeout for the request
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// Make the gRPC call
	req := &hellopb.HelloRequest{Name: name}
	resp, err := c.client.SayHello(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to call SayHello: %w", err)
	}

	return resp.Message, nil
}

// Close closes the gRPC connection
func (c *HelloClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
