package encryption

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"sync"
	"testing"
	"time"
)

// mockKMS is a mock implementation of KMSInterface for testing.
type mockKMS struct {
	keys     map[string]*DataEncryptionKey
	mu       sync.Mutex
	version  int
	failNext bool
}

func newMockKMS() *mockKMS {
	return &mockKMS{
		keys:    make(map[string]*DataEncryptionKey),
		version: 1,
	}
}

func (m *mockKMS) GenerateDEK(keyID string) (*DataEncryptionKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failNext {
		m.failNext = false
		return nil, fmt.Errorf("KMS error")
	}

	// Generate random 32-byte key for AES-256
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}

	dek := &DataEncryptionKey{
		KeyID:     keyID,
		Version:   m.version,
		Key:       key,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(90 * 24 * time.Hour),
	}

	m.keys[fmt.Sprintf("%s:%d", keyID, m.version)] = dek
	m.version++

	return dek, nil
}

func (m *mockKMS) GetDEK(keyID string, version int) (*DataEncryptionKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%d", keyID, version)
	dek, ok := m.keys[key]
	if !ok {
		return nil, fmt.Errorf("DEK not found: %s", key)
	}

	return dek, nil
}

func (m *mockKMS) Close() error {
	return nil
}

func (m *mockKMS) SetFailNext(fail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failNext = fail
}

// TestEncryptDecrypt tests basic encryption and decryption.
func TestEncryptDecrypt(t *testing.T) {
	kms := newMockKMS()
	config := DefaultConfig()
	config.DEKCacheTTL = 1 * time.Hour

	service := NewEncryptionService(kms, config)
	defer func() { _ = service.Close() }()

	plaintext := []byte("Hello, World! This is a test message.")

	// Encrypt
	encrypted, err := service.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	if encrypted.Ciphertext == nil {
		t.Error("Ciphertext is nil")
	}
	if len(encrypted.Nonce) != 12 {
		t.Errorf("Expected nonce length 12, got %d", len(encrypted.Nonce))
	}
	if encrypted.KeyID == "" {
		t.Error("KeyID is empty")
	}
	if encrypted.KeyVersion <= 0 {
		t.Errorf("Expected positive key version, got %d", encrypted.KeyVersion)
	}

	// Decrypt
	decrypted, err := service.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Errorf("Decrypted text doesn't match original.\nExpected: %s\nGot: %s", plaintext, decrypted)
	}
}

// TestEncryptDecrypt_EmptyMessage tests encryption of empty message.
func TestEncryptDecrypt_EmptyMessage(t *testing.T) {
	kms := newMockKMS()
	service := NewEncryptionService(kms, DefaultConfig())
	defer func() { _ = service.Close() }()

	plaintext := []byte("")

	encrypted, err := service.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := service.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Error("Decrypted empty message doesn't match")
	}
}

// TestEncryptDecrypt_LargeMessage tests encryption of large message.
func TestEncryptDecrypt_LargeMessage(t *testing.T) {
	kms := newMockKMS()
	service := NewEncryptionService(kms, DefaultConfig())
	defer func() { _ = service.Close() }()

	// Create 10KB message
	plaintext := make([]byte, 10000)
	for i := range plaintext {
		plaintext[i] = byte(i % 256)
	}

	encrypted, err := service.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := service.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Error("Decrypted large message doesn't match")
	}
}

// TestDEKCache tests DEK caching functionality.
func TestDEKCache(t *testing.T) {
	kms := newMockKMS()
	config := DefaultConfig()
	config.DEKCacheTTL = 100 * time.Millisecond

	service := NewEncryptionService(kms, config)
	defer func() { _ = service.Close() }()

	// First encryption should generate DEK
	_, err := service.Encrypt([]byte("test1"))
	if err != nil {
		t.Fatalf("First encrypt failed: %v", err)
	}

	// Second encryption should use cached DEK
	_, err = service.Encrypt([]byte("test2"))
	if err != nil {
		t.Fatalf("Second encrypt failed: %v", err)
	}

	// Both should use same DEK version
	if kms.version != 2 {
		t.Errorf("Expected version 2 (one DEK generated), got %d", kms.version)
	}
}

// TestDEKCache_Expiration tests DEK cache expiration.
func TestDEKCache_Expiration(t *testing.T) {
	kms := newMockKMS()
	config := DefaultConfig()
	config.DEKCacheTTL = 50 * time.Millisecond

	service := NewEncryptionService(kms, config)
	defer func() { _ = service.Close() }()

	// First encryption
	_, err := service.Encrypt([]byte("test1"))
	if err != nil {
		t.Fatalf("First encrypt failed: %v", err)
	}

	// Wait for cache to expire
	time.Sleep(100 * time.Millisecond)

	// Second encryption should generate new DEK
	_, err = service.Encrypt([]byte("test2"))
	if err != nil {
		t.Fatalf("Second encrypt failed: %v", err)
	}

	// Should have generated 2 DEKs
	if kms.version != 3 {
		t.Errorf("Expected version 3 (two DEKs generated), got %d", kms.version)
	}
}

