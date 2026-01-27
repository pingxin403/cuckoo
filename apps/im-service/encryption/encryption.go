package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"sync"
	"time"
)

// EncryptionService provides message encryption using AES-256-GCM.
// Validates: Requirements 11.5, 11.6, 11.7
type EncryptionService struct {
	kms             KMSInterface
	dekCache        *DEKCache
	keyID           string
	keyRotationDays int
}

// KMSInterface defines the interface for Key Management Service operations.
type KMSInterface interface {
	GenerateDEK(keyID string) (*DataEncryptionKey, error)
	GetDEK(keyID string, version int) (*DataEncryptionKey, error)
	Close() error
}

// DataEncryptionKey represents a data encryption key.
type DataEncryptionKey struct {
	KeyID     string
	Version   int
	Key       []byte // 32 bytes for AES-256
	CreatedAt time.Time
	ExpiresAt time.Time
}

// DEKCache caches data encryption keys with TTL.
type DEKCache struct {
	keys map[string]*CachedDEK
	mu   sync.RWMutex
	ttl  time.Duration
}

// CachedDEK represents a cached data encryption key.
type CachedDEK struct {
	DEK       *DataEncryptionKey
	CachedAt  time.Time
	ExpiresAt time.Time
}

// EncryptedMessage represents an encrypted message.
type EncryptedMessage struct {
	Ciphertext []byte
	Nonce      []byte // 12 bytes for GCM
	AuthTag    []byte // 16 bytes for GCM (included in ciphertext)
	KeyID      string
	KeyVersion int
}

// Config contains configuration for the encryption service.
type Config struct {
	KMSURL          string
	MasterKeyID     string
	DEKCacheTTL     time.Duration
	KeyRotationDays int
}

// DefaultConfig returns default encryption configuration.
func DefaultConfig() Config {
	return Config{
		KMSURL:          "http://kms.default.svc.cluster.local:8080",
		MasterKeyID:     "master-key-1",
		DEKCacheTTL:     1 * time.Hour,
		KeyRotationDays: 90,
	}
}

// NewEncryptionService creates a new encryption service.
func NewEncryptionService(kms KMSInterface, config Config) *EncryptionService {
	return &EncryptionService{
		kms:             kms,
		dekCache:        NewDEKCache(config.DEKCacheTTL),
		keyID:           config.MasterKeyID,
		keyRotationDays: config.KeyRotationDays,
	}
}

// NewDEKCache creates a new DEK cache.
func NewDEKCache(ttl time.Duration) *DEKCache {
	cache := &DEKCache{
		keys: make(map[string]*CachedDEK),
		ttl:  ttl,
	}

	// Start cleanup goroutine
	go cache.cleanupExpired()

	return cache
}

// Encrypt encrypts a message using AES-256-GCM.
// Validates: Requirements 11.5, 11.6
func (s *EncryptionService) Encrypt(plaintext []byte) (*EncryptedMessage, error) {
	// Get or generate DEK
	dek, err := s.getDEK()
	if err != nil {
		return nil, fmt.Errorf("failed to get DEK: %w", err)
	}

	// Create AES cipher
	block, err := aes.NewCipher(dek.Key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random nonce (12 bytes for GCM)
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and authenticate
	// GCM.Seal appends the ciphertext and auth tag
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	return &EncryptedMessage{
		Ciphertext: ciphertext,
		Nonce:      nonce,
		KeyID:      dek.KeyID,
		KeyVersion: dek.Version,
	}, nil
}

// Decrypt decrypts a message using AES-256-GCM.
// Validates: Requirements 11.5, 11.6
func (s *EncryptionService) Decrypt(encrypted *EncryptedMessage) ([]byte, error) {
	// Get DEK from cache or KMS
	dek, err := s.getDEKByVersion(encrypted.KeyID, encrypted.KeyVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get DEK: %w", err)
	}

	// Create AES cipher
	block, err := aes.NewCipher(dek.Key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Decrypt and verify
	plaintext, err := gcm.Open(nil, encrypted.Nonce, encrypted.Ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// getDEK gets or generates a data encryption key.
func (s *EncryptionService) getDEK() (*DataEncryptionKey, error) {
	// Check cache first
	if dek := s.dekCache.Get(s.keyID); dek != nil {
		// Check if key needs rotation
		if time.Since(dek.CreatedAt) > time.Duration(s.keyRotationDays)*24*time.Hour {
			// Key is too old, generate new one
			return s.generateNewDEK()
		}
		return dek, nil
	}

	// Generate new DEK
	return s.generateNewDEK()
}

// getDEKByVersion gets a specific version of a DEK (for decryption).
func (s *EncryptionService) getDEKByVersion(keyID string, version int) (*DataEncryptionKey, error) {
	cacheKey := fmt.Sprintf("%s:%d", keyID, version)

	// Check cache first
	if dek := s.dekCache.Get(cacheKey); dek != nil {
		return dek, nil
	}

	// Fetch from KMS
	dek, err := s.kms.GetDEK(keyID, version)
	if err != nil {
		return nil, err
	}

	// Cache it
	s.dekCache.Put(cacheKey, dek)

	return dek, nil
}

// generateNewDEK generates a new data encryption key.
func (s *EncryptionService) generateNewDEK() (*DataEncryptionKey, error) {
	dek, err := s.kms.GenerateDEK(s.keyID)
	if err != nil {
		return nil, err
	}

	// Cache the new DEK
	s.dekCache.Put(s.keyID, dek)

	return dek, nil
}

// Get retrieves a DEK from cache.
func (c *DEKCache) Get(keyID string) *DataEncryptionKey {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cached, ok := c.keys[keyID]
	if !ok {
		return nil
	}

	// Check if expired
	if time.Now().After(cached.ExpiresAt) {
		return nil
	}

	return cached.DEK
}

// Put stores a DEK in cache.
func (c *DEKCache) Put(keyID string, dek *DataEncryptionKey) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.keys[keyID] = &CachedDEK{
		DEK:       dek,
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(c.ttl),
	}
}

// cleanupExpired removes expired DEKs from cache.
func (c *DEKCache) cleanupExpired() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for keyID, cached := range c.keys {
			if now.After(cached.ExpiresAt) {
				delete(c.keys, keyID)
			}
		}
		c.mu.Unlock()
	}
}

// Close closes the encryption service.
func (s *EncryptionService) Close() error {
	if s.kms != nil {
		return s.kms.Close()
	}
	return nil
}
