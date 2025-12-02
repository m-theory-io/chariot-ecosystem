package chariot

import (
	"context"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"runtime"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/pbkdf2"

	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/logs"
	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/vault"
)

// Global CryptoManager instance
var globalCryptoManager *CryptoManager
var cryptoInitOnce sync.Once

// CryptoManager handles all cryptographic operations using the configured secret provider
type CryptoManager struct {
	logger     *logs.ZapLogger
	keyCache   map[string]*cachedKey
	cacheMutex sync.RWMutex
}

// Cached key with expiration
type cachedKey struct {
	keyData   []byte
	expiresAt time.Time
}

// Secure memory utilities
func SecureZero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

func SecureCopy(src []byte) []byte {
	dst := make([]byte, len(src))
	copy(dst, src)
	return dst
}

// SecureBuffer that zeros itself when done
type SecureBuffer struct {
	data []byte
}

func NewSecureBuffer(size int) *SecureBuffer {
	sb := &SecureBuffer{
		data: make([]byte, size),
	}
	sb.EnableAutoZero()
	return sb
}

func (sb *SecureBuffer) Bytes() []byte {
	return sb.data
}

func (sb *SecureBuffer) Zero() {
	SecureZero(sb.data)
}

func (sb *SecureBuffer) Len() int {
	return len(sb.data)
}

func (sb *SecureBuffer) finalize() {
	sb.Zero()
}

func (sb *SecureBuffer) EnableAutoZero() {
	runtime.SetFinalizer(sb, (*SecureBuffer).finalize)
}

// Lazy initialization function
func getCryptoManager() *CryptoManager {
	cryptoInitOnce.Do(func() {
		globalCryptoManager = NewCryptoManager()
	})
	return globalCryptoManager
}

// Constructor for CryptoManager
func NewCryptoManager() *CryptoManager {
	return &CryptoManager{
		logger:   cfg.ChariotLogger,
		keyCache: make(map[string]*cachedKey),
	}
}

// Key management methods
func (cm *CryptoManager) getKeyFromVault(keyID string) ([]byte, error) {
	// Check cache first
	cm.cacheMutex.RLock()
	if cached, exists := cm.keyCache[keyID]; exists {
		if time.Now().Before(cached.expiresAt) {
			keyData := SecureCopy(cached.keyData)
			cm.cacheMutex.RUnlock()
			return keyData, nil
		}
		// Key expired, remove from cache
		SecureZero(cached.keyData)
		delete(cm.keyCache, keyID)
	}
	cm.cacheMutex.RUnlock()

	// Retrieve key material through the secret provider
	secretValue, err := vault.GetSecretValue(context.Background(), keyID)
	if err != nil {
		cm.logger.Error("Failed to retrieve key from secret provider", zap.String("key_id", keyID), zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve key %s: %v", keyID, err)
	}

	// Decode base64 key data
	keyData, err := base64.StdEncoding.DecodeString(secretValue)
	if err != nil {
		return nil, fmt.Errorf("failed to decode key %s: %v", keyID, err)
	}

	// Cache the key for 1 hour
	cm.cacheMutex.Lock()
	cm.keyCache[keyID] = &cachedKey{
		keyData:   SecureCopy(keyData),
		expiresAt: time.Now().Add(time.Hour),
	}
	cm.cacheMutex.Unlock()

	cm.logger.Info("Key retrieved from vault", zap.String("key_id", keyID))
	return keyData, nil
}

func (cm *CryptoManager) getRSAKeyFromVault(keyID string) (*rsa.PrivateKey, error) {
	// For RSA keys, they might be stored as PEM-encoded strings
	secretValue, err := vault.GetSecretValue(context.Background(), keyID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve RSA key %s: %v", keyID, err)
	}

	// Parse PEM-encoded private key
	block, _ := pem.Decode([]byte(secretValue))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block for key %s", keyID)
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS8 format
		parsedKey, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("failed to parse RSA key %s: %v (PKCS1: %v)", keyID, err2, err)
		}

		var ok bool
		privateKey, ok = parsedKey.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("key %s is not an RSA private key", keyID)
		}
	}

	return privateKey, nil
}

