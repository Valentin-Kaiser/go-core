package security

import (
	"bytes"
	"crypto/tls"
	"crypto/x509/pkix"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ProtonMail/gopenpgp/v3/profile"
)

func TestGetRandomBytes(t *testing.T) {
	testCases := []uint{1, 16, 32, 64, 128}

	for _, size := range testCases {
		bytes, err := GetRandomBytes(size)
		if err != nil {
			t.Errorf("GetRandomBytes(%d) returned error: %v", size, err)
			continue
		}

		if len(bytes) != int(size) {
			t.Errorf("GetRandomBytes(%d) returned %d bytes, expected %d", size, len(bytes), size)
		}

		// Check that bytes are not all zeros (very unlikely with crypto/rand)
		allZeros := true
		for _, b := range bytes {
			if b != 0 {
				allZeros = false
				break
			}
		}
		if allZeros && size > 0 {
			t.Errorf("GetRandomBytes(%d) returned all zeros, unlikely for crypto/rand", size)
		}
	}
}

func TestGetRandomBytesZero(t *testing.T) {
	bytes, err := GetRandomBytes(0)
	if err != nil {
		t.Errorf("GetRandomBytes(0) returned error: %v", err)
	}
	if len(bytes) != 0 {
		t.Errorf("GetRandomBytes(0) returned %d bytes, expected 0", len(bytes))
	}
}

func TestSHA256(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		{"hello", "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"},
		{"Hello, World!", "dffd6021bb2bd5b0af676290809ec3a53191dd81c7f70a4b28688a362182986f"},
		{"The quick brown fox jumps over the lazy dog", "d7a8fbb307d7809469ca9abcb0082e4f8d5651e46d3cdb762d02d0bf37c9e592"},
	}

	for _, tc := range testCases {
		result := SHA256([]byte(tc.input))
		if result != tc.expected {
			t.Errorf("SHA256(%q) = %s, expected %s", tc.input, result, tc.expected)
		}
	}
}

func TestXX(t *testing.T) {
	testCases := []struct {
		input string
	}{
		{""},
		{"hello"},
		{"Hello, World!"},
		{"The quick brown fox jumps over the lazy dog"},
	}

	for _, tc := range testCases {
		result := XX([]byte(tc.input))
		// We can't predict the exact hash, but it should be consistent
		result2 := XX([]byte(tc.input))
		if result != result2 {
			t.Errorf("XX(%q) not consistent: %d != %d", tc.input, result, result2)
		}
	}
}

func TestMd5(t *testing.T) {
	testCases := []struct {
		input    interface{}
		expected string
	}{
		{"", "d41d8cd98f00b204e9800998ecf8427e"},
		{"hello", "5d41402abc4b2a76b9719d911017c592"},
		{"Hello, World!", "65a8e27d8879283831b664bd8b7f0ad4"},
		{42, "a1d0c6e83f027327d8461063f4ac58a6"},
		{true, "b326b5062b2f0e69046810717534cb09"},
	}

	for _, tc := range testCases {
		result := Md5(tc.input)
		if result != tc.expected {
			t.Errorf("Md5(%v) = %s, expected %s", tc.input, result, tc.expected)
		}
	}
}

func TestStringToKey32Bytes(t *testing.T) {
	testCases := []string{
		"",
		"password",
		"secret",
		"very long passphrase with many characters",
	}

	for _, tc := range testCases {
		key := StringToKey32Bytes(tc)
		if len(key) != 32 {
			t.Errorf("StringToKey32Bytes(%q) returned %d bytes, expected 32", tc, len(key))
		}

		// Test consistency
		key2 := StringToKey32Bytes(tc)
		if !bytes.Equal(key, key2) {
			t.Errorf("StringToKey32Bytes(%q) not consistent", tc)
		}
	}
}

func TestStringToKey32(t *testing.T) {
	testCases := []string{
		"",
		"password",
		"secret",
		"very long passphrase with many characters",
	}

	for _, tc := range testCases {
		key := StringToKey32(tc)
		if len(key) != 32 {
			t.Errorf("StringToKey32(%q) returned %d characters, expected 32", tc, len(key))
		}

		// Test consistency
		key2 := StringToKey32(tc)
		if key != key2 {
			t.Errorf("StringToKey32(%q) not consistent", tc)
		}

		// Test that it's hex encoded
		for _, c := range key {
			if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
				t.Errorf("StringToKey32(%q) contains non-hex character: %c", tc, c)
				break
			}
		}
	}
}

