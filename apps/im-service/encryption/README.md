# Message Encryption Module

This module provides AES-256-GCM encryption for IM messages with KMS integration and automatic key rotation.

## Features

- **AES-256-GCM Encryption**: Industry-standard authenticated encryption
- **KMS Integration**: Secure key management with external KMS
- **DEK Caching**: 1-hour TTL cache for performance
- **Automatic Key Rotation**: Keys rotate every 90 days
- **Backward Compatibility**: Old messages remain decryptable using key versioning

## Architecture

```
┌─────────────────┐
│  IM Service     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Encryption     │
│  Service        │
└────────┬────────┘
         │
    ┌────┴────┐
    │         │
    ▼         ▼
┌────────┐ ┌────────┐
│  DEK   │ │  KMS   │
│ Cache  │ │ Client │
└────────┘ └────────┘
```

## Usage

### Basic Encryption/Decryption

```go
import "github.com/pingxin403/cuckoo/apps/im-service/encryption"

// Create KMS client (implementation-specific)
kms := NewKMSClient(config)

// Create encryption service
config := encryption.DefaultConfig()
service := encryption.NewEncryptionService(kms, config)
defer service.Close()

// Encrypt message
plaintext := []byte("Hello, World!")
encrypted, err := service.Encrypt(plaintext)
if err != nil {
    log.Fatal(err)
}

// Decrypt message
decrypted, err := service.Decrypt(encrypted)
if err != nil {
    log.Fatal(err)
}
```

### Configuration

```go
config := encryption.Config{
    KMSURL:          "http://kms.default.svc.cluster.local:8080",
    MasterKeyID:     "master-key-1",
    DEKCacheTTL:     1 * time.Hour,
    KeyRotationDays: 90,
}
```

## Data Structures

### EncryptedMessage

```go
type EncryptedMessage struct {
    Ciphertext []byte  // Encrypted content + auth tag
    Nonce      []byte  // 12-byte nonce (IV)
    KeyID      string  // KMS key identifier
    KeyVersion int     // Key version for rotation
}
```

### DataEncryptionKey

```go
type DataEncryptionKey struct {
    KeyID      string
    Version    int
    Key        []byte    // 32 bytes for AES-256
    CreatedAt  time.Time
    ExpiresAt  time.Time
}
```

## Security Properties

1. **Authenticated Encryption**: GCM mode provides both confidentiality and authenticity
2. **Unique Nonces**: Each encryption uses a fresh random nonce
3. **Key Rotation**: Automatic rotation every 90 days reduces key exposure
4. **Secure Key Storage**: Keys never stored in plaintext, only in KMS
5. **Cache Security**: DEK cache has 1-hour TTL to limit exposure

## Performance

- **Encryption**: ~1-2ms per message (10KB)
- **Decryption**: ~1-2ms per message (10KB)
- **DEK Cache Hit**: <0.1ms
- **DEK Cache Miss**: ~10-50ms (KMS roundtrip)

## Testing

Run unit tests:
```bash
go test -v ./encryption
```

Run with coverage:
```bash
go test -coverprofile=coverage.out ./encryption
go tool cover -html=coverage.out
```

## Compliance

- **GDPR**: Encryption at rest for personal data
- **HIPAA**: AES-256 encryption meets requirements
- **PCI DSS**: Strong cryptography for cardholder data

## Key Rotation

Keys automatically rotate after 90 days:

1. New messages use new key version
2. Old messages remain decryptable with old key version
3. KMS maintains all key versions
4. Optional: Bulk re-encryption of old messages

## Limitations

- **Not E2EE**: Server-side encryption, not end-to-end
- **KMS Dependency**: Requires external KMS for key management
- **Performance**: Adds 1-2ms latency per message
- **Storage**: Adds ~40 bytes per message (nonce + metadata)

## Future Enhancements

- [ ] Support for multiple encryption algorithms
- [ ] Client-side encryption (E2EE mode)
- [ ] Hardware Security Module (HSM) integration
- [ ] Bulk re-encryption tool for key rotation
- [ ] Encryption metrics and monitoring
