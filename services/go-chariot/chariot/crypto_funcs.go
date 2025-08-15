package chariot

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
)

// Registration function for the runtime
func RegisterCryptoFunctions(rt *Runtime) {
	crypto := getCryptoManager()
	crypto.RegisterCryptoFunctions(rt)
}

// Chariot function registration
func (cm *CryptoManager) RegisterCryptoFunctions(rt *Runtime) {
	// encrypt(keyID, data) - returns base64 encoded ciphertext
	rt.Register("encrypt", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("encrypt requires: keyID, data")
		}

		// Unwrap scope entries
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		keyID := string(args[0].(Str))
		dataStr := string(args[1].(Str))

		// Convert to []byte for crypto operations
		dataBytes := []byte(dataStr)
		defer SecureZero(dataBytes) // Zero out after use

		encrypted, err := cm.EncryptWithKey(keyID, dataBytes)
		if err != nil {
			return nil, fmt.Errorf("encryption failed: %v", err)
		}

		// Return as base64 string for Chariot
		return Str(base64.StdEncoding.EncodeToString(encrypted)), nil
	})

	// decrypt(keyID, encryptedData) - expects base64 encoded ciphertext
	rt.Register("decrypt", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("decrypt requires: keyID, encryptedData")
		}

		// Unwrap scope entries
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		keyID := string(args[0].(Str))
		encryptedStr := string(args[1].(Str))

		// Decode from base64
		encryptedBytes, err := base64.StdEncoding.DecodeString(encryptedStr)
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64: %v", err)
		}
		defer SecureZero(encryptedBytes) // Zero out after use

		decrypted, err := cm.DecryptWithKey(keyID, encryptedBytes)
		if err != nil {
			return nil, fmt.Errorf("decryption failed: %v", err)
		}
		defer SecureZero(decrypted) // Zero out after conversion

		// Convert back to string for Chariot
		return Str(string(decrypted)), nil
	})

	// sign(keyID, data) - returns base64 encoded signature
	rt.Register("sign", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("sign requires: keyID, data")
		}

		// Unwrap scope entries
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		keyID := string(args[0].(Str))
		dataStr := string(args[1].(Str))

		dataBytes := []byte(dataStr)
		defer SecureZero(dataBytes)

		signature, err := cm.SignWithKey(keyID, dataBytes)
		if err != nil {
			return nil, fmt.Errorf("signing failed: %v", err)
		}

		return Str(base64.StdEncoding.EncodeToString(signature)), nil
	})

	// verify(keyID, data, signature) - expects base64 encoded signature
	rt.Register("verify", func(args ...Value) (Value, error) {
		if len(args) != 3 {
			return nil, errors.New("verify requires: keyID, data, signature")
		}

		// Unwrap scope entries
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		keyID := string(args[0].(Str))
		dataStr := string(args[1].(Str))
		sigStr := string(args[2].(Str))

		dataBytes := []byte(dataStr)
		defer SecureZero(dataBytes)

		signature, err := base64.StdEncoding.DecodeString(sigStr)
		if err != nil {
			return nil, fmt.Errorf("failed to decode signature: %v", err)
		}
		defer SecureZero(signature)

		err = cm.VerifyWithKey(keyID, dataBytes, signature)
		return Bool(err == nil), nil
	})

	// hash256(data) - returns hex encoded hash
	rt.Register("hash256", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("hash256 requires: data")
		}

		// Unwrap scope entries
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}

		dataStr := string(args[0].(Str))
		dataBytes := []byte(dataStr)

		hash := cm.HashSHA256(dataBytes)
		return Str(fmt.Sprintf("%x", hash)), nil
	})

	// hash512(data) - SHA-512 hash
	rt.Register("hash512", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("hash512 requires: data")
		}

		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}

		dataStr := string(args[0].(Str))
		dataBytes := []byte(dataStr)

		hash := cm.HashSHA512(dataBytes)
		return Str(fmt.Sprintf("%x", hash)), nil
	})

	// generateKey(keySize) - generates AES key and returns base64 encoded
	rt.Register("generateKey", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("generateKey requires: keySize")
		}

		// Unwrap scope entries
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}

		keySize := int(args[0].(Number))

		key, err := cm.GenerateAESKey(keySize)
		if err != nil {
			return nil, fmt.Errorf("key generation failed: %v", err)
		}
		defer SecureZero(key) // Zero out after encoding

		return Str(base64.StdEncoding.EncodeToString(key)), nil
	})

	// generateRSAKey(bits) - Generate RSA key pair
	rt.Register("generateRSAKey", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("generateRSAKey requires: bits (minimum 2048)")
		}

		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}

		bits := int(args[0].(Number))
		if bits < 2048 {
			return nil, errors.New("RSA key size must be at least 2048 bits")
		}

		privateKey, publicKey, err := cm.GenerateRSAKeyPair(bits)
		if err != nil {
			return nil, fmt.Errorf("RSA key generation failed: %v", err)
		}

		// Return both keys as a JSON object
		result := make(map[string]Value)

		// Encode private key as PEM
		privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
		privateKeyPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: privateKeyBytes,
		})

		// Encode public key as PEM
		publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal public key: %v", err)
		}
		publicKeyPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: publicKeyBytes,
		})

		result["privateKey"] = Str(string(privateKeyPEM))
		result["publicKey"] = Str(string(publicKeyPEM))

		return result, nil
	})

	// randomBytes(size) - generates random bytes and returns base64 encoded
	rt.Register("randomBytes", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("randomBytes requires: size")
		}

		// Unwrap scope entries
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}

		size := int(args[0].(Number))

		bytes, err := cm.GenerateRandomBytes(size)
		if err != nil {
			return nil, fmt.Errorf("random bytes generation failed: %v", err)
		}

		return Str(base64.StdEncoding.EncodeToString(bytes)), nil
	})

	// encryptDirect(key, data) - encrypt with base64 key directly (no vault)
	rt.Register("encryptDirect", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("encryptDirect requires: key, data")
		}

		// Unwrap scope entries
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		keyStr := string(args[0].(Str))
		dataStr := string(args[1].(Str))

		// Decode base64 key
		keyBytes, err := base64.StdEncoding.DecodeString(keyStr)
		if err != nil {
			return nil, fmt.Errorf("failed to decode key: %v", err)
		}
		defer SecureZero(keyBytes)

		// Convert data to bytes
		dataBytes := []byte(dataStr)
		defer SecureZero(dataBytes)

		encrypted, err := cm.EncryptAES(dataBytes, keyBytes)
		if err != nil {
			return nil, fmt.Errorf("encryption failed: %v", err)
		}

		// Return as base64 string
		return Str(base64.StdEncoding.EncodeToString(encrypted)), nil
	})

	// decryptDirect(key, encryptedData) - decrypt with base64 key directly (no vault)
	rt.Register("decryptDirect", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("decryptDirect requires: key, encryptedData")
		}

		// Unwrap scope entries
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		keyStr := string(args[0].(Str))
		encryptedStr := string(args[1].(Str))

		// Decode base64 key
		keyBytes, err := base64.StdEncoding.DecodeString(keyStr)
		if err != nil {
			return nil, fmt.Errorf("failed to decode key: %v", err)
		}
		defer SecureZero(keyBytes)

		// Decode base64 encrypted data
		encryptedBytes, err := base64.StdEncoding.DecodeString(encryptedStr)
		if err != nil {
			return nil, fmt.Errorf("failed to decode encrypted data: %v", err)
		}
		defer SecureZero(encryptedBytes)

		decrypted, err := cm.DecryptAES(encryptedBytes, keyBytes)
		if err != nil {
			return nil, fmt.Errorf("decryption failed: %v", err)
		}
		defer SecureZero(decrypted)

		// Convert back to string
		return Str(string(decrypted)), nil
	})
}

// Global convenience functions
func GlobalEncryptWithKey(keyID string, data []byte) ([]byte, error) {
	return getCryptoManager().EncryptWithKey(keyID, data)
}

func GlobalDecryptWithKey(keyID string, ciphertext []byte) ([]byte, error) {
	return getCryptoManager().DecryptWithKey(keyID, ciphertext)
}

func GlobalSignWithKey(keyID string, data []byte) ([]byte, error) {
	return getCryptoManager().SignWithKey(keyID, data)
}

func GlobalVerifyWithKey(keyID string, data []byte, signature []byte) error {
	return getCryptoManager().VerifyWithKey(keyID, data, signature)
}

func GlobalHashSHA256(data []byte) []byte {
	return getCryptoManager().HashSHA256(data)
}

func GlobalDeriveKeyPBKDF2(password string, salt []byte, iterations int, keyLen int) []byte {
	return getCryptoManager().DeriveKeyPBKDF2(password, salt, iterations, keyLen)
}