func TestGetRandomBytesBase64(t *testing.T) {
	testCases := []uint{1, 16, 32, 64}

	for _, size := range testCases {
		encoded, err := GetRandomBytesBase64(size)
		if err != nil {
			t.Errorf("GetRandomBytesBase64(%d) returned error: %v", size, err)
			continue
		}

		// Base64 encoding should produce a string
		if len(encoded) == 0 && size > 0 {
			t.Errorf("GetRandomBytesBase64(%d) returned empty string", size)
		}

		// Test that it's valid base64
		if size > 0 {
			for _, c := range encoded {
				if (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') &&
					(c < '0' || c > '9') && c != '+' && c != '/' && c != '=' {
					t.Errorf("GetRandomBytesBase64(%d) contains invalid base64 character: %c", size, c)
					break
				}
			}
		}
	}
}

func TestGenerateRandomPassword(t *testing.T) {
	testCases := []int{1, 8, 16, 32, 64}

	for _, length := range testCases {
		password, err := GenerateRandomPassword(length)
		if err != nil {
			t.Errorf("GenerateRandomPassword(%d) returned error: %v", length, err)
			continue
		}

		if len(password) != length {
			t.Errorf("GenerateRandomPassword(%d) returned password of length %d, expected %d", length, len(password), length)
		}

		// Test that password contains only valid characters
		validChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789.-#!+/"
		for _, c := range password {
			if !strings.ContainsRune(validChars, c) {
				t.Errorf("GenerateRandomPassword(%d) contains invalid character: %c", length, c)
				break
			}
		}
	}
}

func TestGenerateRandomPasswordZero(t *testing.T) {
	password, err := GenerateRandomPassword(0)
	if err != nil {
		t.Errorf("GenerateRandomPassword(0) returned error: %v", err)
	}
	if len(password) != 0 {
		t.Errorf("GenerateRandomPassword(0) returned password of length %d, expected 0", len(password))
	}
}

func TestReadOrSavePassphrase(t *testing.T) {
	// Create a temporary directory for test
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_passphrase.txt")

	// Test creating new passphrase
	passphrase1, err := ReadOrSavePassphrase(testFile, 32)
	if err != nil {
		t.Errorf("ReadOrSavePassphrase() returned error: %v", err)
	}

	if len(passphrase1) != 32 {
		t.Errorf("ReadOrSavePassphrase() returned passphrase of length %d, expected 32", len(passphrase1))
	}

	// Test reading existing passphrase
	passphrase2, err := ReadOrSavePassphrase(testFile, 32)
	if err != nil {
		t.Errorf("ReadOrSavePassphrase() returned error when reading existing file: %v", err)
	}

	if !bytes.Equal(passphrase1, passphrase2) {
		t.Error("ReadOrSavePassphrase() returned different passphrase when reading existing file")
	}

	// Test that file exists and has correct content
	content, err := os.ReadFile(filepath.Clean(testFile))
	if err != nil {
		t.Errorf("Failed to read test file: %v", err)
	}

	if !bytes.Equal(content[:32], passphrase1) {
		t.Error("File content doesn't match returned passphrase")
	}
}

func TestReadOrSavePassphraseTooShort(t *testing.T) {
	// Create a temporary file with short content
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "short_passphrase.txt")

	err := os.WriteFile(testFile, []byte("short"), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test reading passphrase that's too short
	_, err = ReadOrSavePassphrase(testFile, 32)
	if err == nil {
		t.Error("ReadOrSavePassphrase() should return error for too short passphrase")
	}
}

func TestReadOrSavePassphraseFilePermissions(t *testing.T) {
	// Create a temporary directory for test
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "perm_test.txt")

	// Create passphrase file
	_, err := ReadOrSavePassphrase(testFile, 16)
	if err != nil {
		t.Errorf("ReadOrSavePassphrase() returned error: %v", err)
	}

	// Check file permissions
	info, err := os.Stat(testFile)
	if err != nil {
		t.Errorf("Failed to stat test file: %v", err)
	}

	if runtime.GOOS != "windows" {
		mode := info.Mode()
		if mode.Perm() != 0600 {
			t.Errorf("File permissions are %o, expected 0600", mode.Perm())
		}
	}
}

