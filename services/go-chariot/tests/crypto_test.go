package tests

import (
	"testing"

	"github.com/bhouse1273/go-chariot/chariot"
)

// tests/crypto_test.go
func TestCryptoOperations(t *testing.T) {
	tests := []TestCase{
		{
			Name: "SHA256 Hash",
			Script: []string{
				`setq(data, "Hello, World!")`,
				`setq(hash, hash256(data))`,
				`hash`,
			},
			ExpectedType:  "chariot.Str",
			ExpectedValue: chariot.Str("dffd6021bb2bd5b0af676290809ec3a53191dd81c7f70a4b28688a362182986f"),
		},
		{
			Name: "SHA512 Hash",
			Script: []string{
				`setq(data, "test data")`,
				`hash512(data)`,
			},
			ExpectedType:  "chariot.Str",
			ExpectedValue: chariot.Str("0e1e21ecf105ec853d24d728867ad70613c21663a4693074b2a3619c1bd39d66b588c33723bb466c72424e80e3ca63c249078ab347bab9428500e7ee43059d0d"),
		},
		{
			Name: "AES Encryption/Decryption with Direct Functions",
			Script: []string{
				`setq(key, generateKey(32))`,
				`setq(encrypted, encryptDirect(key, "hello world"))`,
				`decryptDirect(key, encrypted)`,
			},
			ExpectedValue: chariot.Str("hello world"),
		},
		{
			Name: "Multiple Round-trip Encryption",
			Script: []string{
				`setq(key, generateKey(32))`,
				`setq(message, "This is a longer test message with special chars: !@#$%^&*()")`,
				`setq(encrypted, encryptDirect(key, message))`,
				`decryptDirect(key, encrypted)`,
			},
			ExpectedValue: chariot.Str("This is a longer test message with special chars: !@#$%^&*()"),
		},
	}

	RunTestCases(t, tests)
}
