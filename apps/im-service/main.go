package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	// Placeholder main.go for im-service
	// The actual IM Service implementation (Task 9) is not yet started.
	// Currently implemented components:
	// - Sequence Generator (Task 5) - see sequence/ package
	// - Registry Client (Task 6) - see registry/ package

	port := os.Getenv("PORT")
	if port == "" {
		port = "9094"
	}

	log.Printf("im-service placeholder - port %s", port)
	log.Println("Note: Full IM Service implementation (message routing) is pending Task 9")
	fmt.Println("Implemented components:")
	fmt.Println("  - Sequence Generator (sequence/)")
	fmt.Println("  - Registry Client (registry/)")
}