func TestStringToKey32BytesEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"single character", "a"},
		{"long string", strings.Repeat("test", 1000)},
		{"unicode string", "测试中文字符串🚀"},
		{"special characters", "!@#$%^&*()_+-=[]{}|;':\",./<>?"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StringToKey32Bytes(tt.input)
			if len(result) != 32 {
				t.Errorf("StringToKey32Bytes() returned %d bytes, expected 32", len(result))
			}

			// Test consistency
			result2 := StringToKey32Bytes(tt.input)
			if !bytes.Equal(result, result2) {
				t.Error("StringToKey32Bytes() should return consistent results")
			}
		})
	}
}

func TestStringToKey32EdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"single character", "a"},
		{"long string", strings.Repeat("test", 1000)},
		{"unicode string", "测试中文字符串🚀"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StringToKey32(tt.input)
			if len(result) != 32 {
				t.Errorf("StringToKey32() returned %d characters, expected 32", len(result))
			}

			// Test consistency
			result2 := StringToKey32(tt.input)
			if result != result2 {
				t.Error("StringToKey32() should return consistent results")
			}
		})
	}
}

func TestSHA256EdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
	}{
		{"empty bytes", []byte{}},
		{"single byte", []byte{0}},
		{"null bytes", []byte{0, 0, 0, 0}},
		{"large input", bytes.Repeat([]byte("test"), 10000)},
		{"binary data", []byte{0xFF, 0xFE, 0xFD, 0xFC}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SHA256(tt.input)
			if len(result) != 64 { // SHA256 hex string is 64 characters
				t.Errorf("SHA256() returned %d characters, expected 64", len(result))
			}

			// Test consistency
			result2 := SHA256(tt.input)
			if result != result2 {
				t.Error("SHA256() should return consistent results")
			}
		})
	}
}

func TestXXHashEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
	}{
		{"empty bytes", []byte{}},
		{"single byte", []byte{0}},
		{"large input", bytes.Repeat([]byte("test"), 10000)},
		{"binary data", []byte{0xFF, 0xFE, 0xFD, 0xFC}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := XX(tt.input)

			// Test consistency
			result2 := XX(tt.input)
			if result != result2 {
				t.Error("XX() should return consistent results")
			}
		})
	}
}

func TestMd5EdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
	}{
		{"nil", nil},
		{"empty string", ""},
		{"number", 42},
		{"float", 3.14159},
		{"bool", true},
		{"slice", []int{1, 2, 3}},
		{"struct", struct{ Name string }{"test"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Md5(tt.input)
			if len(result) != 32 { // MD5 hex string is 32 characters
				t.Errorf("Md5() returned %d characters, expected 32", len(result))
			}

			// Test consistency
			result2 := Md5(tt.input)
			if result != result2 {
				t.Error("Md5() should return consistent results")
			}
		})
	}
}

func TestGenerateRandomPasswordEdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		length int
		valid  bool
	}{
		{"zero length", 0, true},
		{"small length", 1, true},
		{"normal length", 16, true},
		{"large length", 1000, true},
		{"negative length", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GenerateRandomPassword(tt.length)

			if tt.valid {
				if err != nil {
					t.Errorf("GenerateRandomPassword() should succeed for length %d: %v", tt.length, err)
				}
				if len(result) != tt.length {
					t.Errorf("GenerateRandomPassword() returned %d characters, expected %d", len(result), tt.length)
				}
			} else {
				// For negative lengths, the behavior depends on the implementation
				// It might fail or return empty string
				_ = result
				_ = err
			}
		})
	}
}