// TestKeyRotation tests automatic key rotation after 90 days.
func TestKeyRotation(t *testing.T) {
	kms := newMockKMS()
	config := DefaultConfig()
	config.KeyRotationDays = 0 // Force immediate rotation for testing

	service := NewEncryptionService(kms, config)
	defer func() { _ = service.Close() }()

	// First encryption
	encrypted1, err := service.Encrypt([]byte("test1"))
	if err != nil {
		t.Fatalf("First encrypt failed: %v", err)
	}

	// Second encryption should rotate key
	encrypted2, err := service.Encrypt([]byte("test2"))
	if err != nil {
		t.Fatalf("Second encrypt failed: %v", err)
	}

	// Should use different key versions
	if encrypted1.KeyVersion == encrypted2.KeyVersion {
		t.Error("Expected different key versions after rotation")
	}

	// Both should be decryptable
	_, err = service.Decrypt(encrypted1)
	if err != nil {
		t.Errorf("Failed to decrypt message with old key: %v", err)
	}

	_, err = service.Decrypt(encrypted2)
	if err != nil {
		t.Errorf("Failed to decrypt message with new key: %v", err)
	}
}

// TestDecrypt_InvalidKey tests decryption with invalid key version.
func TestDecrypt_InvalidKey(t *testing.T) {
	kms := newMockKMS()
	service := NewEncryptionService(kms, DefaultConfig())
	defer func() { _ = service.Close() }()

	encrypted := &EncryptedMessage{
		Ciphertext: []byte("invalid"),
		Nonce:      make([]byte, 12),
		KeyID:      "master-key-1",
		KeyVersion: 999, // Non-existent version
	}

	_, err := service.Decrypt(encrypted)
	if err == nil {
		t.Error("Expected error for invalid key version")
	}
}

// TestDecrypt_TamperedCiphertext tests decryption with tampered ciphertext.
func TestDecrypt_TamperedCiphertext(t *testing.T) {
	kms := newMockKMS()
	service := NewEncryptionService(kms, DefaultConfig())
	defer func() { _ = service.Close() }()

	plaintext := []byte("test message")

	// Encrypt
	encrypted, err := service.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Tamper with ciphertext
	if len(encrypted.Ciphertext) > 0 {
		encrypted.Ciphertext[0] ^= 0xFF
	}

	// Decrypt should fail (authentication failure)
	_, err = service.Decrypt(encrypted)
	if err == nil {
		t.Error("Expected error for tampered ciphertext")
	}
}

// TestKMSFailure tests handling of KMS failures.
func TestKMSFailure(t *testing.T) {
	kms := newMockKMS()
	service := NewEncryptionService(kms, DefaultConfig())
	defer func() { _ = service.Close() }()

	// Configure KMS to fail
	kms.SetFailNext(true)

	_, err := service.Encrypt([]byte("test"))
	if err == nil {
		t.Error("Expected error when KMS fails")
	}
}

// TestConcurrentEncryption tests concurrent encryption operations.
func TestConcurrentEncryption(t *testing.T) {
	kms := newMockKMS()
	service := NewEncryptionService(kms, DefaultConfig())
	defer func() { _ = service.Close() }()

	const numGoroutines = 10
	const numOperations = 100

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*numOperations)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				plaintext := []byte(fmt.Sprintf("message-%d-%d", id, j))

				encrypted, err := service.Encrypt(plaintext)
				if err != nil {
					errors <- err
					continue
				}

				decrypted, err := service.Decrypt(encrypted)
				if err != nil {
					errors <- err
					continue
				}

				if !bytes.Equal(plaintext, decrypted) {
					errors <- fmt.Errorf("mismatch for message-%d-%d", id, j)
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent operation failed: %v", err)
	}
}

// TestDEKCacheCleanup tests automatic cleanup of expired DEKs.
func TestDEKCacheCleanup(t *testing.T) {
	cache := NewDEKCache(50 * time.Millisecond)

	// Add some DEKs
	for i := 0; i < 5; i++ {
		dek := &DataEncryptionKey{
			KeyID:   fmt.Sprintf("key-%d", i),
			Version: i,
			Key:     make([]byte, 32),
		}
		cache.Put(fmt.Sprintf("key-%d", i), dek)
	}

	// Verify all are cached
	cache.mu.RLock()
	count := len(cache.keys)
	cache.mu.RUnlock()

	if count != 5 {
		t.Errorf("Expected 5 cached DEKs, got %d", count)
	}

	// Wait for expiration and cleanup
	time.Sleep(200 * time.Millisecond)

	// Verify cleanup happened (may not be immediate due to cleanup interval)
	// Just verify Get returns nil for expired keys
	for i := 0; i < 5; i++ {
		dek := cache.Get(fmt.Sprintf("key-%d", i))
		if dek != nil {
			t.Errorf("Expected expired DEK to be nil, got %v", dek)
		}
	}
}
