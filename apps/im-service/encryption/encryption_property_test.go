package encryption

import (
	"bytes"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// TestProperty_KeyRotation tests Property 12: Encryption Key Rotation
// Validates: Requirements 14.5, Property 12
//
// Property 12: Encryption Key Rotation
// - Keys rotate after 90 days
// - Old messages remain decryptable with old keys
// - New messages use new keys after rotation
func TestProperty_KeyRotation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random plaintext messages
		message1 := rapid.SliceOfN(rapid.Byte(), 1, 1000).Draw(t, "message1")
		message2 := rapid.SliceOfN(rapid.Byte(), 1, 1000).Draw(t, "message2")

		// Create encryption service with immediate rotation (0 days)
		kms := newMockKMS()
		config := DefaultConfig()
		config.KeyRotationDays = 0                // Force immediate rotation
		config.DEKCacheTTL = 1 * time.Millisecond // Short cache TTL

		service := NewEncryptionService(kms, config)
		defer func() { _ = service.Close() }()

		// Encrypt first message (will use key version 1)
		encrypted1, err := service.Encrypt(message1)
		if err != nil {
			t.Fatalf("Failed to encrypt message1: %v", err)
		}

		// Wait for cache to expire to force rotation
		time.Sleep(10 * time.Millisecond)

		// Encrypt second message (should use key version 2 due to rotation)
		encrypted2, err := service.Encrypt(message2)
		if err != nil {
			t.Fatalf("Failed to encrypt message2: %v", err)
		}

		// Property: New messages use new keys after rotation
		if encrypted1.KeyVersion == encrypted2.KeyVersion {
			t.Fatalf("Expected different key versions after rotation, got %d and %d",
				encrypted1.KeyVersion, encrypted2.KeyVersion)
		}

		// Property: Old messages remain decryptable with old keys
		decrypted1, err := service.Decrypt(encrypted1)
		if err != nil {
			t.Fatalf("Failed to decrypt message1 with old key: %v", err)
		}
		if !bytes.Equal(message1, decrypted1) {
			t.Fatalf("Decrypted message1 doesn't match original")
		}

		// Property: New messages are decryptable with new keys
		decrypted2, err := service.Decrypt(encrypted2)
		if err != nil {
			t.Fatalf("Failed to decrypt message2 with new key: %v", err)
		}
		if !bytes.Equal(message2, decrypted2) {
			t.Fatalf("Decrypted message2 doesn't match original")
		}
	})
}

// TestProperty_KeyRotationAfter90Days tests that keys rotate after 90 days
func TestProperty_KeyRotationAfter90Days(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random plaintext
		plaintext := rapid.SliceOfN(rapid.Byte(), 1, 1000).Draw(t, "plaintext")

		// Create encryption service with 90-day rotation
		kms := newMockKMS()
		config := DefaultConfig()
		config.KeyRotationDays = 90
		config.DEKCacheTTL = 1 * time.Hour

		service := NewEncryptionService(kms, config)
		defer func() { _ = service.Close() }()

		// Encrypt message
		encrypted, err := service.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("Failed to encrypt: %v", err)
		}

		// Get the DEK that was used
		dek := service.dekCache.Get(service.keyID)
		if dek == nil {
			t.Fatal("DEK not found in cache")
		}

		// Simulate 91 days passing by modifying the DEK's creation time
		dek.CreatedAt = time.Now().Add(-91 * 24 * time.Hour)

		// Encrypt another message - should trigger rotation
		plaintext2 := rapid.SliceOfN(rapid.Byte(), 1, 1000).Draw(t, "plaintext2")
		encrypted2, err := service.Encrypt(plaintext2)
		if err != nil {
			t.Fatalf("Failed to encrypt after rotation: %v", err)
		}

		// Property: Key version should be different after 90 days
		if encrypted.KeyVersion == encrypted2.KeyVersion {
			t.Fatalf("Expected key rotation after 90 days, but versions are the same: %d",
				encrypted.KeyVersion)
		}

		// Property: Both messages should still be decryptable
		decrypted, err := service.Decrypt(encrypted)
		if err != nil {
			t.Fatalf("Failed to decrypt old message: %v", err)
		}
		if !bytes.Equal(plaintext, decrypted) {
			t.Fatal("Old message decryption failed")
		}

		decrypted2, err := service.Decrypt(encrypted2)
		if err != nil {
			t.Fatalf("Failed to decrypt new message: %v", err)
		}
		if !bytes.Equal(plaintext2, decrypted2) {
			t.Fatal("New message decryption failed")
		}
	})
}