func TestReadOrSavePassphraseFileOperations(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		filename string
		length   int
		setup    func(string) error
		wantErr  bool
	}{
		{
			name:     "new file",
			filename: "new_passphrase.txt",
			length:   32,
			setup:    nil,
			wantErr:  false,
		},
		{
			name:     "existing file with sufficient length",
			filename: "existing_passphrase.txt",
			length:   16,
			setup: func(path string) error {
				return os.WriteFile(path, []byte("this_is_a_long_enough_passphrase"), 0600)
			},
			wantErr: false,
		},
		{
			name:     "existing file with insufficient length",
			filename: "short_passphrase.txt",
			length:   32,
			setup: func(path string) error {
				return os.WriteFile(path, []byte("short"), 0600)
			},
			wantErr: true,
		},
	}

	// Add permission test only for non-Windows systems
	if runtime.GOOS != "windows" {
		tests = append(tests, struct {
			name     string
			filename string
			length   int
			setup    func(string) error
			wantErr  bool
		}{
			name:     "permission denied directory",
			filename: "readonly/passphrase.txt",
			length:   16,
			setup: func(path string) error {
				dir := filepath.Dir(path)
				if err := os.MkdirAll(dir, 0750); err != nil {
					return err
				}
				// Remove all permissions on Unix-like systems
				return os.Chmod(dir, 0000)
			},
			wantErr: true,
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tempDir, tt.filename)

			if tt.setup != nil {
				if err := tt.setup(filePath); err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			}

			result, err := ReadOrSavePassphrase(filePath, tt.length)

			if tt.wantErr {
				if err == nil {
					t.Error("ReadOrSavePassphrase() should have failed")
				}
			} else {
				if err != nil {
					t.Errorf("ReadOrSavePassphrase() failed: %v", err)
				}
				if len(result) != tt.length {
					t.Errorf("ReadOrSavePassphrase() returned %d bytes, expected %d", len(result), tt.length)
				}
			}

			// Cleanup permission denied test
			if strings.Contains(tt.filename, "readonly") {
				dir := filepath.Dir(filePath)
				// Restore permissions before cleanup
				if err := os.Chmod(dir, 0600); err != nil {
					t.Logf("Failed to restore permissions: %v", err)
				}
			}
		})
	}

	// Separate Windows-specific permission test using a different approach
	if runtime.GOOS == "windows" {
		t.Run("windows permission test - invalid path", func(t *testing.T) {
			// On Windows, test with an invalid path that should cause an error
			invalidPath := filepath.Join("Z:\\nonexistent\\path\\that\\should\\not\\exist", "passphrase.txt")
			_, err := ReadOrSavePassphrase(invalidPath, 16)
			if err == nil {
				t.Error("ReadOrSavePassphrase() should fail with invalid path on Windows")
			}
		})
	}
}

func TestPGPCipherCreation(t *testing.T) {
	// Test with default profile
	cipher1 := NewPGPCipher(nil)
	if cipher1 == nil {
		t.Error("NewPGPCipher() should not return nil")
	}
	if cipher1 != nil && cipher1.handle == nil {
		t.Error("PGPCipher should have a handle")
	}

	// Test with custom profile
	customProfile := profile.RFC4880()
	cipher2 := NewPGPCipher(customProfile)
	if cipher2 == nil {
		t.Error("NewPGPCipher() should not return nil with custom profile")
	}
	if cipher2 != nil && cipher2.handle == nil {
		t.Error("PGPCipher should have a handle with custom profile")
	}
}

func TestPGPCipherWithoutKeysOrPassphrase(t *testing.T) {
	cipher := NewPGPCipher(nil)
	var buf bytes.Buffer

	// Test encryption without keys or passphrase
	cipher.Encrypt([]byte("test message"), &buf)
	if cipher.Error == nil {
		t.Error("Encrypt() should fail without keys or passphrase")
	}

	// Reset error and test decryption
	cipher.Error = nil
	buf.Reset()
	cipher.Decrypt([]byte("encrypted data"), &buf)
	if cipher.Error == nil {
		t.Error("Decrypt() should fail without keys or passphrase")
	}
}