func (cm *CryptoManager) getRSAPublicKeyFromVault(keyID string) (*rsa.PublicKey, error) {
	// Try to get the private key first and extract public key
	privateKey, err := cm.getRSAKeyFromVault(keyID)
	if err == nil {
		return &privateKey.PublicKey, nil
	}

	// If that fails, try to get public key directly
	secretValue, err := vault.GetSecretValue(context.Background(), keyID+"-pub")
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve RSA public key %s: %v", keyID, err)
	}

	// Parse PEM-encoded public key
	block, _ := pem.Decode([]byte(secretValue))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block for public key %s", keyID)
	}

	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse RSA public key %s: %v", keyID, err)
	}

	rsaPublicKey, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("key %s is not an RSA public key", keyID)
	}

	return rsaPublicKey, nil
}

// AES encryption/decryption methods
func (cm *CryptoManager) EncryptWithKey(keyID string, data []byte) ([]byte, error) {
	keyData, err := cm.getKeyFromVault(keyID)
	if err != nil {
		return nil, err
	}
	defer SecureZero(keyData)

	return cm.EncryptAES(data, keyData)
}

func (cm *CryptoManager) DecryptWithKey(keyID string, ciphertext []byte) ([]byte, error) {
	keyData, err := cm.getKeyFromVault(keyID)
	if err != nil {
		return nil, err
	}
	defer SecureZero(keyData)

	return cm.DecryptAES(ciphertext, keyData)
}

func (cm *CryptoManager) EncryptAES(data []byte, key []byte) ([]byte, error) {
	// Validate key length (16, 24, or 32 bytes for AES-128, AES-192, or AES-256)
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, fmt.Errorf("invalid AES key length: %d bytes", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %v", err)
	}

	// Use GCM mode for authenticated encryption
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %v", err)
	}

	// Generate random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %v", err)
	}

	// Encrypt and authenticate
	ciphertext := gcm.Seal(nonce, nonce, data, nil)

	cm.logger.Info("AES encryption completed",
		zap.String("data_size", fmt.Sprintf("%d", len(data))),
		zap.String("ciphertext_size", fmt.Sprintf("%d", len(ciphertext))),
		zap.String("key_size", fmt.Sprintf("%d", len(key)*8)))

	return ciphertext, nil
}

func (cm *CryptoManager) DecryptAES(ciphertext []byte, key []byte) ([]byte, error) {
	// Validate key length
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, fmt.Errorf("invalid AES key length: %d bytes", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %v", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %v", err)
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}

	// Extract nonce and ciphertext
	nonce := ciphertext[:gcm.NonceSize()]
	ciphertext = ciphertext[gcm.NonceSize():]

	// Decrypt and authenticate
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("AES decryption failed: %v", err)
	}

	cm.logger.Info("AES decryption completed",
		zap.String("plaintext_size", fmt.Sprintf("%d", len(plaintext))),
		zap.String("key_size", fmt.Sprintf("%d", len(key)*8)))

	return plaintext, nil
}

// RSA signing/verification methods
func (cm *CryptoManager) SignWithKey(keyID string, data []byte) ([]byte, error) {
	privateKey, err := cm.getRSAKeyFromVault(keyID)
	if err != nil {
		return nil, err
	}

	return cm.SignRSA(data, privateKey)
}

func (cm *CryptoManager) VerifyWithKey(keyID string, data []byte, signature []byte) error {
	publicKey, err := cm.getRSAPublicKeyFromVault(keyID)
	if err != nil {
		return err
	}

	return cm.VerifyRSA(data, signature, publicKey)
}