// TestProperty_MultipleRotations tests multiple key rotations
func TestProperty_MultipleRotations(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random number of messages (3-10)
		numMessages := rapid.IntRange(3, 10).Draw(t, "numMessages")

		kms := newMockKMS()
		config := DefaultConfig()
		config.KeyRotationDays = 0 // Force rotation on each encrypt
		config.DEKCacheTTL = 1 * time.Millisecond

		service := NewEncryptionService(kms, config)
		defer func() { _ = service.Close() }()

		// Encrypt multiple messages with rotations
		type messageData struct {
			plaintext []byte
			encrypted *EncryptedMessage
		}
		messages := make([]messageData, numMessages)

		for i := 0; i < numMessages; i++ {
			plaintext := rapid.SliceOfN(rapid.Byte(), 1, 1000).Draw(t, "plaintext")

			// Wait for cache expiry to force rotation
			if i > 0 {
				time.Sleep(10 * time.Millisecond)
			}

			encrypted, err := service.Encrypt(plaintext)
			if err != nil {
				t.Fatalf("Failed to encrypt message %d: %v", i, err)
			}

			messages[i] = messageData{
				plaintext: plaintext,
				encrypted: encrypted,
			}
		}

		// Property: All messages should be decryptable regardless of key version
		for i, msg := range messages {
			decrypted, err := service.Decrypt(msg.encrypted)
			if err != nil {
				t.Fatalf("Failed to decrypt message %d: %v", i, err)
			}
			if !bytes.Equal(msg.plaintext, decrypted) {
				t.Fatalf("Message %d decryption mismatch", i)
			}
		}

		// Property: Key versions should be monotonically increasing
		for i := 1; i < len(messages); i++ {
			if messages[i].encrypted.KeyVersion <= messages[i-1].encrypted.KeyVersion {
				t.Fatalf("Key versions not monotonically increasing: %d -> %d",
					messages[i-1].encrypted.KeyVersion, messages[i].encrypted.KeyVersion)
			}
		}
	})
}

// TestProperty_KeyRotationPreservesEncryptionStrength tests that rotation doesn't weaken encryption
func TestProperty_KeyRotationPreservesEncryptionStrength(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		plaintext := rapid.SliceOfN(rapid.Byte(), 1, 1000).Draw(t, "plaintext")

		kms := newMockKMS()
		config := DefaultConfig()
		config.KeyRotationDays = 0
		config.DEKCacheTTL = 1 * time.Millisecond

		service := NewEncryptionService(kms, config)
		defer func() { _ = service.Close() }()

		// Encrypt with first key
		encrypted1, err := service.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("Failed to encrypt with key 1: %v", err)
		}

		// Force rotation
		time.Sleep(10 * time.Millisecond)

		// Encrypt with second key
		encrypted2, err := service.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("Failed to encrypt with key 2: %v", err)
		}

		// Property: Both encryptions should use 12-byte nonces (GCM standard)
		if len(encrypted1.Nonce) != 12 {
			t.Fatalf("Expected 12-byte nonce for key 1, got %d", len(encrypted1.Nonce))
		}
		if len(encrypted2.Nonce) != 12 {
			t.Fatalf("Expected 12-byte nonce for key 2, got %d", len(encrypted2.Nonce))
		}

		// Property: Ciphertexts should be different (different nonces)
		if bytes.Equal(encrypted1.Ciphertext, encrypted2.Ciphertext) {
			t.Fatal("Ciphertexts should be different due to different nonces")
		}

		// Property: Both should decrypt correctly
		decrypted1, err := service.Decrypt(encrypted1)
		if err != nil {
			t.Fatalf("Failed to decrypt with key 1: %v", err)
		}
		if !bytes.Equal(plaintext, decrypted1) {
			t.Fatal("Decryption with key 1 failed")
		}

		decrypted2, err := service.Decrypt(encrypted2)
		if err != nil {
			t.Fatalf("Failed to decrypt with key 2: %v", err)
		}
		if !bytes.Equal(plaintext, decrypted2) {
			t.Fatal("Decryption with key 2 failed")
		}
	})
}

// TestProperty_OldKeysRemainAccessible tests that old keys remain accessible after rotation
func TestProperty_OldKeysRemainAccessible(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate multiple messages
		numMessages := rapid.IntRange(2, 5).Draw(t, "numMessages")

		kms := newMockKMS()
		config := DefaultConfig()
		config.KeyRotationDays = 0
		config.DEKCacheTTL = 1 * time.Millisecond

		service := NewEncryptionService(kms, config)
		defer func() { _ = service.Close() }()

		// Encrypt messages with different key versions
		encryptedMessages := make([]*EncryptedMessage, numMessages)
		plaintexts := make([][]byte, numMessages)

		for i := 0; i < numMessages; i++ {
			plaintext := rapid.SliceOfN(rapid.Byte(), 1, 1000).Draw(t, "plaintext")
			plaintexts[i] = plaintext

			if i > 0 {
				time.Sleep(10 * time.Millisecond) // Force rotation
			}

			encrypted, err := service.Encrypt(plaintext)
			if err != nil {
				t.Fatalf("Failed to encrypt message %d: %v", i, err)
			}
			encryptedMessages[i] = encrypted
		}

		// Property: All old messages should still be decryptable
		// even after multiple rotations
		for i := 0; i < numMessages; i++ {
			decrypted, err := service.Decrypt(encryptedMessages[i])
			if err != nil {
				t.Fatalf("Failed to decrypt message %d (version %d): %v",
					i, encryptedMessages[i].KeyVersion, err)
			}
			if !bytes.Equal(plaintexts[i], decrypted) {
				t.Fatalf("Message %d decryption mismatch", i)
			}
		}
	})
}