func TestPGPCipherEncryptWithPasswordEdgeCases(t *testing.T) {
	cipher := NewPGPCipher(nil)
	var buf bytes.Buffer

	tests := []struct {
		name       string
		passphrase []byte
		plaintext  []byte
		writer     io.Writer
		wantErr    bool
	}{
		{
			name:       "empty plaintext",
			passphrase: []byte("test_passphrase"),
			plaintext:  []byte{},
			writer:     &buf,
			wantErr:    true,
		},
		{
			name:       "nil writer",
			passphrase: []byte("test_passphrase"),
			plaintext:  []byte("test message"),
			writer:     nil,
			wantErr:    true,
		},
		{
			name:       "empty passphrase",
			passphrase: []byte{},
			plaintext:  []byte("test message"),
			writer:     &buf,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cipher.Error = nil
			cipher.passphrase = tt.passphrase
			buf.Reset()

			cipher.EncryptWithPassword(tt.plaintext, tt.writer)

			if tt.wantErr {
				if cipher.Error == nil {
					t.Error("EncryptWithPassword() should have failed")
				}
			} else {
				if cipher.Error != nil {
					t.Errorf("EncryptWithPassword() failed: %v", cipher.Error)
				}
			}
		})
	}
}

func TestPGPCipherEncryptWithPublicKeyEdgeCases(t *testing.T) {
	cipher := NewPGPCipher(nil)
	var buf bytes.Buffer

	tests := []struct {
		name      string
		publicKey string
		plaintext []byte
		writer    io.Writer
		wantErr   bool
	}{
		{
			name:      "empty plaintext",
			publicKey: "fake_public_key",
			plaintext: []byte{},
			writer:    &buf,
			wantErr:   true,
		},
		{
			name:      "nil writer",
			publicKey: "fake_public_key",
			plaintext: []byte("test message"),
			writer:    nil,
			wantErr:   true,
		},
		{
			name:      "empty public key",
			publicKey: "",
			plaintext: []byte("test message"),
			writer:    &buf,
			wantErr:   true,
		},
		{
			name:      "invalid public key",
			publicKey: "invalid_key_format",
			plaintext: []byte("test message"),
			writer:    &buf,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cipher.Error = nil
			cipher.publicKey = tt.publicKey
			buf.Reset()

			cipher.EncryptWithPublicKey(tt.plaintext, tt.writer)

			if tt.wantErr {
				if cipher.Error == nil {
					t.Error("EncryptWithPublicKey() should have failed")
				}
			}
		})
	}
}

// Test TLS functions
func TestNewTLSConfig(t *testing.T) {
	// Generate a self-signed certificate for testing
	subject := pkix.Name{
		Organization:  []string{"Test"},
		Country:       []string{"US"},
		Province:      []string{""},
		Locality:      []string{"Test City"},
		StreetAddress: []string{""},
		PostalCode:    []string{""},
	}

	cert, caPool, err := GenerateSelfSignedCertificate(subject)
	if err != nil {
		t.Fatalf("Failed to generate test certificate: %v", err)
	}

	config := NewTLSConfig(cert, caPool, tls.RequireAndVerifyClientCert)

	if config == nil {
		t.Error("NewTLSConfig() should not return nil")
		return
	}

	if config.MinVersion != tls.VersionTLS12 {
		t.Errorf("Expected MinVersion to be TLS 1.2, got %d", config.MinVersion)
	}

	if len(config.Certificates) != 1 {
		t.Errorf("Expected 1 certificate, got %d", len(config.Certificates))
	}

	if config.ClientCAs != caPool {
		t.Error("ClientCAs should be set to the provided CA pool")
	}

	if config.ClientAuth != tls.RequireAndVerifyClientCert {
		t.Error("ClientAuth should be set correctly")
	}
}

func TestGenerateSelfSignedCertificate(t *testing.T) {
	subject := pkix.Name{
		Organization:  []string{"Test Org"},
		Country:       []string{"US"},
		Province:      []string{"CA"},
		Locality:      []string{"San Francisco"},
		StreetAddress: []string{"123 Test St"},
		PostalCode:    []string{"12345"},
	}

	cert, caPool, err := GenerateSelfSignedCertificate(subject)
	if err != nil {
		t.Fatalf("GenerateSelfSignedCertificate() failed: %v", err)
	}

	if len(cert.Certificate) == 0 {
		t.Error("Generated certificate should not be empty")
	}

	if caPool == nil {
		t.Error("CA pool should not be nil")
	}

	// Validate the generated certificate
	err = ValidateCertificate(cert)
	if err != nil {
		t.Errorf("Generated certificate should be valid: %v", err)
	}

	// Check if certificate is not expired
	expired, err := IsCertificateExpired(cert)
	if err != nil {
		t.Errorf("Error checking certificate expiration: %v", err)
	}
	if expired {
		t.Error("Newly generated certificate should not be expired")
	}
}