func (cm *CryptoManager) SignRSA(data []byte, privateKey *rsa.PrivateKey) ([]byte, error) {
	// Hash the data
	hash := sha256.Sum256(data)

	// Sign the hash
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hash[:])
	if err != nil {
		return nil, fmt.Errorf("RSA signing failed: %v", err)
	}

	cm.logger.Info("RSA signature created",
		zap.String("data_size", fmt.Sprintf("%d", len(data))),
		zap.String("signature_size", fmt.Sprintf("%d", len(signature))),
		zap.String("key_size", fmt.Sprintf("%d", privateKey.Size()*8)))

	return signature, nil
}

func (cm *CryptoManager) VerifyRSA(data []byte, signature []byte, publicKey *rsa.PublicKey) error {
	// Hash the data
	hash := sha256.Sum256(data)

	// Verify the signature
	err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hash[:], signature)
	if err != nil {
		return fmt.Errorf("RSA signature verification failed: %v", err)
	}

	cm.logger.Info("RSA signature verified",
		zap.String("data_size", fmt.Sprintf("%d", len(data))),
		zap.String("signature_size", fmt.Sprintf("%d", len(signature))),
		zap.String("key_size", fmt.Sprintf("%d", publicKey.Size()*8)))

	return nil
}

// Key generation methods
func (cm *CryptoManager) GenerateRSAKeyPair(bits int) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	if bits < 2048 {
		return nil, nil, fmt.Errorf("RSA key size must be at least 2048 bits, got %d", bits)
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate RSA key pair: %v", err)
	}

	cm.logger.Info("RSA key pair generated", zap.String("key_size", fmt.Sprintf("%d", bits)))

	return privateKey, &privateKey.PublicKey, nil
}

func (cm *CryptoManager) GenerateAESKey(keySize int) ([]byte, error) {
	if keySize != 16 && keySize != 24 && keySize != 32 {
		return nil, fmt.Errorf("invalid AES key size: %d bytes (must be 16, 24, or 32)", keySize)
	}

	key := make([]byte, keySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate AES key: %v", err)
	}

	cm.logger.Info("AES key generated", zap.String("key_size", fmt.Sprintf("%d", keySize*8)))

	return key, nil
}

func (cm *CryptoManager) GenerateRandomBytes(size int) ([]byte, error) {
	if size <= 0 {
		return nil, fmt.Errorf("invalid size: %d", size)
	}

	bytes := make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %v", err)
	}

	return bytes, nil
}

// Hash functions
func (cm *CryptoManager) HashSHA256(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}

func (cm *CryptoManager) HashSHA512(data []byte) []byte {
	hash := sha512.Sum512(data)
	return hash[:]
}

// Key derivation
func (cm *CryptoManager) DeriveKeyPBKDF2(password string, salt []byte, iterations int, keyLen int) []byte {
	// Convert password to []byte immediately
	passwordBytes := []byte(password)
	defer SecureZero(passwordBytes) // Zero out the password bytes

	if salt == nil {
		// Generate random salt if none provided
		salt = make([]byte, 32)
		_, _ = rand.Read(salt)
	}

	if iterations < 10000 {
		iterations = 100000 // Minimum secure iterations
	}

	key := pbkdf2.Key(passwordBytes, salt, iterations, keyLen, sha256.New)

	cm.logger.Info("PBKDF2 key derivation completed",
		zap.Int("iterations", iterations),
		zap.Int("key_length", keyLen),
		zap.Int("salt_length", len(salt)))

	return key
}

// Utility methods
func (cm *CryptoManager) SecureCompare(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}

	var result byte
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}

	return result == 0
}

func (cm *CryptoManager) ClearKeyCache() {
	cm.cacheMutex.Lock()
	defer cm.cacheMutex.Unlock()

	for keyID, cached := range cm.keyCache {
		SecureZero(cached.keyData)
		delete(cm.keyCache, keyID)
	}

	cm.logger.Info("Key cache cleared")
}
