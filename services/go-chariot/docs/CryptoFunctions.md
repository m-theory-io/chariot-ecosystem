# Chariot Language Reference

## Crypto Functions

Chariot provides built-in cryptographic functions for encryption, decryption, digital signatures, hashing, secure key generation, and RSA keypair management. All cryptographic operations use secure key management (e.g., Azure Key Vault) and never expose raw key material in Chariot code.

### Available Crypto Functions

| Function                        | Description                                                      |
|----------------------------------|------------------------------------------------------------------|
| `encrypt(keyID, data)`           | Encrypt data with the given key ID; returns base64 ciphertext    |
| `decrypt(keyID, ciphertext)`     | Decrypt base64 ciphertext with the given key ID; returns plaintext|
| `sign(keyID, data)`              | Sign data with the given key ID; returns base64 signature        |
| `verify(keyID, data, signature)` | Verify base64 signature with the given key ID; returns `true` or `false` |
| `hash256(data)`                  | Compute SHA-256 hash of data; returns hex string                 |
| `hash512(data)`                  | Compute SHA-512 hash of data; returns hex string                 |
| `generateKey(keySize)`           | Generate a random AES key of given size (bytes); returns base64  |
| `generateRSAKey(bits)`           | Generate an RSA keypair (min 2048 bits); returns PEM strings     |
| `randomBytes(size)`              | Generate random bytes of given size; returns base64              |

---

### Function Details

#### `encrypt(keyID, data)`

Encrypts the given string data using the key referenced by `keyID`. Returns a base64-encoded ciphertext string.

```chariot
setq(cipher, encrypt('my-key-id', 'Sensitive information'))
```

#### `decrypt(keyID, ciphertext)`

Decrypts the base64-encoded ciphertext using the key referenced by `keyID`. Returns the plaintext string.

```chariot
setq(plain, decrypt('my-key-id', cipher))
```

#### `sign(keyID, data)`

Signs the given string data using the key referenced by `keyID`. Returns a base64-encoded signature.

```chariot
setq(sig, sign('my-signing-key', 'Document to sign'))
```

#### `verify(keyID, data, signature)`

Verifies the base64-encoded signature for the given data using the key referenced by `keyID`. Returns `true` if valid, `false` otherwise.

```chariot
setq(isValid, verify('my-signing-key', 'Document to sign', sig))
```

#### `hash256(data)`

Computes the SHA-256 hash of the input string. Returns a hex-encoded string.

```chariot
setq(hash, hash256('data to hash'))
```

#### `hash512(data)`

Computes the SHA-512 hash of the input string. Returns a hex-encoded string.

```chariot
setq(hash, hash512('data to hash'))
```

#### `generateKey(keySize)`

Generates a random AES key of the specified size in bytes (16, 24, or 32 for AES-128/192/256). Returns a base64-encoded key.

```chariot
setq(aesKey, generateKey(32))  // 256-bit AES key
```

#### `generateRSAKey(bits)`

Generates an RSA keypair of the specified size (minimum 2048 bits). Returns a map with PEM-encoded `privateKey` and `publicKey` strings.

```chariot
setq(keys, generateRSAKey(2048))
print(keys.privateKey)
print(keys.publicKey)
```

#### `randomBytes(size)`

Generates a random byte sequence of the specified size. Returns a base64-encoded string.

```chariot
setq(token, randomBytes(16))
```

---

### Notes

- All cryptographic operations use secure key management; keys are referenced by ID only.
- All encryption and signature outputs are base64-encoded strings.
- Hash outputs are hex-encoded strings.
- `generateKey` and `randomBytes` are suitable for generating secrets, tokens, or initialization vectors.
- `generateRSAKey` returns PEM-encoded keys as a map: `{ "privateKey": ..., "publicKey": ... }`.