func TestValidateCertificate(t *testing.T) {
	// Test with empty certificate
	emptyCert := tls.Certificate{}
	err := ValidateCertificate(emptyCert)
	if err == nil {
		t.Error("ValidateCertificate() should fail with empty certificate")
	}

	// Test with valid certificate
	subject := pkix.Name{Organization: []string{"Test"}}
	cert, _, err := GenerateSelfSignedCertificate(subject)
	if err != nil {
		t.Fatalf("Failed to generate test certificate: %v", err)
	}

	err = ValidateCertificate(cert)
	if err != nil {
		t.Errorf("ValidateCertificate() should pass with valid certificate: %v", err)
	}
}

func TestIsCertificateExpired(t *testing.T) {
	// Test with empty certificate
	emptyCert := tls.Certificate{}
	_, err := IsCertificateExpired(emptyCert)
	if err == nil {
		t.Error("IsCertificateExpired() should fail with empty certificate")
	}

	// Test with valid certificate
	subject := pkix.Name{Organization: []string{"Test"}}
	cert, _, err := GenerateSelfSignedCertificate(subject)
	if err != nil {
		t.Fatalf("Failed to generate test certificate: %v", err)
	}

	expired, err := IsCertificateExpired(cert)
	if err != nil {
		t.Errorf("IsCertificateExpired() failed: %v", err)
	}
	if expired {
		t.Error("Newly generated certificate should not be expired")
	}
}

func TestLoadCertAndConfig(t *testing.T) {
	// This test will fail because we don't have actual cert files
	// but it tests the error handling path
	_, err := LoadCertAndConfig("nonexistent.crt", "nonexistent.key", "nonexistent.ca", tls.NoClientCert)
	if err == nil {
		t.Error("LoadCertAndConfig() should fail with nonexistent files")
	}
}

func TestLoadCertificate(t *testing.T) {
	// Test with nonexistent files
	_, err := LoadCertificate("nonexistent.crt", "nonexistent.key")
	if err == nil {
		t.Error("LoadCertificate() should fail with nonexistent files")
	}
}

func TestLoadCACertPool(t *testing.T) {
	// Test with nonexistent file
	_, err := LoadCACertPool("nonexistent.ca")
	if err == nil {
		t.Error("LoadCACertPool() should fail with nonexistent file")
	}

	// Test with invalid CA file
	tempDir := t.TempDir()
	invalidCAFile := filepath.Join(tempDir, "invalid.ca")
	err = os.WriteFile(invalidCAFile, []byte("invalid certificate data"), 0600)
	if err != nil {
		t.Fatalf("Failed to create invalid CA file: %v", err)
	}

	_, err = LoadCACertPool(invalidCAFile)
	if err == nil {
		t.Error("LoadCACertPool() should fail with invalid certificate data")
	}
}

// Benchmark tests for security functions
func BenchmarkSHA256(b *testing.B) {
	data := []byte("benchmark test data for SHA256 hashing function")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		SHA256(data)
	}
}

func BenchmarkXX(b *testing.B) {
	data := []byte("benchmark test data for xxHash function")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		XX(data)
	}
}

func BenchmarkMd5(b *testing.B) {
	data := "benchmark test data for MD5 hashing function"
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		Md5(data)
	}
}

func BenchmarkGetRandomBytes(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := GetRandomBytes(32); err != nil {
			b.Logf("Failed to get random bytes: %v", err)
		}
	}
}

func BenchmarkGenerateRandomPassword(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := GenerateRandomPassword(16); err != nil {
			b.Logf("Failed to generate password: %v", err)
		}
	}
}

func BenchmarkStringToKey32(b *testing.B) {
	input := "benchmark test string for key derivation"
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		StringToKey32(input)
	}
}
